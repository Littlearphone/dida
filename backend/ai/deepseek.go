package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"dida/models"
)

// DeepSeekClient DeepSeek API 客户端
type DeepSeekClient struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
}

// NewDeepSeekClient 创建 DeepSeek API 客户端
func NewDeepSeekClient(endpoint, apiKey, model string) *DeepSeekClient {
	if endpoint == "" {
		endpoint = "https://api.deepseek.com"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	return &DeepSeekClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		client: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

// apiURL 根据 endpoint 自动推断兼容的 API 路径
func (c *DeepSeekClient) apiURL() string {
	e := strings.TrimRight(c.endpoint, "/")
	if strings.Contains(e, "/chat/completions") {
		return e
	}
	if strings.Contains(e, "openrouter.ai") {
		return e + "/api/v1/chat/completions"
	}
	return e + "/v1/chat/completions"
}

// Chat 发送聊天请求并获取完整响应
func (c *DeepSeekClient) Chat(messages []models.Message, temperature *float32) (string, error) {
	req := models.AIRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}
	if temperature != nil {
		req.Temperature = temperature
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	msgCount := len(messages)
	totalChars := 0
	for _, m := range messages {
		totalChars += len([]rune(m.Content))
	}
	estimatedTokens := totalChars / 2
	log.Printf("[AI] 请求 model=%s | endpoint=%s | 消息数=%d | 输入字符数=%d | 估算token≈%d",
		c.model, c.endpoint, msgCount, totalChars, estimatedTokens)

	apiURL := c.apiURL()
	log.Printf("[AI] 发送请求 → %s", apiURL)

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	start := time.Now()
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	elapsed := time.Since(start)
	log.Printf("[AI] 响应耗时=%v | 状态码=%d | 开始读取响应体...", elapsed, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AI] 错误响应: %s", string(respBody))
		return "", fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(respBody))
	}

	var aiResp models.AIResponse
	if err := json.Unmarshal(respBody, &aiResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("API响应中没有choices")
	}

	responseContent := aiResp.Choices[0].Message.Content
	log.Printf("[AI] 响应完成 | 输出字符数=%d | 耗时=%v",
		len([]rune(responseContent)), time.Since(start))

	return responseContent, nil
}

// StreamChat 发送流式聊天请求，通过 onChunk 回调逐块返回内容增量
// 返回完整累积文本，适合需要实时展示生成进度的场景
func (c *DeepSeekClient) StreamChat(messages []models.Message, temperature *float32, onChunk func(string)) (string, error) {
	req := models.AIRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}
	if temperature != nil {
		req.Temperature = temperature
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	msgCount := len(messages)
	totalChars := 0
	for _, m := range messages {
		totalChars += len([]rune(m.Content))
	}
	log.Printf("[AI] 流式请求 model=%s | endpoint=%s | 消息数=%d | 输入字符数=%d",
		c.model, c.endpoint, msgCount, totalChars)

	apiURL := c.apiURL()
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	start := time.Now()
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(respBody))
	}

	var fullText strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	// 使用大缓冲区处理长行
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)
		if data == "[DONE]" {
			break
		}

		// 解析 OpenAI 兼容的 SSE 数据块
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 跳过无法解析的块（如纯文本非JSON行）
			continue
		}
		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				fullText.WriteString(content)
				onChunk(content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[AI] 流式读取错误: %v", err)
		return fullText.String(), fmt.Errorf("读取流失败: %w", err)
	}

	log.Printf("[AI] 流式响应完成 | 输出字符数=%d | 耗时=%v",
		len([]rune(fullText.String())), time.Since(start))
	return fullText.String(), nil
}

// SplitChapters 使用AI拆分小说章节（AI仅标注行号范围，后端从原文切分）
func (c *DeepSeekClient) SplitChapters(content string) (*models.SplitResult, error) {
	log.Printf("[AI] 开始拆分章节 | 内容长度=%d 字符", len([]rune(content)))

	// 按行拆分为段落，空行跳过
	rawLines := strings.Split(content, "\n")
	paras := make([]string, 0, len(rawLines))
	for _, l := range rawLines {
		if trimmed := strings.TrimSpace(l); trimmed != "" {
			paras = append(paras, trimmed)
		}
	}
	paraCount := len(paras)
	log.Printf("[AI] 段落数=%d", paraCount)

	// 构建带行号的内容（AI只读一次，返回行号即可，不必输出全文）
	numbered := make([]string, paraCount)
	for i, p := range paras {
		numbered[i] = fmt.Sprintf("[%d] %s", i+1, p)
	}
	numberedContent := strings.Join(numbered, "\n")

	const promptTemplate = `你是一个专业的小说编辑助手。请分析下方带行号的小说内容，按章节拆分并提取关键信息。

要求：
1. 识别小说标题、作者和简介
2. 用行号标注每章的起止行（startPara/endPara），后端据此切分正文
3. 为每章取精炼且有吸引力的标题
4. 提取主要角色（角色名、性格特征、描述）
5. 提取角色之间的关系（谁和谁是什么关系）
	6. 生成故事大纲

JSON格式（不要包含原文正文）：
{
  "title": "小说标题",
  "author": "作者名",
  "description": "小说简介",
  "chapters": [{"title": "精炼标题", "startPara": 1, "endPara": 20}],
  "characters": [{"name": "角色名", "description": "描述", "alias": "别名", "traits": "性格特征"}],
	  "relationships": [{"source": "角色A", "target": "角色B", "relationType": "关系类型", "description": "关系描述"}],
  "events": [{"name": "事件名", "description": "描述", "timeOrder": "时间顺序"}],
  "outline": "大纲内容"
}

总行数=%d。各章从startPara行到endPara行，连续不重叠、覆盖全文。不要省略任何内容！

带行号的内容：
%s`

	prompt := fmt.Sprintf(promptTemplate, paraCount, numberedContent)

	messages := []models.Message{
		{Role: "system", Content: "你是一个专业的小说编辑助手。只返回JSON，不要包含原文正文。"},
		{Role: "user", Content: prompt},
	}

	temp := float32(0)
	result, err := c.Chat(messages, &temp)
	if err != nil {
		return nil, err
	}

	// 从AI回复中提取JSON
	jsonBytes, err := extractAndCleanJSON(result)
	if err != nil {
		preview := []rune(result)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		log.Printf("[AI] JSON提取失败 | 原因=%v | 前200字符=%s", err, string(preview))
		return nil, fmt.Errorf("AI响应格式错误: %v", err)
	}

	// 解析带行号的AI响应
	var aiResp struct {
		Title       string `json:"title"`
		Author      string `json:"author"`
		Description string `json:"description"`
		Chapters    []struct {
			Title     string `json:"title"`
			StartPara int    `json:"startPara"`
			EndPara   int    `json:"endPara"`
		} `json:"chapters"`
		Characters    []models.Character         `json:"characters"`
		Relationships []models.NovelRelationship `json:"relationships"`
		Events        []models.Event             `json:"events"`
		Outline       string                     `json:"outline"`
	}
	if err := json.Unmarshal(jsonBytes, &aiResp); err != nil {
		preview := []rune(string(jsonBytes))
		if len(preview) > 200 {
			preview = preview[:200]
		}
		log.Printf("[AI] JSON解析失败 | 原因=%v | 前200字符=%s", err, string(preview))
		return nil, fmt.Errorf("解析AI响应JSON失败: %w", err)
	}

	// 根据行号从原文提取正文
	splitResult := &models.SplitResult{
		Title:         aiResp.Title,
		Author:        aiResp.Author,
		Description:   aiResp.Description,
		Characters:    aiResp.Characters,
		Events:        aiResp.Events,
		Relationships: aiResp.Relationships,
		Outline:       aiResp.Outline,
	}
	for _, ch := range aiResp.Chapters {
		start := ch.StartPara
		if start < 1 {
			start = 1
		}
		end := ch.EndPara
		if end > paraCount {
			end = paraCount
		}
		if start > end || start > paraCount {
			continue
		}
		splitResult.Chapters = append(splitResult.Chapters, models.SplitChapter{
			Title:   ch.Title,
			Content: strings.Join(paras[start-1:end], "\n\n"),
		})
	}

	log.Printf("[AI] 拆分完成 | 章节数=%d | 角色数=%d", len(splitResult.Chapters), len(splitResult.Characters))
	return splitResult, nil
}

// ExtractNovelInfo 使用AI提取小说的大纲、角色、关系和事件
// 支持传入已有元数据实现增量提取（AI 基于已有数据识别新增/变更）
func (c *DeepSeekClient) ExtractNovelInfo(chapters []models.Chapter, fullContent string,
	existingOutline string,
	existingChars []models.Character,
	existingRels []models.NovelRelationship,
	existingEvents []models.Event,
) (*models.ExtractionResult, error) {
	chapterSummaries := ""
	for i, ch := range chapters {
		contentPreview := ch.Content
		if len([]rune(contentPreview)) > 500 {
			contentPreview = string([]rune(contentPreview)[:500]) + "..."
		}
		chapterSummaries += fmt.Sprintf("第%d章 %s:\n%s\n\n", i+1, ch.Title, contentPreview)
	}

	if fullContent == "" {
		fullContent = chapterSummaries
	}

	// 构建已有元数据上下文（增量提取用），让 AI 基于已有数据识别新增/变更
	existingCtx := ""
	if existingOutline != "" || len(existingChars) > 0 || len(existingRels) > 0 || len(existingEvents) > 0 {
		existingCtx = "【已有元数据（请结合已有信息与新增内容，输出最完整的版本，包含已有和新识别的内容）】\n"
		if existingOutline != "" {
			existingCtx += "已有大纲：" + existingOutline + "\n"
		}
		if len(existingChars) > 0 {
			names := make([]string, 0, len(existingChars))
			for _, ch := range existingChars {
				desc := ch.Name
				if ch.Description != "" {
					desc += "(" + ch.Description + ")"
				}
				if ch.Traits != "" {
					desc += "[" + ch.Traits + "]"
				}
				names = append(names, desc)
			}
			existingCtx += "已有角色：" + strings.Join(names, "，") + "\n"
		}
		if len(existingRels) > 0 {
			relStrs := make([]string, 0, len(existingRels))
			for _, r := range existingRels {
				relStrs = append(relStrs, r.Source+"→"+r.Target+"("+r.RelationType+")")
			}
			existingCtx += "已有关系：" + strings.Join(relStrs, "，") + "\n"
		}
		if len(existingEvents) > 0 {
			evtNames := make([]string, 0, len(existingEvents))
			for _, e := range existingEvents {
				evtNames = append(evtNames, e.Name)
			}
			existingCtx += "已有事件：" + strings.Join(evtNames, "，") + "\n"
		}
		existingCtx += "\n"
	}

	prompt := fmt.Sprintf(`你是一个专业的小说编辑助手。请分析以下小说内容，提取结构化信息。

%s要求严格按照以下JSON格式返回：
{
  "outline": "完整的故事大纲，包括起承转合、主要情节线",
  "characters": [{"name": "角色名", "description": "角色详细描述", "alias": "别名", "traits": "性格特征"}],
  "relationships": [{"source": "角色A", "target": "角色B", "relationType": "关系类型（如：朋友/敌人/恋人/师徒等）", "description": "关系描述"}],
  "events": [{"name": "事件名称", "description": "事件详细描述", "timeOrder": "时间顺序（如：序章/第一章/事件一/时间点等）", "relatedChars": ["关联角色1", "关联角色2"]}]
}

小说内容：
%s`, existingCtx, fullContent)

	messages := []models.Message{
		{Role: "system", Content: "你是一个专业的小说编辑助手，擅长从小说文本中提取结构化信息。如果提供了已有元数据，请务必在输出中包含已有信息与新识别的内容，输出完整的元数据。请始终用JSON格式回复。"},
		{Role: "user", Content: prompt},
	}

	temp := float32(0)
	result, err := c.Chat(messages, &temp)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := extractAndCleanJSON(result)
	if err != nil {
		return nil, fmt.Errorf("AI响应格式错误: %w", err)
	}

	var extraction models.ExtractionResult
	if err := json.Unmarshal(jsonBytes, &extraction); err != nil {
		return nil, fmt.Errorf("解析AI响应JSON失败: %w", err)
	}

	return &extraction, nil
}

// buildStoryContext 构建故事背景信息块（角色/关系/事件/大纲），用于 AI prompt 前缀
func buildStoryContext(outline string, chars []models.Character, rels []models.NovelRelationship, events []models.Event) string {
	var b strings.Builder

	b.WriteString("【故事背景信息】\n")

	if outline != "" {
		b.WriteString("大纲：" + outline + "\n\n")
	}

	if len(chars) > 0 {
		b.WriteString("【角色一览】\n")
		for _, ch := range chars {
			parts := []string{ch.Name}
			if ch.Alias != "" {
				parts = append(parts, "（"+ch.Alias+"）")
			}
			if ch.Traits != "" {
				parts = append(parts, "— "+ch.Traits)
			}
			if ch.Description != "" {
				parts = append(parts, "："+ch.Description)
			}
			b.WriteString("- " + strings.Join(parts, " ") + "\n")
		}
		b.WriteString("\n")
	}

	if len(rels) > 0 {
		b.WriteString("【人物关系】\n")
		for _, rel := range rels {
			line := "- " + rel.Source + " → " + rel.Target + "：" + rel.RelationType
			if rel.Description != "" {
				line += "（" + rel.Description + "）"
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	if len(events) > 0 {
		b.WriteString("【已发生事件】\n")
		for _, ev := range events {
			line := "- " + ev.Name
			if ev.TimeOrder != nil {
				line = "- [" + fmt.Sprintf("%v", ev.TimeOrder) + "] " + ev.Name
			}
			if ev.Description != "" {
				line += "：" + ev.Description
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// --- 统一 prompt 构造 + 执行框架（消除润色/扩写/续写之间的重复） ---

// aiEditParams 所有编辑类操作（润色/扩写/续写）共用的参数集合
type aiEditParams struct {
	Content                string
	IsSelection            bool
	Outline                string
	Requirement            string
	PreviousChapterContent string
	Characters             []models.Character
	Relationships          []models.NovelRelationship
	Events                 []models.Event
}

// editOp 定义一种编辑操作的 prompt 要素
type editOp struct {
	systemMsg    string // system message
	headingTmpl  string // 标题模板（含 %s 表示 scope）
	instructions string // 通用操作指令
}

// buildMessages 基于 op 配置和参数构建 API 请求消息（含统一的 requirment 末尾处理）
func (op *editOp) buildMessages(params aiEditParams) []models.Message {
	ctx := buildStoryContext(params.Outline, params.Characters, params.Relationships, params.Events)

	scope := "整个章节"
	if params.IsSelection {
		scope = "选中的内容"
	}

	var prevCtx string
	if params.PreviousChapterContent != "" {
		prevCtx = "\n\n【前一章的内容（剧情上下文参考）】\n" + params.PreviousChapterContent + "\n\n【以上为前一章内容】\n"
	}

	// 统一 heading：支持含 %s 的模板（Polish/Expand）和固定字符串（ContinueWrite）
	heading := op.headingTmpl
	if strings.Contains(heading, "%s") {
		heading = fmt.Sprintf(heading, scope)
	}

	// 基础 prompt：上下文 + 内容 + 通用指令
	basePrompt := fmt.Sprintf("%s%s%s\n\n--- 原始内容 ---\n%s\n--- 以上 ---\n\n%s",
		ctx, prevCtx, heading, params.Content, op.instructions)

	// 用户要求放在 prompt 最末尾（利用 recency bias），以最高优先级标记
	if params.Requirement != "" {
		basePrompt += fmt.Sprintf("\n\n--- 最核心的用户修改要求（必须作为最高优先级指令严格执行）---\n%s\n---", params.Requirement)
	}

	return []models.Message{
		{Role: "system", Content: op.systemMsg},
		{Role: "user", Content: basePrompt},
	}
}

// execEdit 执行编辑类 AI 操作（非流式）
func (c *DeepSeekClient) execEdit(op *editOp, params aiEditParams) (string, error) {
	return c.Chat(op.buildMessages(params), nil)
}

// execEditStream 执行编辑类 AI 操作（流式）
func (c *DeepSeekClient) execEditStream(op *editOp, params aiEditParams, onChunk func(string)) (string, error) {
	return c.StreamChat(op.buildMessages(params), nil, onChunk)
}

// 三种编辑操作的配置定义（仅此处定义 prompt 差异，不再为每种操作写一遍函数体）
var (
	polishOp = &editOp{
		systemMsg:    "你是一个专业的小说润色助手。只返回润色后的小说正文。保持原文段落数量和结构，段落之间用空行隔开。用户的修改要求是最优先指令，必须严格遵循。",
		headingTmpl:  "请对以下%s进行润色改进：",
		instructions: "润色要点：改进表达、语法和流畅度，确保与前后文一致。\n格式要求：保持原文段落数量和结构，段落之间用空行隔开。只返回润色后的内容。",
	}

	expandOp = &editOp{
		systemMsg:    "你是一个专业的小说扩写助手。只返回扩写后的小说正文。段落之间用空行隔开。用户的修改要求是最优先指令，必须严格遵循。",
		headingTmpl:  "请对以下%s进行扩写，丰富细节和描写：",
		instructions: "请丰富细节、描写、对话和心理活动等，使内容更加生动充实。参考前一章内容，确保情节和上下文一致。\n格式要求：段落之间用空行隔开。只返回扩写后的内容。",
	}

	continueOp = &editOp{
		systemMsg:    "你是一个专业的小说续写助手。只返回续写的小说正文。续写必须推动情节向前发展，避免重复或模仿前文已有的内容。段落之间用空行隔开。",
		headingTmpl:  "--- 以下是本章需要续写的内容 ---",
		instructions: "请自然地续写接下来发生的内容，推动情节向前发展。注意与前一章保持合理衔接，但核心是创造新的进展——不要复述或模仿前文，避免使用相似的句式和表达。如果前文已经描述过某个场景或对话，续写应当转向新的发展。\n格式要求：段落之间用空行隔开。只返回续写的内容，不要包含任何解释。",
	}
)

// --- 公开 API（保持原有签名兼容） ---

// Polish AI润色（非流式）
func (c *DeepSeekClient) Polish(content string, isSelection bool, outline, requirement, previousChapterContent string, chars []models.Character, rels []models.NovelRelationship, events []models.Event) (string, error) {
	return c.execEdit(polishOp, aiEditParams{
		Content: content, IsSelection: isSelection, Outline: outline,
		Requirement: requirement, PreviousChapterContent: previousChapterContent,
		Characters: chars, Relationships: rels, Events: events,
	})
}

// PolishStream AI润色（流式）
func (c *DeepSeekClient) PolishStream(content string, isSelection bool, outline, requirement, previousChapterContent string, chars []models.Character, rels []models.NovelRelationship, events []models.Event, onChunk func(string)) (string, error) {
	return c.execEditStream(polishOp, aiEditParams{
		Content: content, IsSelection: isSelection, Outline: outline,
		Requirement: requirement, PreviousChapterContent: previousChapterContent,
		Characters: chars, Relationships: rels, Events: events,
	}, onChunk)
}

// Expand AI扩写（非流式）
func (c *DeepSeekClient) Expand(content string, isSelection bool, outline, requirement, previousChapterContent string, chars []models.Character, rels []models.NovelRelationship, events []models.Event) (string, error) {
	return c.execEdit(expandOp, aiEditParams{
		Content: content, IsSelection: isSelection, Outline: outline,
		Requirement: requirement, PreviousChapterContent: previousChapterContent,
		Characters: chars, Relationships: rels, Events: events,
	})
}

// ExpandStream AI扩写（流式）
func (c *DeepSeekClient) ExpandStream(content string, isSelection bool, outline, requirement, previousChapterContent string, chars []models.Character, rels []models.NovelRelationship, events []models.Event, onChunk func(string)) (string, error) {
	return c.execEditStream(expandOp, aiEditParams{
		Content: content, IsSelection: isSelection, Outline: outline,
		Requirement: requirement, PreviousChapterContent: previousChapterContent,
		Characters: chars, Relationships: rels, Events: events,
	}, onChunk)
}

// ContinueWrite AI续写（非流式）
func (c *DeepSeekClient) ContinueWrite(chapterContent, previousChapterContent, outline, requirement string, chars []models.Character, rels []models.NovelRelationship, events []models.Event) (string, error) {
	return c.execEdit(continueOp, aiEditParams{
		Content: chapterContent, Outline: outline,
		Requirement: requirement, PreviousChapterContent: previousChapterContent,
		Characters: chars, Relationships: rels, Events: events,
	})
}

// ContinueWriteStream AI续写（流式）
func (c *DeepSeekClient) ContinueWriteStream(chapterContent, previousChapterContent, outline, requirement string, chars []models.Character, rels []models.NovelRelationship, events []models.Event, onChunk func(string)) (string, error) {
	return c.execEditStream(continueOp, aiEditParams{
		Content: chapterContent, Outline: outline,
		Requirement: requirement, PreviousChapterContent: previousChapterContent,
		Characters: chars, Relationships: rels, Events: events,
	}, onChunk)
}

// CheckConnection 检查API连接是否正常
func (c *DeepSeekClient) CheckConnection() error {
	messages := []models.Message{
		{Role: "user", Content: "Hi"},
	}
	_, err := c.Chat(messages, nil)
	return err
}

// extractAndCleanJSON 从AI响应中提取JSON字节数组
// 自动处理：BOM头、markdown代码块包裹、前缀文本、不完整JSON等情况
func extractAndCleanJSON(text string) ([]byte, error) {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil, fmt.Errorf("响应内容为空")
	}

	// 检查并跳过 BOM (U+FEFF)
	pos := 1
	if len(runes) > 0 && runes[0] != 0xFEFF {
		pos = 0
	}

	// 跳过代码块标记和前缀文本，找到第一个 {
	for pos < len(runes) {
		if pos+7 <= len(runes) && string(runes[pos:pos+7]) == "```json" {
			pos += 7
			continue
		}
		if pos+3 <= len(runes) && string(runes[pos:pos+3]) == "```" {
			pos += 3
			continue
		}
		// 跳过空白和换行
		if runes[pos] == ' ' || runes[pos] == '\t' || runes[pos] == '\n' || runes[pos] == '\r' {
			pos++
			continue
		}
		if runes[pos] == '{' {
			break
		}
		// 遇到非 { 字符（前缀文本），继续跳过
		pos++
	}

	if pos >= len(runes) || runes[pos] != '{' {
		return nil, fmt.Errorf("未找到JSON起始标记")
	}

	// 通过花括号嵌套深度找到匹配的 }，比找最后一个 } 更可靠
	depth := 0
	inString := false
	escaped := false
	end := -1
	for i := pos; i < len(runes); i++ {
		c := runes[i]
		if escaped {
			escaped = false
			continue
		}
		if inString {
			if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	if end == -1 {
		return nil, fmt.Errorf("JSON不完整（未找到闭合的 }，depth=%d）", depth)
	}

	jsonStr := string(runes[pos:end])
	// 去除末尾可能的代码块标记残留
	jsonStr = strings.TrimSuffix(jsonStr, "```")

	return []byte(jsonStr), nil
}

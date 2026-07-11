package ai

import (
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
5. 生成故事大纲

JSON格式（不要包含原文正文）：
{
  "title": "小说标题",
  "author": "作者名",
  "description": "小说简介",
  "chapters": [{"title": "精炼标题", "startPara": 1, "endPara": 20}],
  "characters": [{"name": "角色名", "description": "描述", "alias": "别名", "traits": "性格特征"}],
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
		Characters []models.Character `json:"characters"`
		Events     []models.Event     `json:"events"`
		Outline    string             `json:"outline"`
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
		Title:       aiResp.Title,
		Author:      aiResp.Author,
		Description: aiResp.Description,
		Characters:  aiResp.Characters,
		Events:      aiResp.Events,
		Outline:     aiResp.Outline,
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
func (c *DeepSeekClient) ExtractNovelInfo(chapters []models.Chapter, fullContent string) (*models.ExtractionResult, error) {
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

	prompt := fmt.Sprintf(`你是一个专业的小说编辑助手。请分析以下小说内容，提取结构化信息。

要求严格按照以下JSON格式返回：
{
  "outline": "完整的故事大纲，包括起承转合、主要情节线",
  "characters": [{"name": "角色名", "description": "角色详细描述", "alias": "别名", "traits": "性格特征"}],
  "relationships": [{"source": "角色A", "target": "角色B", "relation": "关系类型（如：朋友/敌人/恋人/师徒等）", "description": "关系描述"}],
  "events": [{"name": "事件名称", "description": "事件详细描述", "timeOrder": "时间顺序（如：序章/第一章/事件一/时间点等）", "relatedChars": ["关联角色1", "关联角色2"]}]
}

小说内容：
%s`, fullContent)

	messages := []models.Message{
		{Role: "system", Content: "你是一个专业的小说编辑助手，擅长从小说文本中提取结构化信息。请始终用JSON格式回复。"},
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

// ContinueWrite AI续写
func (c *DeepSeekClient) ContinueWrite(chapterContent, outline, requirement string) (string, error) {
	var prompt string
	if requirement != "" {
		prompt = fmt.Sprintf(`你是一个专业的小说续写助手。

当前章节内容：
%s

故事大纲：
%s

续写要求：
%s

请续写接下来的内容，保持风格一致，情节连贯。只返回续写的内容，不要包含任何解释。`, chapterContent, outline, requirement)
	} else {
		prompt = fmt.Sprintf(`你是一个专业的小说续写助手。

当前章节内容：
%s

故事大纲：
%s

请自然地续写接下来的内容，保持风格一致，情节连贯。只返回续写的内容，不要包含任何解释。`, chapterContent, outline)
	}

	messages := []models.Message{
		{Role: "system", Content: "你是一个专业的小说续写助手。请只返回续写的小说正文内容，不要包含任何解释或标注。"},
		{Role: "user", Content: prompt},
	}

	return c.Chat(messages, nil)
}

// Polish AI润色
func (c *DeepSeekClient) Polish(content string, isSelection bool, outline, requirement string) (string, error) {
	scope := "整个章节"
	if isSelection {
		scope = "选中的内容"
	}

	var prompt string
	if requirement != "" {
		prompt = fmt.Sprintf(`你是一个专业的小说润色助手。请对以下%s进行润色改进。

原始内容：
%s

故事大纲（供参考）：
%s

润色要求：
%s

请保持原意和风格，改进表达、语法和流畅度。只返回润色后的内容。`, scope, content, outline, requirement)
	} else {
		prompt = fmt.Sprintf(`你是一个专业的小说润色助手。请对以下%s进行润色改进。

原始内容：
%s

故事大纲（供参考）：
%s

请保持原意和风格，改进表达、语法和流畅度。只返回润色后的内容。`, scope, content, outline)
	}

	messages := []models.Message{
		{Role: "system", Content: "你是一个专业的小说润色助手。请只返回润色后的小说正文内容，不要包含任何解释或标注。"},
		{Role: "user", Content: prompt},
	}

	return c.Chat(messages, nil)
}

// Expand AI扩写
func (c *DeepSeekClient) Expand(content string, isSelection bool, outline, requirement string) (string, error) {
	scope := "整个章节"
	if isSelection {
		scope = "选中的内容"
	}

	var prompt string
	if requirement != "" {
		prompt = fmt.Sprintf(`你是一个专业的小说扩写助手。请对以下%s进行扩写，丰富细节和描写。

原始内容：
%s

故事大纲（供参考）：
%s

扩写要求：
%s

请丰富细节、描写、对话和心理活动等，使内容更加生动充实。只返回扩写后的内容。`, scope, content, outline, requirement)
	} else {
		prompt = fmt.Sprintf(`你是一个专业的小说扩写助手。请对以下%s进行扩写，丰富细节和描写。

原始内容：
%s

故事大纲（供参考）：
%s

请丰富细节、描写、对话和心理活动等，使内容更加生动充实。只返回扩写后的内容。`, scope, content, outline)
	}

	messages := []models.Message{
		{Role: "system", Content: "你是一个专业的小说扩写助手。请只返回扩写后的小说正文内容，不要包含任何解释或标注。"},
		{Role: "user", Content: prompt},
	}

	return c.Chat(messages, nil)
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

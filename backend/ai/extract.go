package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"dida/models"
)

// SplitChapters 使用AI拆分小说章节（AI仅标注行号范围，后端从原文切分）
func (c *DeepSeekClient) SplitChapters(content string) (*models.SplitResult, error) {
	log.Printf("[AI] 开始拆分章节 | 内容长度=%d 字符", len([]rune(content)))

	rawLines := strings.Split(content, "\n")
	paras := make([]string, 0, len(rawLines))
	for _, l := range rawLines {
		if trimmed := strings.TrimSpace(l); trimmed != "" {
			paras = append(paras, trimmed)
		}
	}
	paraCount := len(paras)
	log.Printf("[AI] 段落数=%d", paraCount)

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

	jsonBytes, err := extractAndCleanJSON(result)
	if err != nil {
		preview := []rune(result)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("AI响应格式错误: %v", err)
	}

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
		return nil, fmt.Errorf("解析AI响应JSON失败: %w", err)
	}

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

// extractAndCleanJSON 从AI响应中提取JSON字节数组
func extractAndCleanJSON(text string) ([]byte, error) {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil, fmt.Errorf("响应内容为空")
	}

	pos := 1
	if len(runes) > 0 && runes[0] != 0xFEFF {
		pos = 0
	}

	for pos < len(runes) {
		if pos+7 <= len(runes) && string(runes[pos:pos+7]) == "```json" {
			pos += 7
			continue
		}
		if pos+3 <= len(runes) && string(runes[pos:pos+3]) == "```" {
			pos += 3
			continue
		}
		if runes[pos] == ' ' || runes[pos] == '\t' || runes[pos] == '\n' || runes[pos] == '\r' {
			pos++
			continue
		}
		if runes[pos] == '{' {
			break
		}
		pos++
	}

	if pos >= len(runes) || runes[pos] != '{' {
		return nil, fmt.Errorf("未找到JSON起始标记")
	}

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
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	return []byte(jsonStr), nil
}

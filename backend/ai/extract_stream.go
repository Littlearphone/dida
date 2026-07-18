package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"dida/models"
)

// ExtractNovelInfoStream 使用AI流式提取小说信息，通过 onChunk 回调实时输出原始响应文本
func (c *DeepSeekClient) ExtractNovelInfoStream(
	chapters []models.Chapter,
	fullContent string,
	existingOutline string,
	existingChars []models.Character,
	existingRels []models.NovelRelationship,
	existingEvents []models.Event,
	onChunk func(string),
) (*models.ExtractionResult, error) {
	chapterSummaries := ""
	for i, ch := range chapters {
		chapterSummaries += fmt.Sprintf("第%d章 %s:\n%s\n\n", i+1, ch.Title, ch.Content)
	}
	if fullContent == "" {
		fullContent = chapterSummaries
	}

	existingCtx := ""
	if existingOutline != "" || len(existingChars) > 0 || len(existingRels) > 0 || len(existingEvents) > 0 {
		existingCtx = "【已有元数据（仅作参考背景，了解已有哪些内容，避免重复提取）】\n"
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
  "events": [{"name": "事件名称", "description": "完整描述该情节——谁、因何而起、做了什么、导致什么结果，需完整交代前因后果", "timeOrder": "时间顺序（如：序章/第一章/事件一/时间点等）", "relatedChars": ["关联角色1", "关联角色2"]}]
}

小说内容：
%s`, existingCtx, fullContent)

	messages := []models.Message{
		{Role: "system", Content: "你是一个专业的小说编辑助手，擅长从小说文本中提取结构化信息。已有元数据仅作参考背景，避免重复提取。只返回本次分析新识别的内容。请始终用JSON格式回复。"},
		{Role: "user", Content: prompt},
	}

	temp := float32(0)
	fullText, err := c.StreamChat(messages, &temp, onChunk)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := extractAndCleanJSON(fullText)
	if err != nil {
		preview := []rune(fullText)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		log.Printf("[AI] 流式提取JSON失败，原始响应前300字符: %s", string(preview))
		return nil, fmt.Errorf("AI响应JSON提取失败：%w（可能是API返回了非JSON内容或内容过长导致截断，请重试）", err)
	}

	var extraction models.ExtractionResult
	if err := json.Unmarshal(jsonBytes, &extraction); err != nil {
		jsonPreview := []rune(string(jsonBytes))
		if len(jsonPreview) > 300 {
			jsonPreview = jsonPreview[:300]
		}
		log.Printf("[AI] 流式提取JSON解析失败，提取到的JSON前300字符: %s", string(jsonPreview))
		return nil, fmt.Errorf("提取结果JSON解析失败：%w（可能是字段格式不匹配，请重试）", err)
	}

	log.Printf("[AI] 流式提取完成 | 大纲字数=%d | 角色数=%d | 关系数=%d | 事件数=%d",
		len([]rune(extraction.Outline)), len(extraction.Characters), len(extraction.Relationships), len(extraction.Events))
	return &extraction, nil
}

package ai

import (
	"fmt"
	"strings"

	"dida/models"
)

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

// buildMessages 基于 op 配置和参数构建 API 请求消息
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

	heading := op.headingTmpl
	if strings.Contains(heading, "%s") {
		heading = fmt.Sprintf(heading, scope)
	}

	basePrompt := fmt.Sprintf("%s%s%s\n\n--- 原始内容 ---\n%s\n--- 以上 ---\n\n%s",
		ctx, prevCtx, heading, params.Content, op.instructions)

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

// 三种编辑操作的配置定义
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
		instructions: "请自然地续写接下来发生的内容，推动情节向前发展。注意与前一章保持合理衔接。\n格式要求：段落之间用空行隔开。只返回续写的内容，不要包含任何解释。",
	}
)

// --- 公开 API ---

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

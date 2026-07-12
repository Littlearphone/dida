package models

import (
	"time"
)

// Novel 小说模型，包含元数据、角色、关系和事件信息
type Novel struct {
	ID            string              `json:"id"`
	Title         string              `json:"title"`
	Author        string              `json:"author"`
	CoverPath     string              `json:"coverPath,omitempty"`   // 封面图片路径
	Description   string              `json:"description,omitempty"` // AI识别或用户填写的小说简介
	ChapterIDs    []string            `json:"chapterIds"`            // 章节ID有序列表
	Outline       string              `json:"outline,omitempty"`     // 大纲内容
	Characters    []Character         `json:"characters,omitempty"`
	Relationships []NovelRelationship `json:"relationships,omitempty"` // 平铺的角色关系
	Events        []Event             `json:"events,omitempty"`
	CreatedAt     time.Time           `json:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt"`
	WordCount     int64               `json:"wordCount"`
}

// Character 角色信息
type Character struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Alias       string `json:"alias,omitempty"`  // 别名
	Traits      string `json:"traits,omitempty"` // 性格特征
	// 注意：角色关系已迁移至 Novel.Relationships 平铺存储
}

// NovelRelationship 小说级角色关系（平铺，不挂在单个角色下）
type NovelRelationship struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	RelationType string `json:"relationType"`
	Description  string `json:"description,omitempty"`
}

// Event 事件信息
type Event struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	TimeOrder   interface{} `json:"timeOrder,omitempty"` // 时间顺序描述（兼容AI返回字符串或数字）
	ChapterID   string      `json:"chapterId,omitempty"` // 关联章节ID
}

// Chapter 章节模型
type Chapter struct {
	ID        string    `json:"id"`
	NovelID   string    `json:"novelId"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Order     int       `json:"order"`
	WordCount int64     `json:"wordCount"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Settings 应用设置
type Settings struct {
	NovelPath          string  `json:"novelPath"`          // 小说保存路径
	AIConfigured       bool    `json:"aiConfigured"`       // 是否已配置AI
	AIModel            string  `json:"aiModel"`            // AI模型（默认 deepseek-chat）
	Endpoint           string  `json:"endpoint"`           // AI接口地址
	APIKey             string  `json:"apiKey"`             // API密钥
	AutoSave           bool    `json:"autoSave"`           // 是否启用自动保存
	AutoSaveMs         int     `json:"autoSaveMs"`         // 自动保存间隔（毫秒）
	DefaultFontSize    int     `json:"defaultFontSize"`    // 默认字号
	DefaultLineSpacing float64 `json:"defaultLineSpacing"` // 默认行距
}

// AIConfig 用于设置API的请求体
type AIConfig struct {
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"apiKey"`
	Model    string `json:"model"`
}

// AIRequest AI对话请求
type AIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature *float32  `json:"temperature,omitempty"` // nil=使用模型默认，设置为0可得到确定性结果
}

// Message AI对话消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIResponse DeepSeek API响应
type AIResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice API响应的选项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage Token用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SplitResult AI拆分章节的结果
type SplitResult struct {
	Title         string              `json:"title,omitempty"`       // AI识别的小说标题
	Author        string              `json:"author,omitempty"`      // AI识别的作者
	Description   string              `json:"description,omitempty"` // AI识别的小说简介
	Chapters      []SplitChapter      `json:"chapters"`
	Characters    []Character         `json:"characters"`
	Relationships []NovelRelationship `json:"relationships,omitempty"` // AI识别的角色关系
	Events        []Event             `json:"events"`
	Outline       string              `json:"outline"`
}

// SplitChapter AI拆分出的章节
type SplitChapter struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ExtractionResult AI提取的小说信息
type ExtractionResult struct {
	Outline       string              `json:"outline"`
	Characters    []Character         `json:"characters"`
	Relationships []NovelRelationship `json:"relationships"`
	Events        []ExtractedEvent    `json:"events"`
}

// ExtractedEvent 提取的事件
type ExtractedEvent struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	TimeOrder    interface{} `json:"timeOrder"` // 时间顺序描述（兼容AI返回字符串或数字）
	RelatedChars []string    `json:"relatedChars"`
}

// ChapterSplitRequest 章节拆分请求
type ChapterSplitRequest struct {
	Content string `json:"content"`
}

// ContinueWriteRequest 续写请求
type ContinueWriteRequest struct {
	ChapterContent         string              `json:"chapterContent"`
	PreviousChapterContent string              `json:"previousChapterContent,omitempty"` // 上一章内容（新建章节时提供作为上下文）
	Outline                string              `json:"outline"`
	Requirement            string              `json:"requirement"`
	Characters             []Character         `json:"characters,omitempty"`
	Relationships          []NovelRelationship `json:"relationships,omitempty"`
	Events                 []Event             `json:"events,omitempty"`
}

// PolishRequest 润色请求
type PolishRequest struct {
	Content                string              `json:"content"`                          // 选中的内容或整章内容
	PreviousChapterContent string              `json:"previousChapterContent,omitempty"` // 上一章内容（剧情上下文参考）
	IsSelection            bool                `json:"isSelection"`                      // 是否为选中内容
	Outline                string              `json:"outline"`
	Requirement            string              `json:"requirement"`
	Characters             []Character         `json:"characters,omitempty"`
	Relationships          []NovelRelationship `json:"relationships,omitempty"`
	Events                 []Event             `json:"events,omitempty"`
}

// ExpandRequest 扩写请求
type ExpandRequest struct {
	Content                string              `json:"content"`
	PreviousChapterContent string              `json:"previousChapterContent,omitempty"`
	IsSelection            bool                `json:"isSelection"`
	Outline                string              `json:"outline"`
	Requirement            string              `json:"requirement"`
	Characters             []Character         `json:"characters,omitempty"`
	Relationships          []NovelRelationship `json:"relationships,omitempty"`
	Events                 []Event             `json:"events,omitempty"`
}

// AIResult AI处理结果
type AIResult struct {
	Original string `json:"original"`
	Result   string `json:"result"`
}

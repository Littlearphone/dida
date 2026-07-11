// 小说数据模型
export interface Novel {
  id: string
  title: string
  author: string
  description?: string  // AI识别或用户填写的小说简介
  coverPath?: string
  chapterIds: string[]
  outline?: string
  characters?: Character[]
  events?: Event[]
  createdAt: string
  updatedAt: string
  wordCount: number
}

// 角色信息
export interface Character {
  name: string
  description?: string
  alias?: string
  traits?: string
  relationships?: Relationship[]
}

// 角色关系
export interface Relationship {
  targetName: string
  relationType: string
  description?: string
}

// 事件信息
export interface Event {
  name: string
  description?: string
  timeOrder?: string
  chapterId?: string
}

// 章节模型
export interface Chapter {
  id: string
  novelId: string
  title: string
  content: string
  order: number
  wordCount: number
  createdAt: string
  updatedAt: string
}

// 应用设置
export interface Settings {
  novelPath: string
  aiConfigured: boolean
  aiModel: string
  endpoint: string
  hasApiKey: boolean
  autoSave: boolean
  autoSaveMs: number
  defaultFontSize: number
  defaultLineSpacing: number
}

// 设置表单（用于提交更新）
export interface SettingsForm {
  novelPath?: string
  aiModel?: string
  endpoint?: string
  apiKey?: string
  autoSave?: boolean
  autoSaveMs?: number
  defaultFontSize?: number
  defaultLineSpacing?: number
}

// AI 拆分章节结果
export interface SplitResult {
  title?: string           // AI识别的小说标题
  author?: string          // AI识别的作者
  description?: string     // AI识别的小说简介
  chapters: SplitChapter[]
  characters: Character[]
  events: Event[]
  outline: string
}

export interface SplitChapter {
  title: string
  content: string
}

// AI 提取结果
export interface ExtractionResult {
  outline: string
  characters: Character[]
  relationships: RelationshipEntry[]
  events: ExtractedEvent[]
}

export interface RelationshipEntry {
  source: string
  target: string
  relation: string
  description?: string
}

export interface ExtractedEvent {
  name: string
  description: string
  timeOrder: string
  relatedChars: string[]
}

// AI 结果（润色/扩写）
export interface AIResult {
  original: string
  result: string
}

// AI 状态
export interface AIStatus {
  configured: boolean
  connected?: boolean
  model: string
  endpoint: string
  error?: string
}

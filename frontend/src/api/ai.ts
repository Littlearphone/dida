import type {AIResult, AIStatus, ExtractionResult, SplitResult} from '../types'

const BASE = '/api/ai'

/** 检查 AI 连接状态 */
export async function checkAIStatus(): Promise<AIStatus> {
  const res = await fetch(`${BASE}/status`)
  if (!res.ok) throw new Error('检查AI状态失败')
  return res.json()
}

/** 使用 AI 拆分章节 */
export async function splitChapters(content: string): Promise<SplitResult> {
  const res = await fetch(`${BASE}/split-chapters`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'AI拆分失败')
  }
  return res.json()
}

/** 提取小说信息 */
export async function extractInfo(data: {
  chapters: { id: string; title: string; content: string; order: number }[]
  fullContent?: string
}): Promise<ExtractionResult> {
  const res = await fetch(`${BASE}/extract-info`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'AI提取失败')
  }
  return res.json()
}

/** AI 续写 */
export async function continueWrite(data: {
  chapterContent: string
  outline: string
  requirement?: string
}): Promise<{ result: string }> {
  const res = await fetch(`${BASE}/continue-write`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'AI续写失败')
  }
  return res.json()
}

/** AI 润色 */
export async function polish(data: {
  content: string
  isSelection: boolean
  outline: string
  requirement?: string
}): Promise<AIResult> {
  const res = await fetch(`${BASE}/polish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'AI润色失败')
  }
  return res.json()
}

/** AI 扩写 */
export async function expand(data: {
  content: string
  isSelection: boolean
  outline: string
  requirement?: string
}): Promise<AIResult> {
  const res = await fetch(`${BASE}/expand`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'AI扩写失败')
  }
  return res.json()
}

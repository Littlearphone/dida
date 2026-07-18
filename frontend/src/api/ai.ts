import type {AIResult, AIStatus, Character, Event, ExtractionResult, NovelRelationship, SplitResult} from '../types'

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

/** 提取小说信息（支持传入已有元数据进行增量提取，支持取消和流式进度） */
export async function extractInfo(data: {
  chapters: { id: string; title: string; content: string; order: number }[]
  fullContent?: string
  existingOutline?: string
  existingCharacters?: Character[]
  existingRelations?: NovelRelationship[]
  existingEvents?: Event[]
}, signal?: AbortSignal, onChunk?: (text: string) => void): Promise<ExtractionResult> {
  const res = await fetch(`${BASE}/extract-info`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      // 有 onChunk 回调时请求 SSE 流式响应，实时显示 AI 原始输出
      ...(onChunk ? { Accept: 'text/event-stream' } : {}),
    },
    body: JSON.stringify(data),
    signal,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || 'AI提取失败')
  }

  const contentType = res.headers.get('content-type') || ''

  // SSE 流式模式：实时接收 AI 原始响应文本 + 最终结构化结果
  if (onChunk && contentType.includes('text/event-stream')) {
    return readExtractSSEStream(res, onChunk)
  }

  // 非流式模式
  return res.json()
}

/** 解析提取专用的 SSE 流：原始 chunks 通过 onChunk 回调，最后返回解析后的结构化结果 */
async function readExtractSSEStream(
  res: Response,
  onChunk: (text: string) => void,
): Promise<ExtractionResult> {
  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const parts = buffer.split('\n\n')
    buffer = parts.pop() || ''

    for (const part of parts) {
      for (const line of part.split('\n')) {
        if (!line.startsWith('data: ')) continue
        const data = line.slice(6).trim()
        if (data === '[DONE]') {
          // 流结束但没有收到结果，抛错
          throw new Error('AI提取未返回有效结果')
        }

        try {
          const parsed = JSON.parse(data)
          if (parsed.error) throw new Error(parsed.error)
          if (parsed.text) onChunk(parsed.text)
          // 最后一个事件携带结构化结果
          if (parsed.result) return parsed.result as ExtractionResult
        } catch (e: any) {
          if (e.message && !e.message.includes('JSON')) throw e
        }
      }
    }
  }

  throw new Error('AI提取响应不完整')
}

/** AI 续写
 *  支持流式（传入 onChunk）和非流式两种模式
 *  @param onChunk 流式模式下每次收到内容增量时回调，参数为完整累积文本和当前增量
 *  @param signal 用于取消请求的 AbortSignal
 */
export async function continueWrite(
  data: {
    chapterContent: string
    previousChapterContent?: string // 上一章内容（新建章节上下文的连续参考）
    outline: string
    requirement?: string
    characters?: Character[]
    relationships?: NovelRelationship[]
    events?: Event[]
  },
  onChunk?: (fullText: string, delta: string) => void,
  signal?: AbortSignal,
): Promise<string> {
  const res = await fetch(`${BASE}/continue-write`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      // 有 onChunk 回调时请求流式响应
      ...(onChunk ? { Accept: 'text/event-stream' } : {}),
    },
    body: JSON.stringify(data),
    signal,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'AI续写失败' }))
    throw new Error(err.error || 'AI续写失败')
  }

  const contentType = res.headers.get('content-type') || ''

  // SSE 流式模式
  if (onChunk && contentType.includes('text/event-stream')) {
    return readSSEStream(res, onChunk)
  }

  // 非流式：JSON 响应
  const result = await res.json()
  return result.result || ''
}

/** 解析 SSE 流，逐块回调 */
async function readSSEStream(
  res: Response,
  onChunk: (fullText: string, delta: string) => void,
): Promise<string> {
  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let fullText = ''
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })

    // 按 SSE 事件分隔符（双换行）拆分
    const parts = buffer.split('\n\n')
    buffer = parts.pop() || '' // 保留不完整部分

    for (const part of parts) {
      for (const line of part.split('\n')) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6).trim()
          if (data === '[DONE]') return fullText

          try {
            const parsed = JSON.parse(data)
            if (parsed.error) {
              throw new Error(parsed.error)
            }
            if (parsed.text) {
              fullText += parsed.text
              onChunk(fullText, parsed.text)
            }
          } catch (e: any) {
            if (e.message && e.message !== 'AI续写失败') {
              throw e
            }
          }
        }
      }
    }
  }

  // 处理 buffer 中剩余内容
  if (buffer.startsWith('data: ')) {
    const data = buffer.slice(6).trim()
    if (data !== '[DONE]') {
      try {
        const parsed = JSON.parse(data)
        if (parsed.text) fullText += parsed.text
      } catch { /* 忽略不完整结尾 */ }
    }
  }

  return fullText
}

/** AI 润色（支持流式，传入 onChunk 时以 SSE 流式返回） */
export async function polish(
  data: {
    content: string
    isSelection: boolean
    previousChapterContent?: string
    outline: string
    requirement?: string
    characters?: Character[]
    relationships?: NovelRelationship[]
    events?: Event[]
  },
  onChunk?: (fullText: string, delta: string) => void,
  signal?: AbortSignal,
): Promise<string> {
  const res = await fetch(`${BASE}/polish`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(onChunk ? { Accept: 'text/event-stream' } : {}),
    },
    body: JSON.stringify(data),
    signal,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'AI润色失败' }))
    throw new Error(err.error || 'AI润色失败')
  }

  const contentType = res.headers.get('content-type') || ''
  if (onChunk && contentType.includes('text/event-stream')) {
    return readSSEStream(res, onChunk)
  }

  const result = await res.json()
  return result.result || ''
}

/** AI 扩写（支持流式，传入 onChunk 时以 SSE 流式返回） */
export async function expand(
  data: {
    content: string
    isSelection: boolean
    previousChapterContent?: string
    outline: string
    requirement?: string
    characters?: Character[]
    relationships?: NovelRelationship[]
    events?: Event[]
  },
  onChunk?: (fullText: string, delta: string) => void,
  signal?: AbortSignal,
): Promise<string> {
  const res = await fetch(`${BASE}/expand`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(onChunk ? { Accept: 'text/event-stream' } : {}),
    },
    body: JSON.stringify(data),
    signal,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'AI扩写失败' }))
    throw new Error(err.error || 'AI扩写失败')
  }

  const contentType = res.headers.get('content-type') || ''
  if (onChunk && contentType.includes('text/event-stream')) {
    return readSSEStream(res, onChunk)
  }

  const result = await res.json()
  return result.result || ''
}

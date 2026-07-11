import type {Chapter, Novel} from '../types'

const BASE = '/api/novels'

/** 获取所有小说列表 */
export async function fetchNovels(): Promise<Novel[]> {
  const res = await fetch(BASE)
  if (!res.ok) throw new Error('获取小说列表失败')
  return res.json()
}

/** 创建新小说 */
export async function createNovel(title: string, author: string = ''): Promise<Novel> {
  const res = await fetch(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, author }),
  })
  if (!res.ok) throw new Error('创建小说失败')
  return res.json()
}

/** 获取小说详情 */
export async function getNovel(id: string): Promise<Novel> {
  const res = await fetch(`${BASE}/${id}`)
  if (!res.ok) throw new Error('获取小说详情失败')
  return res.json()
}

/** 更新小说元数据 */
export async function updateNovel(id: string, data: { title?: string; description?: string; outline?: string; characters?: any[]; events?: any[] }): Promise<Novel> {
  const res = await fetch(`${BASE}/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('更新小说失败')
  return res.json()
}

/** 删除小说 */
export async function deleteNovel(id: string): Promise<void> {
  const res = await fetch(`${BASE}/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('删除小说失败')
}

/** 导入小说 */
export async function importNovel(data: {
  title: string
  skipAISplit: boolean
  chapters: { title: string; content: string }[]
}): Promise<Novel> {
  const res = await fetch(`${BASE}/import`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('导入小说失败')
  return res.json()
}

/** 获取小说所有章节 */
export async function getChapters(novelId: string): Promise<Chapter[]> {
  const res = await fetch(`${BASE}/${novelId}/chapters`)
  if (!res.ok) throw new Error('获取章节列表失败')
  return res.json()
}

/** 重排章节顺序 */
export async function reorderChapters(novelId: string, chapterIds: string[]): Promise<void> {
  const res = await fetch(`${BASE}/${novelId}/chapters/reorder`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ chapterIds }),
  })
  if (!res.ok) throw new Error('重排章节失败')
}

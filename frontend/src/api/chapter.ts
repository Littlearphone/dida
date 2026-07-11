import type {Chapter} from '../types'

const BASE = '/api/chapters'

/** 创建章节 */
export async function createChapter(data: {
  novelId: string
  title: string
  content: string
  order: number
}): Promise<Chapter> {
  const res = await fetch(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('创建章节失败')
  return res.json()
}

/** 获取章节内容 */
export async function getChapter(id: string): Promise<Chapter> {
  const res = await fetch(`${BASE}/${id}`)
  if (!res.ok) throw new Error('获取章节失败')
  return res.json()
}

/** 更新章节 */
export async function updateChapter(id: string, data: { title?: string; content?: string }): Promise<Chapter> {
  const res = await fetch(`${BASE}/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('更新章节失败')
  return res.json()
}

/** 自动保存章节 */
export async function autoSaveChapter(id: string, content: string): Promise<Chapter> {
  const res = await fetch(`${BASE}/${id}/autosave`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  })
  if (!res.ok) throw new Error('自动保存失败')
  return res.json()
}

/** 删除章节 */
export async function deleteChapter(id: string): Promise<void> {
  const res = await fetch(`${BASE}/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('删除章节失败')
}

import type {Settings, SettingsForm} from '../types'

const BASE = '/api/settings'

/** 获取当前设置 */
export async function getSettings(): Promise<Settings> {
  const res = await fetch(BASE)
  if (!res.ok) throw new Error('获取设置失败')
  return res.json()
}

/** 更新设置 */
export async function updateSettings(data: SettingsForm): Promise<void> {
  const res = await fetch(BASE, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('更新设置失败')
}

/** 获取 API Key（掩码后） */
export async function getAPIKey(): Promise<string> {
  const res = await fetch(`${BASE}/apikey`)
  if (!res.ok) throw new Error('获取API Key失败')
  const data = await res.json()
  return data.apiKey
}

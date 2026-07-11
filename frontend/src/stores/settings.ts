import {defineStore} from 'pinia'
import {ref} from 'vue'
import type {Settings, SettingsForm} from '../types'
import * as settingsApi from '../api/settings'

/** 应用设置状态管理 */
export const useSettingsStore = defineStore('settings', () => {
  const settings = ref<Settings | null>(null)
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      settings.value = await settingsApi.getSettings()
    } catch (e) {
      console.warn('加载设置失败:', e)
    } finally {
      loading.value = false
    }
  }

  async function update(form: SettingsForm): Promise<boolean> {
    try {
      await settingsApi.updateSettings(form)
      await load()
      return true
    } catch (e) {
      console.warn('更新设置失败:', e)
      return false
    }
  }

  return { settings, loading, load, update }
})

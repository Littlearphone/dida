import {defineStore} from 'pinia'
import {ref} from 'vue'
import type {Chapter, Novel} from '../types'
import * as novelApi from '../api/novel'
import * as chapterApi from '../api/chapter'

/** 小说/章节全局状态管理 */
export const useNovelStore = defineStore('novel', () => {
  // === 状态 ===
  const novels = ref<Novel[]>([])
  const currentNovel = ref<Novel | null>(null)
  const chapters = ref<Chapter[]>([])
  const currentChapter = ref<Chapter | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // === 小说操作 ===
  async function loadNovels() {
    loading.value = true
    error.value = null
    try {
      novels.value = await novelApi.fetchNovels()
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  async function createNovel(title: string, author: string = ''): Promise<Novel | null> {
    try {
      const novel = await novelApi.createNovel(title, author)
      novels.value.unshift(novel)
      return novel
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
      return null
    }
  }

  async function deleteNovel(id: string): Promise<boolean> {
    try {
      await novelApi.deleteNovel(id)
      novels.value = novels.value.filter(n => n.id !== id)
      if (currentNovel.value?.id === id) {
        currentNovel.value = null
        chapters.value = []
        currentChapter.value = null
      }
      return true
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
      return false
    }
  }

  function selectNovel(novel: Novel | null) {
    currentNovel.value = novel
    chapters.value = []
    currentChapter.value = null
    if (novel) {
      loadChapters(novel.id)
    }
  }

  // === 章节操作 ===
  async function loadChapters(novelId: string) {
    try {
      chapters.value = await novelApi.getChapters(novelId)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
    }
  }

  async function loadChapter(id: string) {
    try {
      currentChapter.value = await chapterApi.getChapter(id)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
    }
  }

  function selectChapter(chapter: Chapter | null) {
    currentChapter.value = chapter
  }

  async function createChapter(data: {
    novelId: string
    title: string
    content: string
    order: number
  }): Promise<Chapter | null> {
    try {
      const ch = await chapterApi.createChapter(data)
      chapters.value.push(ch)
      // 更新小说章节列表
      const novel = novels.value.find(n => n.id === data.novelId)
      if (novel) {
        novel.chapterIds.push(ch.id)
      }
      return ch
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
      return null
    }
  }

  async function updateChapter(id: string, data: { title?: string; content?: string }): Promise<boolean> {
    try {
      const updated = await chapterApi.updateChapter(id, data)
      const idx = chapters.value.findIndex(c => c.id === id)
      if (idx !== -1) chapters.value[idx] = updated
      if (currentChapter.value?.id === id) currentChapter.value = updated
      return true
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
      return false
    }
  }

  async function autoSaveChapter(id: string, content: string): Promise<boolean> {
    try {
      const updated = await chapterApi.autoSaveChapter(id, content)
      const idx = chapters.value.findIndex(c => c.id === id)
      if (idx !== -1) chapters.value[idx] = updated
      if (currentChapter.value?.id === id) currentChapter.value = updated
      return true
    } catch (e: unknown) {
      console.warn('自动保存失败:', e instanceof Error ? e.message : e)
      return false
    }
  }

  async function deleteChapter(id: string): Promise<boolean> {
    try {
      await chapterApi.deleteChapter(id)
      chapters.value = chapters.value.filter(c => c.id !== id)
      if (currentChapter.value?.id === id) currentChapter.value = null
      return true
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
      return false
    }
  }

  async function importNovel(data: {
    title: string
    skipAISplit: boolean
    chapters: { title: string; content: string }[]
  }): Promise<Novel | null> {
    try {
      const novel = await novelApi.importNovel(data)
      novels.value.unshift(novel)
      return novel
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
      return null
    }
  }

  async function reorderChapters(novelId: string, chapterIds: string[]): Promise<boolean> {
    try {
      await novelApi.reorderChapters(novelId, chapterIds)
      return true
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
      return false
    }
  }

  async function updateNovelMeta(id: string, data: { title?: string; description?: string }): Promise<boolean> {
    try {
      const updated = await novelApi.updateNovel(id, data)
      const idx = novels.value.findIndex(n => n.id === id)
      if (idx !== -1) {
        novels.value[idx] = { ...novels.value[idx], ...updated }
      }
      return true
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
      return false
    }
  }

  return {
    novels, currentNovel, chapters, currentChapter, loading, error,
    loadNovels, createNovel, deleteNovel, selectNovel,
    loadChapters, loadChapter, selectChapter,
    createChapter, updateChapter, autoSaveChapter, deleteChapter,
    importNovel,
    reorderChapters,
    updateNovelMeta,
  }
})

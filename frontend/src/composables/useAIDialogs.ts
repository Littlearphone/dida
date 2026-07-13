/**
 * AI 弹框状态 & 提取逻辑 composable
 * 管理：续写/润色/扩写/AI设置/小说信息/提取结果等弹框的状态
 */
import { computed, nextTick, ref } from 'vue'
import type { ExtractionResult } from '../types'
import * as aiApi from '../api/ai'

export function useAIDialogs(
  novelStore: any,
  settingsStore: any,
  editor: any,
  contentChanged: { value: boolean },
  doSaveChapter: () => Promise<boolean>,
  message: any,
) {
  // === AI 弹框状态 ===
  const showContinueWrite = ref(false)
  const showAIEdit = ref(false)
  const showAISetup = ref(false)
  const showNovelInfo = ref(false)
  const showExtractResult = ref(false)
  const extractResult = ref<ExtractionResult | null>(null)
  const extractLoading = ref(false)
  const aiEditMode = ref<'polish' | 'expand'>('polish')
  const pendingEditContent = ref('')
  const refinedContinueResult = ref('')
  const aiConfigured = computed(() => settingsStore.settings?.aiConfigured ?? false)

  function openAIEdit(mode: 'polish' | 'expand') {
    aiEditMode.value = mode
    pendingEditContent.value = ''
    showAIEdit.value = true
  }

  function handleContinueEdit(mode: 'polish' | 'expand', text: string) {
    pendingEditContent.value = text
    aiEditMode.value = mode
    showAIEdit.value = true
    refinedContinueResult.value = ''
  }

  function handleReplaceExternal(text: string) {
    refinedContinueResult.value = text
  }

  // === AI 续写后自动提取元数据 ===
  async function handlePostInsertExtract() {
    const n = novelStore.currentNovel
    const currentChapter = novelStore.currentChapter
    if (!currentChapter || !n || !settingsStore.settings?.aiConfigured) return

    if (contentChanged.value) await doSaveChapter()
    const ch = currentChapter
    if (!ch) return

    extractLoading.value = true
    const loadingMsg = message.loading('正在提取章节元数据...', { duration: 0 })
    try {
      const result = await aiApi.extractInfo({
        chapters: [{
          id: ch.id,
          title: ch.title || '',
          content: ch.content,
          order: ch.order,
        }],
        existingOutline: n.outline,
        existingCharacters: n.characters,
        existingRelations: n.relationships,
        existingEvents: n.events,
      })
      loadingMsg.destroy()
      showExtractResult.value = true
      extractResult.value = result
    } catch (e: unknown) {
      loadingMsg.destroy()
      console.warn('AI 提取元数据失败:', e instanceof Error ? e.message : e)
      message.warning('AI 元数据提取未完成，不影响已有内容')
    } finally {
      extractLoading.value = false
    }
  }

  /** 手动提取/补充元数据 */
  async function handleExtractSupplement() {
    const ch = novelStore.currentChapter
    const n = novelStore.currentNovel
    if (!ch || !n) { message.warning('没有可分析的章节内容'); return }

    let contentToExtract: string
    if (editor.value) {
      const { from, to } = editor.value.state.selection
      contentToExtract = from !== to
        ? editor.value.state.doc.textBetween(from, to)
        : ch.content
    } else {
      contentToExtract = ch.content
    }
    if (!contentToExtract) { message.warning('内容为空'); return }

    extractLoading.value = true
    const loadingMsg = message.loading('正在提取元数据，请稍候...', { duration: 0 })
    try {
      const result = await aiApi.extractInfo({
        chapters: [{
          id: ch.id,
          title: ch.title || '',
          content: contentToExtract,
          order: ch.order,
        }],
        existingOutline: n.outline,
        existingCharacters: n.characters,
        existingRelations: n.relationships,
        existingEvents: n.events,
      })
      loadingMsg.destroy()
      showExtractResult.value = true
      extractResult.value = result
    } catch (e: unknown) {
      loadingMsg.destroy()
      const msg = e instanceof Error ? e.message : String(e)
      if (msg.includes('JSON') || msg.includes('json') || msg.includes('格式')) {
        message.error('AI 返回格式异常，请稍后重试或检查内容是否过长')
      } else {
        message.error('提取失败: ' + msg)
      }
    } finally {
      extractLoading.value = false
    }
  }

  return {
    showContinueWrite,
    showAIEdit,
    showAISetup,
    showNovelInfo,
    showExtractResult,
    extractResult,
    extractLoading,
    aiEditMode,
    pendingEditContent,
    refinedContinueResult,
    aiConfigured,
    openAIEdit,
    handleContinueEdit,
    handleReplaceExternal,
    handlePostInsertExtract,
    handleExtractSupplement,
  }
}

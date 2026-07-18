/**
 * AI 弹框状态 & 提取逻辑 composable
 * 管理：续写/润色/扩写/AI设置/小说信息/提取结果等弹框的状态
 */
import { computed, nextTick, ref } from 'vue'
import type { ExtractionResult } from '../types'
import * as aiApi from '../api/ai'

/** 归类提取错误，返回对用户友好的提示 */
function classifyExtractError(e: unknown): string {
  if (e instanceof Error) {
    if (e.name === 'AbortError') return 'CANCELLED'
    const msg = e.message
    // 后端 JSON 提取失败（AI 返回格式异常）
    if (msg.includes('JSON') || msg.includes('json') || msg.includes('格式') || msg.includes('未找到')) {
      return 'AI 返回格式异常（JSON 解析失败），请重试或适当精简内容后重试'
    }
    // 后端 API 错误（DeepSeek/OpenAI 侧）
    if (msg.includes('API返回错误') || msg.includes('API请求失败')) {
      return `AI 接口异常：${msg}`
    }
    // 网络错误
    if (msg.includes('fetch') || msg.includes('network') || msg.includes('NetworkError') || msg.includes('Failed to fetch')) {
      return '网络连接失败，请检查 AI 接口地址和网络状态'
    }
    // 超时
    if (msg.includes('timeout') || msg.includes('Timeout')) {
      return 'AI 响应超时（超过 5 分钟），请重试或精简章节内容'
    }
    return `提取失败：${msg}`
  }
  return `提取失败：${String(e)}`
}

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
  /** 流式提取时累积的 AI 原始响应文本，用于实时展示进度 */
  const extractStreamText = ref('')
  const aiEditMode = ref<'polish' | 'expand'>('polish')
  const pendingEditContent = ref('')
  const refinedContinueResult = ref('')
  const aiConfigured = computed(() => settingsStore.settings?.aiConfigured ?? false)

  // 提取操作的取消控制器
  let extractAbortController: AbortController | null = null

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

  /** 取消正在进行的提取请求 */
  function cancelExtract() {
    if (extractAbortController) {
      extractAbortController.abort()
      extractAbortController = null
    }
    extractLoading.value = false
    showExtractResult.value = false
  }

  /** 执行提取的公共逻辑 */
  async function doExtract(contentToExtract: string, isAutoExtract: boolean) {
    const ch = novelStore.currentChapter
    const n = novelStore.currentNovel
    if (!ch || !n) return

    extractLoading.value = true
    extractStreamText.value = '' // 重置流式文本
    // 创建新的 AbortController 用于本次请求
    extractAbortController = new AbortController()
    const signal = extractAbortController.signal

    // 手动提取：立刻打开弹框展示流式进度；自动提取：静默 toast
    if (!isAutoExtract) {
      extractResult.value = null
      showExtractResult.value = true
    }
    const loadingMsg = isAutoExtract
      ? message.loading('正在提取章节元数据...', { duration: 0 })
      : null

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
      }, signal,
      // onChunk: 累积 AI 原始响应文本，供 ExtractResultDialog 实时展示
      (text: string) => {
        extractStreamText.value += text
      })

      if (loadingMsg) loadingMsg.destroy()
      extractResult.value = result
      // 弹框已在上面打开，result 更新后自动切换为结构化结果视图
    } catch (e: unknown) {
      if (loadingMsg) loadingMsg.destroy()
      const classified = classifyExtractError(e)
      if (classified === 'CANCELLED') {
        showExtractResult.value = false // 取消时关闭弹框
        return
      }
      if (isAutoExtract) {
        // 自动提取（续写后）失败不打扰用户
        console.warn('AI 提取元数据失败:', e instanceof Error ? e.message : e)
        message.warning('AI 元数据提取未完成，不影响已有内容')
      } else {
        // 手动提取失败：关闭进度弹框，显示错误
        showExtractResult.value = false
        message.error(classified)
      }
    } finally {
      extractLoading.value = false
      extractAbortController = null
    }
  }

  // === AI 续写后自动提取元数据 ===
  async function handlePostInsertExtract() {
    const n = novelStore.currentNovel
    const currentChapter = novelStore.currentChapter
    if (!currentChapter || !n || !settingsStore.settings?.aiConfigured) return

    if (contentChanged.value) await doSaveChapter()
    const ch = currentChapter
    if (!ch) return

    await doExtract(ch.content, true)
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

    await doExtract(contentToExtract, false)
  }

  return {
    showContinueWrite,
    showAIEdit,
    showAISetup,
    showNovelInfo,
    showExtractResult,
    extractResult,
    extractLoading,
    extractStreamText,
    aiEditMode,
    pendingEditContent,
    refinedContinueResult,
    aiConfigured,
    openAIEdit,
    handleContinueEdit,
    handleReplaceExternal,
    handlePostInsertExtract,
    handleExtractSupplement,
    cancelExtract,
  }
}

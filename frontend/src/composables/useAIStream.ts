/**
 * AI 流式请求共享逻辑 — 进度模拟、自动重试、请求取消
 * 被 AIContinueDialog 和 AIEditDialog 共用
 */
import { ref, onUnmounted } from 'vue'
import type { Ref } from 'vue'

export function useAIStream() {
  const loading = ref(false)
  const progress = ref(0)
  const progressText = ref('')
  let abortController: AbortController | null = null
  let progressTimer: ReturnType<typeof setInterval> | null = null

  /** 最大自动重试次数 */
  const MAX_RETRIES = 2
  let retries = 0
  let cancelRequested = false

  function getAbortSignal(): AbortSignal {
    abortController = new AbortController()
    return abortController.signal
  }

  function cancelRequest() {
    cancelRequested = true
    abortController?.abort()
  }

  function cleanupRequest() {
    if (loading.value) cancelRequest()
    stopProgressSimulation()
    abortController = null
  }

  function resetRetry() {
    retries = 0
    cancelRequested = false
  }

  function shouldRetry(): boolean {
    return retries <= MAX_RETRIES && !cancelRequested
  }

  function incrementRetry() {
    retries++
  }

  function isCanceled(): boolean {
    return cancelRequested
  }

  // === 进度模拟 ===
  function startProgressSimulation(statusPrefix: string) {
    progress.value = 0
    progressText.value = `正在准备${statusPrefix}请求...`
    progressTimer = setInterval(() => {
      const remaining = 90 - progress.value
      if (remaining > 0) {
        progress.value += Math.max(0.5, remaining * 0.08)
      }
      if (progress.value < 20) {
        progressText.value = `正在准备${statusPrefix}请求...`
      } else if (progress.value < 50) {
        progressText.value = '正在请求 AI 服务...'
      } else {
        progressText.value = `AI 正在${statusPrefix}...`
      }
    }, 200)
  }

  function stopProgressSimulation() {
    if (progressTimer) {
      clearInterval(progressTimer)
      progressTimer = null
    }
  }

  function completeProgress(label: string) {
    stopProgressSimulation()
    progress.value = 100
    progressText.value = `${label}完成`
  }

  function errorProgress(label: string) {
    stopProgressSimulation()
    progress.value = 100
    progressText.value = `${label}出错（已保留部分内容）`
  }

  onUnmounted(() => {
    stopProgressSimulation()
    abortController?.abort()
  })

  return {
    loading, progress, progressText,
    MAX_RETRIES,
    getAbortSignal, cancelRequest, cleanupRequest,
    resetRetry, shouldRetry, isCanceled, incrementRetry,
    startProgressSimulation, stopProgressSimulation,
    completeProgress, errorProgress,
  }
}

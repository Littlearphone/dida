import { ref, type Ref } from 'vue'

/**
 * 自动保存 composable
 * @param contentChanged - 内容是否已更改的响应式引用
 * @param getDebounceMs - 返回防抖毫秒数的函数
 * @param save - 实际保存操作的异步函数
 */
export function useAutoSave(
  contentChanged: Ref<boolean>,
  getDebounceMs: () => number,
  save: () => Promise<boolean>,
) {
  const showSavedIndicator = ref(false)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let pollingTimer: ReturnType<typeof setInterval> | null = null
  let indicatorTimer: ReturnType<typeof setTimeout> | null = null

  /** 防抖触发保存 */
  function triggerAutoSave() {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => doSave(), getDebounceMs())
  }

  /** 立即执行保存 */
  async function doSave() {
    if (!contentChanged.value) return
    const ok = await save()
    if (ok) {
      contentChanged.value = false
      showSavedIndicator.value = true
      if (indicatorTimer) clearTimeout(indicatorTimer)
      indicatorTimer = setTimeout(() => { showSavedIndicator.value = false }, 2000)
    }
  }

  /** 启动轮询兜底（5 秒） */
  function startPolling() {
    stopPolling()
    pollingTimer = setInterval(() => {
      if (contentChanged.value) doSave()
    }, 5000)
  }

  function stopPolling() {
    if (pollingTimer) { clearInterval(pollingTimer); pollingTimer = null }
  }

  /** 完全停止 */
  function stop() {
    if (debounceTimer) { clearTimeout(debounceTimer); debounceTimer = null }
    stopPolling()
    if (indicatorTimer) { clearTimeout(indicatorTimer); indicatorTimer = null }
  }

  return {
    showSavedIndicator,
    triggerAutoSave,
    doSave,
    startPolling,
    stop,
  }
}

/**
 * 小说导出 composable
 * 构建纯文本格式并触发浏览器下载
 */
import { stripHtml } from '../utils/editor'
import type { MessageApiInjection } from 'naive-ui/es/message/src/MessageProvider'

/**
 * 小说导出 composable
 * @param novelStore - useNovelStore() 返回值
 */
export function useExport(novelStore: any, message: MessageApiInjection) {
  /** 导出整本小说为纯文本文件 */
  function handleExport() {
    const novel = novelStore.currentNovel
    if (!novel || novelStore.chapters.length === 0) {
      message.warning('没有可导出的内容')
      return
    }

    // 按章节序号排序
    const sorted = [...novelStore.chapters].sort((a, b) => a.order - b.order)

    // 构建纯文本内容
    const lines: string[] = []
    lines.push(novel.title)
    if (novel.author) lines.push(`作者：${novel.author}`)
    if (novel.description) lines.push(`简介：${novel.description}`)
    if (novel.outline) lines.push(`大纲：${novel.outline}`)
    lines.push('')
    lines.push('━'.repeat(48))
    lines.push('')

    for (const ch of sorted) {
      const plain = stripHtml(ch.content)
      lines.push(ch.title || `第${ch.order}章`)
      lines.push('─'.repeat(24))
      lines.push('')
      lines.push(plain)
      lines.push('')
      lines.push('')
    }

    const content = lines.join('\n')

    // 触发浏览器下载
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${novel.title}.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    message.success('导出完成')
  }

  return { handleExport }
}

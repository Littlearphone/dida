/**
 * 编辑器 HTML/文本工具函数
 */
import { createDocument } from '@tiptap/core'

/** 纯文本/混合文本 → Tiptap 兼容的段落 HTML */
export function toTiptapHtml(html: string): string {
  if (!html) return ''
  if (!/<\/?[a-z][\s\S]*?>/i.test(html)) {
    return html.split(/\n\s*\n/).map(s => s.trim()).filter(Boolean).map(s => {
      const withBr = s.replace(/\r?\n/g, '<br>')
      return `<p>${withBr}</p>`
    }).join('')
  }
  let out = html.replace(/<div/gi, '<p').replace(/<\/div\s*>/gi, '</p>')
  if (/(?:<br\s*\/?>\s*){2,}/i.test(out)) {
    const parts = out.split(/(?:<br\s*\/?>\s*){2,}/i).filter(s => s.trim())
    if (parts.length > 1) out = parts.map(s => `<p>${s.trim()}</p>`).join('')
  }
  return out
}

/** 设置编辑器内容并标记已变更 */
export function setEditorContent(editor: any, text: string, onChanged?: () => void) {
  editor.commands.setContent(toTiptapHtml(text))
  onChanged?.()
}

/** HTML → 纯文本（保留段落结构，用于导出/字数统计） */
export function stripHtml(html: string): string {
  return html
    .replace(/<\/p>/gi, '\n\n')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<[^>]+>/g, '')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

/** 计算字数（去标签 + 去空白） */
export function wordCount(html: string): number {
  return stripHtml(html).replace(/\s/g, '').length
}

/** HTML → 纯文本段落（用于 AI 请求前传参，保留分段） */
export function htmlToPlainText(html: string): string {
  if (!/<\/?[a-z][\s\S]*?>/i.test(html)) {
    return normalizeParagraphs(html)
  }
  const div = document.createElement('div')
  div.innerHTML = html
  div.querySelectorAll('p, div, h1, h2, h3, h4, h5, h6, li, blockquote').forEach(el => {
    el.after('\n\n')
  })
  div.querySelectorAll('br').forEach(el => el.replaceWith('\n'))
  return normalizeParagraphs(div.textContent || '')
}

/** 归一化段落换行：单换行 → 双换行（段落分隔） */
export function normalizeParagraphs(text: string): string {
  if (!text) return text
  if (!/\n\n/.test(text) && /\n/.test(text)) {
    return text.split(/\n+/).map(s => s.trim()).filter(Boolean).join('\n\n')
  }
  return text
}

/**
 * 搜索/替换核心逻辑 — 从 SearchReplacePanel 提取
 * 管理 ProseMirror Decoration 高亮、本章/全书搜索替换
 */
import { computed, nextTick, ref } from 'vue'
import { useNovelStore } from '../stores/novel'
import { Plugin, PluginKey, TextSelection } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'

export interface SearchMatch {
  from: number
  to: number
}

export interface AllChapterMatch {
  chapterId: string
  chapterTitle: string
  total: number
  snippets: Array<{ index: number; before: string; match: string; after: string }>
}

export function useSearch(editor: any, doSaveChapter: () => Promise<boolean>) {
  const novelStore = useNovelStore()

  const showSearch = ref(false)
  const showReplace = ref(false)
  const searchQuery = ref('')
  const replaceText = ref('')
  const currentMatchIndex = ref(0)
  const totalMatches = ref(0)
  const searchAll = ref(false)
  const allChapterMatches = ref<AllChapterMatch[]>([])

  const allSearchTotal = computed(() =>
    allChapterMatches.value.reduce((sum, c) => sum + c.total, 0),
  )

  const searchPluginKey = new PluginKey('search-highlight')
  let searchPlugin: Plugin | null = null
  let matches: SearchMatch[] = []

  // === 在 ProseMirror doc 中查找所有匹配位置 ===
  function findMatchesInDoc(doc: any, query: string): SearchMatch[] {
    if (!query) return []
    const results: SearchMatch[] = []
    const lowerQuery = query.toLowerCase()
    doc.descendants((node: any, pos: number) => {
      if (node.isText) {
        const text = node.text || ''
        const lowerText = text.toLowerCase()
        let idx = 0
        while ((idx = lowerText.indexOf(lowerQuery, idx)) !== -1) {
          results.push({ from: pos + idx, to: pos + idx + query.length })
          idx += query.length
        }
      }
      return !node.isText
    })
    return results
  }

  function stripHtml(html: string): string {
    const div = document.createElement('div')
    div.innerHTML = html
    return div.textContent || ''
  }

  function searchAllChapters(query: string): AllChapterMatch[] {
    if (!query) return []
    const lowerQuery = query.toLowerCase()
    const results: AllChapterMatch[] = []
    for (const ch of novelStore.chapters) {
      const text = stripHtml(ch.content)
      const lowerText = text.toLowerCase()
      const snippets: AllChapterMatch['snippets'] = []
      let idx = 0
      while ((idx = lowerText.indexOf(lowerQuery, idx)) !== -1) {
        const start = Math.max(0, idx - 20)
        const end = Math.min(text.length, idx + query.length + 20)
        const before = (start > 0 ? '…' : '') + text.slice(start, idx)
        const match = text.slice(idx, idx + query.length)
        const after = text.slice(idx + query.length, end) + (end < text.length ? '…' : '')
        snippets.push({ index: idx, before, match, after })
        idx += query.length
      }
      if (snippets.length > 0) {
        results.push({
          chapterId: ch.id,
          chapterTitle: ch.title || `第${novelStore.chapters.indexOf(ch) + 1}章`,
          total: snippets.length,
          snippets,
        })
      }
    }
    return results
  }

  /**
   * 将选区滚动到指定匹配位置并更新高亮
   * 合并 setSelection + scrollIntoView + decoration 为单次 dispatch，
   * 避免多事务竞态导致滚动静默失败。
   * view.focus() 必须在 dispatch 之前调用：scrollToSelection 在 dispatch
   * 内部的 updateState 中执行，若编辑器没有 DOM 焦点，selectionToDOM 可能
   * 静默失败，导致 startDOM 位于编辑器外，滚动被忽略。
   */
  function scrollToMatch(idx: number) {
    const ed = editor.value
    if (!ed || idx < 0 || idx >= matches.length) return
    const m = matches[idx]
    const { state, view } = ed
    const tr = state.tr
    tr.setSelection(TextSelection.create(state.doc, m.from, m.to))
    tr.setMeta(searchPluginKey, { matchData: matches, currentIdx: idx })
    tr.scrollIntoView()
    // 先确保编辑器有 DOM 焦点，再派发事务（scrollToSelection 在 dispatch 内部执行）
    view.focus()
    view.dispatch(tr)
  }

  /**
   * 跳转到指定章节的第 matchIndex 个匹配（0-based）
   * @param matchIndex 匹配序号，默认 0（第一个）；点击 snippet 时传入对应的 si 索引
   */
  async function navigateToChapterSearch(chapterId: string, matchIndex: number = 0) {
    const ed = editor.value
    if (!ed) return
    // 限制 matchIndex 不越界（matches 在 updateSearch 后填充）
    const goTo = (idx: number) => {
      if (matches.length === 0) return
      const targetIdx = Math.min(idx, matches.length - 1)
      currentMatchIndex.value = targetIdx + 1
      scrollToMatch(targetIdx)
    }
    if (novelStore.currentChapter?.id === chapterId) {
      updateSearch()
      goTo(matchIndex)
      return
    }
    await doSaveChapter()
    const ch = novelStore.chapters.find(c => c.id === chapterId)
    if (!ch) return
    novelStore.selectChapter(ch)
    // 等待编辑器同步新章节内容（watch 触发 → dispatch → DOM 更新）
    await nextTick()
    await new Promise(r => setTimeout(r, 80))
    // 新文档需要重建高亮插件：先注销旧插件（按 key 清理），再重新注册
    if (editor.value) {
      editor.value.unregisterPlugin(searchPluginKey)
      searchPlugin = null
    }
    updateSearch()
    // 等待编辑器完成布局（requestAnimationFrame 保证 DOM 已渲染），再定位
    await new Promise(r => requestAnimationFrame(r))
    goTo(matchIndex)
  }

  // === Plugin 管理 ===
  function createSearchPlugin() {
    return new Plugin({
      key: searchPluginKey,
      state: {
        init() { return DecorationSet.empty },
        apply(tr, old) {
          const meta = tr.getMeta(searchPluginKey)
          if (!meta) return old.map(tr.mapping, tr.doc)
          if (meta.clear) return DecorationSet.empty
          const { matchData, currentIdx } = meta
          if (!matchData || matchData.length === 0) return DecorationSet.empty
          return DecorationSet.create(
            tr.doc,
            matchData.map((m: SearchMatch, i: number) =>
              Decoration.inline(m.from, m.to, {
                class: i === currentIdx ? 'search-current' : 'search-highlight',
              }),
            ),
          )
        },
      },
      props: {
        decorations(state) { return this.getState(state) },
      },
    })
  }

  function registerPlugin() {
    // 编辑器未就绪时不设置 searchPlugin，留给 updateSearch 懒注册处理
    if (!editor.value) return false
    const plugin = createSearchPlugin()
    editor.value.registerPlugin(plugin)
    searchPlugin = plugin
    return true
  }

  function unregisterPlugin() {
    if (searchPlugin && editor.value) {
      editor.value.unregisterPlugin(searchPluginKey)
      searchPlugin = null
    }
  }

  /**
   * 确保搜索插件已注册（懒注册）
   * 解决 Vue 生命周期中子组件的 onMounted 先于父组件执行，
   * 导致 registerPlugin() 调用时编辑器实例尚未创建的时序问题
   */
  function ensurePluginRegistered(): boolean {
    if (searchPlugin) return true
    return registerPlugin()
  }

  // === 搜索与导航 ===
  function updateSearch() {
    const query = searchQuery.value
    const ed = editor.value
    if (!query || !ed) { clearHighlights(); allChapterMatches.value = []; return }
    // 懒注册：首次搜索时确保插件已注册（编辑器此时必然已就绪）
    if (!ensurePluginRegistered()) return
    matches = findMatchesInDoc(ed.state.doc, query)
    totalMatches.value = matches.length
    currentMatchIndex.value = matches.length > 0 ? 1 : 0
    ed.view.dispatch(
      ed.state.tr.setMeta(searchPluginKey, {
        matchData: matches,
        currentIdx: currentMatchIndex.value - 1,
      }),
    )
    if (searchAll.value) {
      allChapterMatches.value = searchAllChapters(query)
    }
  }

  function clearHighlights() {
    const ed = editor.value
    if (!ed) return
    matches = []
    totalMatches.value = 0
    currentMatchIndex.value = 0
    ed.view.dispatch(ed.state.tr.setMeta(searchPluginKey, { clear: true }))
  }

  /** 跳转到下一个匹配（循环），合并 setSelection + scrollIntoView + decoration 为单次 dispatch */
  function findNext() {
    if (matches.length === 0 || !editor.value) return
    currentMatchIndex.value = (currentMatchIndex.value % matches.length) + 1
    scrollToMatch(currentMatchIndex.value - 1)
  }

  /** 跳转到上一个匹配（循环），合并 setSelection + scrollIntoView + decoration 为单次 dispatch */
  function findPrev() {
    if (matches.length === 0 || !editor.value) return
    currentMatchIndex.value =
      currentMatchIndex.value <= 1 ? matches.length : currentMatchIndex.value - 1
    scrollToMatch(currentMatchIndex.value - 1)
  }

  // === 替换 ===
  function replaceCurrent() {
    const ed = editor.value
    if (!ed || !replaceText.value) return
    const idx = currentMatchIndex.value - 1
    const m = matches[idx]
    if (!m) return
    const { state, view } = ed
    const tr = state.tr.replaceWith(m.from, m.to, state.schema.text(replaceText.value))
    view.dispatch(tr)
    view.focus()
    nextTick(() => updateSearch())
  }

  function replaceAll() {
    const ed = editor.value
    if (!ed || !replaceText.value || matches.length === 0) return
    const { state, view } = ed
    const sorted = [...matches].sort((a, b) => b.from - a.from)
    const tr = state.tr
    for (const m of sorted) {
      tr.replaceWith(m.from, m.to, state.schema.text(replaceText.value))
    }
    view.dispatch(tr)
    view.focus()
    nextTick(() => updateSearch())
  }

  function replaceInHtml(html: string, from: string, to: string): string {
    const div = document.createElement('div')
    div.innerHTML = html
    function walkText(node: Node) {
      if (node.nodeType === Node.TEXT_NODE) {
        const text = node.textContent || ''
        if (text.includes(from)) {
          node.textContent = text.split(from).join(to)
        }
      } else {
        for (let i = 0; i < node.childNodes.length; i++) walkText(node.childNodes[i])
      }
    }
    walkText(div)
    return div.innerHTML
  }

  /** 全书替换（需要 window.confirm 确认） */
  async function replaceAllInBook(): Promise<boolean> {
    if (!replaceText.value || allChapterMatches.value.length === 0) return false
    const query = searchQuery.value
    const totalCount = allSearchTotal.value
    if (!window.confirm(
      `将在全部 ${allChapterMatches.value.length} 个章节中替换「${query}」→「${replaceText.value}」，共 ${totalCount} 处。确认？`,
    )) return false

    let replacedCount = 0
    for (const cm of allChapterMatches.value) {
      const ch = novelStore.chapters.find(c => c.id === cm.chapterId)
      if (!ch) continue
      const newContent = replaceInHtml(ch.content, query, replaceText.value)
      if (newContent !== ch.content) {
        await novelStore.updateChapter(ch.id, { content: newContent })
        replacedCount += cm.total
      }
    }
    const cur = novelStore.currentChapter
    if (cur) {
      const updated = novelStore.chapters.find(c => c.id === cur.id)
      if (updated && updated.content !== cur.content) {
        editor.value?.commands.setContent(updated.content || '')
      }
    }
    // 重新执行搜索以刷新编辑器内的高亮：替换后 query 的匹配数变为 0，高亮自动清除
    // allChapterMatches 由 updateSearch 在 searchAll 模式下同步更新，无需手动赋值
    nextTick(() => updateSearch())
    return true
  }

  // === 面板开关 ===
  function openSearch(initialText?: string) {
    showSearch.value = true
    showReplace.value = false
    if (initialText) {
      searchQuery.value = initialText
      nextTick(() => updateSearch())
    }
  }

  function openReplace(initialText?: string) {
    showSearch.value = true
    showReplace.value = true
    if (initialText) {
      searchQuery.value = initialText
      nextTick(() => updateSearch())
    }
  }

  function closeSearch() {
    showSearch.value = false
    showReplace.value = false
    searchQuery.value = ''
    replaceText.value = ''
    searchAll.value = false
    allChapterMatches.value = []
    clearHighlights()
    editor.value?.commands.focus()
  }

  function fillSearchFromSelection(forcedText?: string) {
    const ed = editor.value
    if (!ed) return
    if (forcedText) {
      searchQuery.value = forcedText
      nextTick(() => updateSearch())
      return
    }
    const { from, to } = ed.state.selection
    if (from !== to) {
      searchQuery.value = ed.state.doc.textBetween(from, to).slice(0, 200)
      nextTick(() => updateSearch())
    }
  }

  return {
    // 状态
    showSearch, showReplace, searchQuery, replaceText,
    currentMatchIndex, totalMatches, searchAll,
    allChapterMatches, allSearchTotal,
    // 操作
    updateSearch, clearHighlights, findNext, findPrev,
    replaceCurrent, replaceAll, replaceAllInBook,
    openSearch, openReplace, closeSearch, fillSearchFromSelection,
    navigateToChapterSearch,
    // 插件
    registerPlugin, unregisterPlugin, searchPluginKey,
    // 内部暴露（供键盘事件使用）
    isOpen: () => showSearch.value,
    isReplaceOpen: () => showReplace.value,
  }
}

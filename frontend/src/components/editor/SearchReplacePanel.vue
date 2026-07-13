<script setup lang="ts">
/**
 * 搜索/替换面板 — 从 NovelEditor 提取
 * 管理 ProseMirror Decoration 高亮、本章搜索/替换、全书搜索/替换
 */
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useNovelStore } from '../../stores/novel'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import { NButton, NIcon, NInput, NText, useMessage } from 'naive-ui'
import {
  ChevronUpOutline as PrevIcon,
  ChevronDownOutline as NextIcon,
  CloseOutline as CloseIcon,
} from '@vicons/ionicons5'

const props = defineProps<{
  editor: any // Tiptap Editor 实例
  doSaveChapter: () => Promise<boolean>
}>()

const novelStore = useNovelStore()
const message = useMessage()

// === 搜索/替换状态 ===
const showSearch = ref(false)
const showReplace = ref(false)
const searchQuery = ref('')
const replaceText = ref('')
const currentMatchIndex = ref(0)
const totalMatches = ref(0)
const searchAll = ref(false)

/** 全书搜索结果：按章节分组 */
interface AllChapterMatch {
  chapterId: string
  chapterTitle: string
  total: number
  snippets: Array<{ index: number; before: string; match: string; after: string }>
}
const allChapterMatches = ref<AllChapterMatch[]>([])
const allSearchTotal = computed(() =>
  allChapterMatches.value.reduce((sum, c) => sum + c.total, 0),
)

const searchPluginKey = new PluginKey('search-highlight')
let searchPlugin: Plugin | null = null
let matches: Array<{ from: number; to: number }> = []

// === 搜索核心逻辑 ===

/** 遍历 ProseMirror doc 找到所有匹配位置 */
function findMatchesInDoc(doc: any, query: string): Array<{ from: number; to: number }> {
  if (!query) return []
  const results: Array<{ from: number; to: number }> = []
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

/** 从 HTML 中提取纯文本 */
function stripHtml(html: string): string {
  const div = document.createElement('div')
  div.innerHTML = html
  return div.textContent || ''
}

/** 全书搜索：遍历所有章节内容查找匹配 */
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

/** 切换到某章节并执行搜索定位 */
async function navigateToChapterSearch(chapterId: string) {
  const ed = props.editor
  if (!ed) return

  // 如果已在目标章节，搜索当前内容
  if (novelStore.currentChapter?.id === chapterId) {
    updateSearch()
    if (matches.length > 0) findNext()
    return
  }
  // 不同章节：先保存再切换
  await props.doSaveChapter()
  const ch = novelStore.chapters.find(c => c.id === chapterId)
  if (!ch) return
  novelStore.selectChapter(ch)
  await nextTick()
  updateSearch()
  if (matches.length > 0) findNext()
}

/** 创建 Decoration 插件 */
function createSearchPluginInst() {
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
          matchData.map((m: { from: number; to: number }, i: number) =>
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

/** 扫描并高亮（本章 + 全书同步） */
function updateSearch() {
  const query = searchQuery.value
  const ed = props.editor
  if (!query || !ed) { clearSearchHighlights(); allChapterMatches.value = []; return }

  // 本章搜索
  matches = findMatchesInDoc(ed.state.doc, query)
  totalMatches.value = matches.length
  currentMatchIndex.value = matches.length > 0 ? 1 : 0
  ed.view.dispatch(
    ed.state.tr.setMeta(searchPluginKey, {
      matchData: matches,
      currentIdx: currentMatchIndex.value - 1,
    }),
  )

  // 全书搜索结果
  if (searchAll.value) {
    allChapterMatches.value = searchAllChapters(query)
  }
}

/** 清除高亮 */
function clearSearchHighlights() {
  const ed = props.editor
  if (!ed) return
  matches = []
  totalMatches.value = 0
  currentMatchIndex.value = 0
  ed.view.dispatch(
    ed.state.tr.setMeta(searchPluginKey, { clear: true }),
  )
}

/** 下一个匹配 */
function findNext() {
  if (matches.length === 0 || !props.editor) return
  currentMatchIndex.value = (currentMatchIndex.value % matches.length) + 1
  const idx = currentMatchIndex.value - 1
  const m = matches[idx]
  props.editor.commands.setTextSelection({ from: m.from, to: m.to })
  props.editor.commands.scrollIntoView()
  props.editor.view.dispatch(
    props.editor.state.tr.setMeta(searchPluginKey, { matchData: matches, currentIdx: idx }),
  )
}

/** 上一个匹配 */
function findPrev() {
  if (matches.length === 0 || !props.editor) return
  currentMatchIndex.value =
    currentMatchIndex.value <= 1 ? matches.length : currentMatchIndex.value - 1
  const idx = currentMatchIndex.value - 1
  const m = matches[idx]
  props.editor.commands.setTextSelection({ from: m.from, to: m.to })
  props.editor.commands.scrollIntoView()
  props.editor.view.dispatch(
    props.editor.state.tr.setMeta(searchPluginKey, { matchData: matches, currentIdx: idx }),
  )
}

/** 替换当前匹配 */
function replaceCurrent() {
  const ed = props.editor
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

/** 全部替换（本章） */
function replaceAll() {
  const ed = props.editor
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

/** 在 HTML 中做纯文本替换（只替换标签外的文本） */
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
      for (let i = 0; i < node.childNodes.length; i++) {
        walkText(node.childNodes[i])
      }
    }
  }
  walkText(div)
  return div.innerHTML
}

/** 全书替换 */
async function replaceAllInBook() {
  if (!replaceText.value || allChapterMatches.value.length === 0) return
  const query = searchQuery.value
  const totalCount = allSearchTotal.value
  if (!window.confirm(`将在全部 ${allChapterMatches.value.length} 个章节中替换「${query}」→「${replaceText.value}」，共 ${totalCount} 处。确认？`)) return

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
  // 如果当前章节内容变了，同步到编辑器
  const cur = novelStore.currentChapter
  if (cur) {
    const updated = novelStore.chapters.find(c => c.id === cur.id)
    if (updated && updated.content !== cur.content) {
      props.editor?.commands.setContent(updated.content || '')
    }
  }
  // 刷新搜索结果
  allChapterMatches.value = searchAllChapters(query)
  message.success(`全书替换完成，共处理 ${replacedCount} 处`)
}

/** 关闭搜索 */
function closeSearch() {
  showSearch.value = false
  showReplace.value = false
  searchQuery.value = ''
  replaceText.value = ''
  searchAll.value = false
  allChapterMatches.value = []
  clearSearchHighlights()
  props.editor?.commands.focus()
}

/** 从编辑器选中文本填充搜索框 */
function fillSearchFromSelection(forcedText?: string) {
  const ed = props.editor
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

/** 打开搜索栏 */
function openSearch(initialText?: string) {
  showSearch.value = true
  showReplace.value = false
  if (initialText) {
    searchQuery.value = initialText
    nextTick(() => updateSearch())
  }
  nextTick(() => (document.querySelector('.search-input input') as HTMLInputElement)?.focus())
}

/** 打开替换栏 */
function openReplace(initialText?: string) {
  showSearch.value = true
  showReplace.value = true
  if (initialText) {
    searchQuery.value = initialText
    nextTick(() => updateSearch())
  }
  nextTick(() => (document.querySelector('.replace-input input') as HTMLInputElement)?.focus())
}

/** 搜索框内键盘快捷键 */
function handleKeydown(e: KeyboardEvent) {
  if (!showSearch.value) return
  const isCtrl = e.ctrlKey || e.metaKey

  // F3 / Ctrl+G / Enter（搜索框内）→ 下一个 / Shift+上一个
  if (e.key === 'F3' || (isCtrl && e.key === 'g') ||
      (e.key === 'Enter' && (e.target as HTMLElement).closest('.search-bar'))) {
    e.preventDefault()
    if (e.shiftKey) findPrev(); else findNext()
    return
  }

  // Escape → 关闭搜索
  if (e.key === 'Escape') { closeSearch(); e.preventDefault(); return }
}

// === 暴露给父组件的方法 ===
defineExpose({
  isOpen: () => showSearch.value,
  isReplaceOpen: () => showReplace.value,
  openSearch,
  openReplace,
  closeSearch,
  findNext,
  findPrev,
  fillSearchFromSelection,
})

// === 生命周期 ===
onMounted(() => {
  searchPlugin = createSearchPluginInst()
  if (props.editor) {
    props.editor.registerPlugin(searchPlugin)
  }
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  if (searchPlugin && props.editor) {
    props.editor.unregisterPlugin(searchPluginKey)
  }
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <!-- 搜索 / 替换栏 -->
  <div v-if="showSearch" class="search-bar">
    <div class="search-bar-inner">
      <div class="search-row">
        <!-- 范围切换：本章 / 全书 -->
        <div class="search-scope">
          <span class="scope-btn" :class="{ active: !searchAll }" @click="searchAll = false; updateSearch()">本章</span>
          <span class="scope-divider">|</span>
          <span class="scope-btn" :class="{ active: searchAll }" @click="searchAll = true; updateSearch()">全书</span>
        </div>
        <n-input :value="searchQuery" placeholder="搜索正文..." size="small"
          class="search-input" style="width:200px"
          @update:value="(v: string) => { searchQuery = v; updateSearch() }"
          @keydown.enter="findNext()" />

        <n-button quaternary size="tiny" :disabled="totalMatches === 0" @click="findPrev" title="上一个 (Shift+F3)">
          <template #icon><n-icon size="14"><PrevIcon/></n-icon></template>
        </n-button>
        <n-button quaternary size="tiny" :disabled="totalMatches === 0" @click="findNext" title="下一个 (F3)">
          <template #icon><n-icon size="14"><NextIcon/></n-icon></template>
        </n-button>

        <n-text v-if="totalMatches > 0" class="match-counter">{{ currentMatchIndex }}/{{ totalMatches }}</n-text>
        <template v-if="searchAll && allSearchTotal > 0">
          <n-text depth="3" class="match-counter" style="margin-left:0">（全 {{ allSearchTotal }}）</n-text>
        </template>
        <n-text v-if="totalMatches === 0 && !(searchAll && allSearchTotal > 0)" depth="3" class="match-counter">无结果</n-text>

        <n-button quaternary size="tiny" class="search-close" @click="closeSearch" title="关闭搜索 (Esc)">
          <template #icon><n-icon size="14"><CloseIcon/></n-icon></template>
        </n-button>
      </div>

      <div v-if="showReplace" class="search-row">
        <span class="search-scope-placeholder" aria-hidden="true">
          <span class="scope-btn">本章</span>
          <span class="scope-divider">|</span>
          <span class="scope-btn">全书</span>
        </span>
        <n-input :value="replaceText" placeholder="替换为..." size="small"
          class="replace-input" style="width:200px"
          @update:value="(v: string) => replaceText = v" />

        <template v-if="!searchAll">
          <n-button size="tiny" :disabled="totalMatches === 0 || !replaceText" @click="replaceCurrent">替换</n-button>
          <n-button size="tiny" :disabled="totalMatches === 0 || !replaceText" @click="replaceAll">全部替换</n-button>
        </template>
        <template v-else>
          <n-button size="tiny" :disabled="allSearchTotal === 0 || !replaceText" @click="replaceAllInBook">全书替换</n-button>
        </template>
      </div>

      <!-- 全书搜索结果列表 -->
      <div v-if="searchAll && allChapterMatches.length > 0" class="all-search-results">
        <div v-for="cm in allChapterMatches" :key="cm.chapterId" class="all-search-chapter">
          <div class="all-search-chapter-title" @click="navigateToChapterSearch(cm.chapterId)">
            {{ cm.chapterTitle }}（{{ cm.total }} 处）
          </div>
          <div v-for="(s, si) in cm.snippets.slice(0, 5)" :key="si"
            class="all-search-snippet" @click="navigateToChapterSearch(cm.chapterId)">
            <span class="snippet-before">{{ s.before }}</span>
            <span class="snippet-match">{{ s.match }}</span>
            <span class="snippet-after">{{ s.after }}</span>
          </div>
          <div v-if="cm.snippets.length > 5" class="all-search-more">还有 {{ cm.snippets.length - 5 }} 处…</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.search-bar {
  background: #fafafa;
  border-bottom: 1px solid #eee;
  padding: 6px 16px;
  flex-shrink: 0;
  display: flex; justify-content: center;
}
.search-bar-inner {
  max-width: 960px;
  width: 100%;
  padding: 0 64px;
  display: flex; flex-direction: column; gap: 6px;
}
.search-row {
  display: flex; align-items: center; gap: 6px;
  flex-wrap: wrap;
}
.match-counter {
  font-size: 12px; min-width: 44px;
  text-align: center; white-space: nowrap;
}
.search-close {
  margin-left: 4px;
}
.search-scope {
  display: flex; align-items: center; gap: 4px;
  font-size: 12px; color: #888; user-select: none;
  flex-shrink: 0;
}
.scope-btn {
  cursor: pointer; padding: 2px 6px; border-radius: 3px;
  transition: all 0.15s;
}
.scope-btn:hover { background: #eee; }
.scope-btn.active { color: #2080f0; font-weight: 600; background: #e8f4ff; }
.scope-divider { color: #ddd; }
.search-scope-placeholder {
  display: flex; align-items: center; gap: 4px;
  font-size: 12px; flex-shrink: 0;
  visibility: hidden; pointer-events: none;
}
.all-search-results {
  max-height: 240px; overflow-y: auto;
  border-top: 1px solid #eee; padding-top: 8px;
  display: flex; flex-direction: column; gap: 8px;
}
.all-search-chapter-title {
  font-size: 13px; font-weight: 600; color: #333;
  cursor: pointer; padding: 4px 6px; border-radius: 4px;
  margin-bottom: 2px;
}
.all-search-chapter-title:hover { background: #e8f4ff; color: #2080f0; }
.all-search-snippet {
  font-size: 12px; color: #666; cursor: pointer;
  padding: 3px 8px; border-radius: 3px; line-height: 1.5;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.all-search-snippet:hover { background: #f5f5f5; }
.snippet-before, .snippet-after { color: #999; }
.snippet-match { color: #d03050; font-weight: 600; background: #fff0f0; border-radius: 2px; padding: 0 1px; }
.all-search-more { font-size: 11px; color: #999; padding: 2px 8px; }
</style>

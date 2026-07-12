<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, provide, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useNovelStore } from '../stores/novel'
import { useSettingsStore } from '../stores/settings'
import { EDITOR_ACTIONS_KEY } from '../types/editor'
import type { ExtractionResult } from '../types'
import { useEditorAppearance } from '../composables/useEditorAppearance'
import { useAutoSave } from '../composables/useAutoSave'
import { setWindowTitle } from '../utils/windowTitle'
import * as aiApi from '../api/ai'
import ChapterSidebar from '../components/editor/ChapterSidebar.vue'
import EditorToolbar from '../components/editor/EditorToolbar.vue'
import EditorStatusBar from '../components/editor/EditorStatusBar.vue'
import AIContinueDialog from '../components/editor/AIContinueDialog.vue'
import AIEditDialog from '../components/editor/AIEditDialog.vue'
import AISetupDialog from '../components/editor/AISetupDialog.vue'
import NovelInfoDialog from '../components/editor/NovelInfoDialog.vue'
import ExtractResultDialog from '../components/editor/ExtractResultDialog.vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { createDocument } from '@tiptap/core'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import { DOMSerializer } from '@tiptap/pm/model'
import StarterKit from '@tiptap/starter-kit'
import { NLayout, NText, NInput, NButton, NIcon, useMessage } from 'naive-ui'
import {
  ChevronUpOutline as PrevIcon,
  ChevronDownOutline as NextIcon,
  CloseOutline as CloseIcon,
  CutOutline as SplitIcon,
} from '@vicons/ionicons5'

const route = useRoute()
const router = useRouter()
const novelStore = useNovelStore()
const settingsStore = useSettingsStore()

const message = useMessage()
const novelId = route.params.novelId as string
const currentChapter = computed(() => novelStore.currentChapter)
const siderCollapsed = ref(window.innerWidth < 960)

// === Tiptap 编辑器 ===
const editContent = ref('')
const contentChanged = ref(false)
/** 用户是否有过实际编辑（用于判断后退是否可用，避免初始加载后后退误亮） */
const hasUserEdited = ref(false)
const editor = useEditor({
  content: '',
  extensions: [StarterKit],
  editorProps: {
    // 为搜索导航留出顶部间距，避免被搜索栏遮挡
    scrollMargin: { top: 80, right: 0, bottom: 40, left: 0 },
  },
  onUpdate: ({ editor: ed, transaction }) => {
    editContent.value = ed.getHTML()
    if (transaction.docChanged) hasUserEdited.value = true
    if (!contentChanged.value) contentChanged.value = true
    triggerAutoSave()
  },
})

function undo() {
  const ed = editor.value
  if (!ed) return
  // 直接用 commands.undo()，避免 chain().focus() 产生多余事务干扰历史追踪
  ed.commands.undo()
  ed.view.focus()
}
function redo() {
  const ed = editor.value
  if (!ed) return
  ed.commands.redo()
  ed.view.focus()
}
const undoable = computed(() => hasUserEdited.value && (editor.value?.can().undo() ?? false))
const redoable = computed(() => editor.value?.can().redo() ?? false)

// === 提供编辑器操作给 AI 弹框 ===
provide(EDITOR_ACTIONS_KEY, {
  setContent: (html: string) => setEditorContent(html),
  getContent: () => editContent.value,
  markChanged: () => { contentChanged.value = true },
  getSelectionText: () => {
    if (!editor.value) return ''
    const { from, to } = editor.value.state.selection
    if (from === to) return ''
    return editor.value.state.doc.textBetween(from, to)
  },
  replaceSelection: (text: string) => {
    if (!editor.value) return
    const { state, view } = editor.value
    const { from, to } = state.selection
    if (from === to) {
      // 无选区时在文档末尾追加，用 insertContentAt 解析 HTML 保留段落
      editor.value.commands.insertContentAt(state.doc.content.size, toTiptapHtml(text))
    } else {
      // 有选区时替换，用 insertContentAt 解析 HTML 保留换行分段
      editor.value.commands.insertContentAt({ from, to }, toTiptapHtml(text))
    }
    view.focus()
    contentChanged.value = true
  },
  appendContent: (text: string) => {
    if (!editor.value) return
    // 将纯文本转换为段落 HTML 后在文档末尾插入，保留分段
    const html = toTiptapHtml(text)
    editor.value.commands.insertContentAt(editor.value.state.doc.content.size, html)
    contentChanged.value = true
  },
  /** AI 内容插入后自动提取元数据 */
  triggerExtract: () => { nextTick(() => handlePostInsertExtract()) },
})

/** AI 续写内容插入后自动提取元数据 */
async function handlePostInsertExtract() {
  const n = novelStore.currentNovel
  if (!currentChapter.value || !n || !settingsStore.settings?.aiConfigured) return

  // 先保存当前章节确保内容是最新的
  if (contentChanged.value) await doSaveChapter()
  // 保存后重新获取章节引用（doSaveChapter 会替换 currentChapter 对象）
  const ch = currentChapter.value
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
    // 静默失败：提取是辅助功能，不影响主流程，但至少提示用户
    console.warn('AI 提取元数据失败:', e instanceof Error ? e.message : e)
    message.warning('AI 元数据提取未完成，不影响已有内容')
  } finally {
    extractLoading.value = false
  }
}

/** 手动提取/补充元数据：选中文本（若有）或整章内容 → AI 分析 → ExtractResultDialog */
async function handleExtractSupplement() {
  const ch = currentChapter.value
  const n = novelStore.currentNovel
  if (!ch || !n) { message.warning('没有可分析的章节内容'); return }

  // 优先取编辑器选区文本，无选区时用整章内容
  let contentToExtract: string
  if (editor.value) {
    const { from, to } = editor.value.state.selection
    contentToExtract = from !== to ? editor.value.state.doc.textBetween(from, to) : ch.content
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
    // 后端 JSON 解析错误通常是 AI 返回格式不符合要求，给更友好的提示
    if (msg.includes('JSON') || msg.includes('json') || msg.includes('格式')) {
      message.error('AI 返回格式异常，请稍后重试或检查内容是否过长')
    } else {
      message.error('提取失败: ' + msg)
    }
  } finally {
    extractLoading.value = false
  }
}

// === 外观 ===
const {
  fontSize, lineHeight, paragraphSpacing, isBold, isItalic, fontFamily,
  fontOptions, editorStyles, initFromSettings,
} = useEditorAppearance()

// === 自动保存 ===
async function doSaveChapter() {
  if (!currentChapter.value || !contentChanged.value) return false
  const content = editor.value?.getHTML() || ''
  return await novelStore.updateChapter(currentChapter.value.id, { content })
}
const { showSavedIndicator, triggerAutoSave, doSave, startPolling, stop: stopAutoSave } = useAutoSave(
  contentChanged,
  () => settingsStore.settings?.autoSaveMs || 2000,
  doSaveChapter,
)

// === HTML 转换 ===
function toTiptapHtml(html: string): string {
  if (!html) return ''
  if (!/<\/?[a-z][\s\S]*?>/i.test(html)) {
    // 纯文本：按双换行（\n\n）划分段落，段落内单换行转 <br>
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

function setEditorContent(text: string) {
  editContent.value = text
  contentChanged.value = true
  editor.value?.commands.setContent(toTiptapHtml(text))
}

// === 切换章节时同步编辑器（初始化时不执行） ===
const isInitializing = ref(true)
watch(currentChapter, (ch, oldCh) => {
  if (ch && editor.value && !isInitializing.value) {
    // 同一章节（保存触发的引用更新），不重置编辑器内容和历史，只更新 editContent
    if (oldCh && ch.id === oldCh.id) {
      editContent.value = ch.content
      return
    }
    // 真正的章节切换，用 addToHistory=false 的事务设内容，避免产生 undo 记录
    const { state, view, schema } = editor.value
    const html = toTiptapHtml(ch.content)
    const doc = createDocument(html, schema)
    const tr = state.tr.replaceWith(0, state.doc.content.size, doc)
    tr.setMeta('addToHistory', false)
    view.dispatch(tr)
    editContent.value = ch.content
    contentChanged.value = false
    hasUserEdited.value = false
  }
})

// === 返回小说列表 ===
function goBack() {
  // 退出时恢复默认窗口标题
  setWindowTitle('AI 小说编辑器')
  const go = () => { novelStore.selectNovel(null); router.push({ name: 'NovelList' }) }
  if (contentChanged.value) doSave().then(go)
  else go()
}

// === 字数统计 ===
const wordCount = computed(() =>
  (editContent.value.replace(/<[^>]*>/g, '').replace(/\s/g, '')).length,
)

// === 格式化工具有多余空段落时可用 ===
const canFormat = computed(() => {
  const html = editContent.value
  if (!html) return false
  // HTML 中有空段落（<p></p> 或 <p><br></p>）且至少有一个非空段落
  return /<p>\s*<\/p>|<p>\s*<br>\s*<\/p>/i.test(html) && /<p>[^<]+<\/p>/i.test(html)
})

function formatContent() {
  // 删除所有空段落（保留非空段落）
  const formatted = editContent.value
    .replace(/<p>\s*<\/p>/gi, '')
    .replace(/<p>\s*<br\s*\/?>\s*<\/p>/gi, '')
  if (formatted !== editContent.value) setEditorContent(formatted)
}

// === 拆分章节（选中文本拆出为新章） ===
/** 编辑器是否包含非空选区 */
const hasSelection = computed(() => {
  const ed = editor.value
  if (!ed) return false
  const { from, to } = ed.state.selection
  return from !== to
})

/** 获取选区 HTML 内容 */
function getSelectedHtml(): string {
  const ed = editor.value
  if (!ed) return ''
  const { state } = ed
  const { from, to } = state.selection
  if (from === to) return ''
  const slice = state.doc.slice(from, to)
  const serializer = DOMSerializer.fromSchema(state.schema)
  const fragment = serializer.serializeFragment(slice.content)
  const tempDiv = document.createElement('div')
  tempDiv.appendChild(fragment)
  return tempDiv.innerHTML
}

// 拆分章节弹框状态
const showSplitDialog = ref(false)
const splitChapterTitle = ref('')
const splittingChapter = ref(false)

/** 点击拆分按钮：打开弹框让用户输入章节名 */
function handleSplitClick() {
  const ed = editor.value
  const ch = currentChapter.value
  if (!ed || !ch) return

  const selectedHtml = getSelectedHtml()
  if (!selectedHtml) {
    message.warning('请先在正文中选择要拆分的文本')
    return
  }

  // 预设章节名：基于当前章节名
  splitChapterTitle.value = `${ch.title || '新章'}（拆出）`
  showSplitDialog.value = true
}

/** 确认拆分：从当前章节删除选中内容，创建新章节 */
async function confirmSplit() {
  const ed = editor.value
  const ch = currentChapter.value
  const n = novelStore.currentNovel
  if (!ed || !ch || !n) return

  const title = splitChapterTitle.value.trim()
  if (!title) { message.warning('请输入章节标题'); return }

  const selectedHtml = getSelectedHtml()
  if (!selectedHtml) { message.warning('请先选择要拆分的文本'); return }

  splittingChapter.value = true

  // 1. 从当前章节删除选中内容
  const { state, view } = ed
  const tr = state.tr.deleteSelection()
  view.dispatch(tr)
  view.focus()
  contentChanged.value = true
  await doSaveChapter()

  // 2. 创建新章节，临时放在末尾
  const newCh = await novelStore.createChapter({
    novelId: n.id,
    title,
    content: selectedHtml,
    order: novelStore.chapters.length + 1,
  })
  if (!newCh) { message.error('创建章节失败'); splittingChapter.value = false; return }

  // 3. 重新排序：将新章节放到当前章节之后
  const currentIdx = novelStore.chapters.findIndex(c => c.id === ch.id)
  const ids = novelStore.chapters.filter(c => c.id !== newCh.id).map(c => c.id)
  ids.splice(currentIdx + 1, 0, newCh.id)
  await novelStore.reorderChapters(n.id, ids)

  // 4. 重新加载章节列表并跳转到新章节
  await novelStore.loadChapters(n.id)
  const found = novelStore.chapters.find(c => c.id === newCh.id)
  if (found) novelStore.selectChapter(found)
  showSplitDialog.value = false
  splittingChapter.value = false
  message.success('已拆分为新章节')
}

/** 取消拆分 */
function cancelSplit() {
  showSplitDialog.value = false
  splitChapterTitle.value = ''
}

// === 搜索 / 替换（ProseMirror Decoration 高亮 + 全书搜索） ===
const showSearch = ref(false)
const showReplace = ref(false)
const searchQuery = ref('')
const replaceText = ref('')
const currentMatchIndex = ref(0)
const totalMatches = ref(0)
/** 全书搜索模式 */
const searchAll = ref(false)

/** 全书搜索结果：按章节分组 */
interface AllChapterMatch {
  chapterId: string
  chapterTitle: string
  /** 本章匹配总数 */
  total: number
  /** 每条匹配的上下文（前后各取一段纯文本） */
  snippets: Array<{ index: number; before: string; match: string; after: string }>
}
const allChapterMatches = ref<AllChapterMatch[]>([])
/** 全书搜索结果的最高计数（用于排序显示） */
const allSearchTotal = computed(() =>
  allChapterMatches.value.reduce((sum, c) => sum + c.total, 0),
)

const searchPluginKey = new PluginKey('search-highlight')
let searchPlugin: Plugin | null = null
let matches: Array<{ from: number; to: number }> = []

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

/** 从 HTML 中提取纯文本（去标签、解码实体） */
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
      // 取匹配前后各 20 个字符作为上下文
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
  // 如果已在目标章节，直接搜索
  if (currentChapter.value?.id === chapterId) {
    updateSearch()
    if (matches.length > 0) {
      findNext()
      return
    }
    // 编辑器内容可能已变更导致匹配消失 → 用已保存内容覆盖编辑器再搜
    const ch = novelStore.chapters.find(c => c.id === chapterId)
    if (ch && ch.content !== editContent.value) {
      setEditorContent(ch.content)
      await nextTick()
      updateSearch()
      if (matches.length > 0) findNext()
    }
    return
  }
  // 先保存当前章节
  if (contentChanged.value) await doSaveChapter()
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
        // 没有 search meta 时，将旧装饰器映射通过事务 → 保持高亮不因光标/点击清除
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
  if (!query) { clearSearchHighlights(); allChapterMatches.value = []; return }

  // 本章搜索（高亮）
  if (editor.value) {
    matches = findMatchesInDoc(editor.value.state.doc, query)
    totalMatches.value = matches.length
    currentMatchIndex.value = matches.length > 0 ? 1 : 0
    editor.value.view.dispatch(
      editor.value.state.tr.setMeta(searchPluginKey, {
        matchData: matches,
        currentIdx: currentMatchIndex.value - 1,
      }),
    )
  }

  // 全书搜索结果（始终同步，切换搜索模式时无需重新请求）
  if (searchAll.value) {
    allChapterMatches.value = searchAllChapters(query)
  }
}

/** 清除高亮 */
function clearSearchHighlights() {
  if (!editor.value) return
  matches = []
  totalMatches.value = 0
  currentMatchIndex.value = 0
  editor.value.view.dispatch(
    editor.value.state.tr.setMeta(searchPluginKey, { clear: true }),
  )
}

/** 下一个匹配 */
function findNext() {
  if (matches.length === 0 || !editor.value) return
  currentMatchIndex.value = (currentMatchIndex.value % matches.length) + 1
  const idx = currentMatchIndex.value - 1
  const m = matches[idx]
  editor.value.commands.setTextSelection({ from: m.from, to: m.to })
  editor.value.commands.scrollIntoView()
  editor.value.view.dispatch(
    editor.value.state.tr.setMeta(searchPluginKey, { matchData: matches, currentIdx: idx }),
  )
}

/** 上一个匹配 */
function findPrev() {
  if (matches.length === 0 || !editor.value) return
  currentMatchIndex.value =
    currentMatchIndex.value <= 1 ? matches.length : currentMatchIndex.value - 1
  const idx = currentMatchIndex.value - 1
  const m = matches[idx]
  editor.value.commands.setTextSelection({ from: m.from, to: m.to })
  editor.value.commands.scrollIntoView()
  editor.value.view.dispatch(
    editor.value.state.tr.setMeta(searchPluginKey, { matchData: matches, currentIdx: idx }),
  )
}

/** 替换当前：用单次 transaction 替换匹配文本，避免 chain().focus() 产生多余事务 */
function replaceCurrent() {
  if (!editor.value || !replaceText.value) return
  const idx = currentMatchIndex.value - 1
  const m = matches[idx]
  if (!m) return
  const { state, view } = editor.value
  const tr = state.tr.replaceWith(m.from, m.to, state.schema.text(replaceText.value))
  view.dispatch(tr)
  view.focus()
  nextTick(() => updateSearch())
}

/** 全部替换：从后往前替换，单次 transaction 完成全部操作 */
function replaceAll() {
  if (!editor.value || !replaceText.value || matches.length === 0) return
  const { state, view } = editor.value
  const sorted = [...matches].sort((a, b) => b.from - a.from)
  const tr = state.tr
  for (const m of sorted) {
    tr.replaceWith(m.from, m.to, state.schema.text(replaceText.value))
  }
  view.dispatch(tr)
  view.focus()
  nextTick(() => updateSearch())
}

/** 全书替换：遍历所有章节执行替换 */
async function replaceAllInBook() {
  if (!replaceText.value || allChapterMatches.value.length === 0) return
  const query = searchQuery.value
  // 确认弹框
  if (!window.confirm(`将在全部 ${allChapterMatches.value.length} 个章节中替换「${query}」→「${replaceText.value}」，共 ${allSearchTotal.value} 处。确认？`)) return

  let replacedCount = 0
  for (const cm of allChapterMatches.value) {
    const ch = novelStore.chapters.find(c => c.id === cm.chapterId)
    if (!ch) continue
    // 在 HTML 文本层面做替换
    const newContent = replaceInHtml(ch.content, query, replaceText.value)
    if (newContent !== ch.content) {
      await novelStore.updateChapter(ch.id, { content: newContent })
      replacedCount += cm.total
    }
  }
  // 如果当前章节内容变了，同步到编辑器
  const cur = currentChapter.value
  if (cur && novelStore.chapters.find(c => c.id === cur.id)?.content !== cur.content) {
    const updated = novelStore.chapters.find(c => c.id === cur.id)
    if (updated) {
      setEditorContent(updated.content || '')
      contentChanged.value = true
    }
  }
  // 刷新搜索结果
  allChapterMatches.value = searchAllChapters(query)
  message.success(`全书替换完成，共处理 ${replacedCount} 处`)
}

/** 在 HTML 中做纯文本替换（只替换标签外的文本） */
function replaceInHtml(html: string, from: string, to: string): string {
  // 使用 DOM 解析替换，避免破坏 HTML 结构
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

/** 关闭搜索 */
function closeSearch() {
  showSearch.value = false
  showReplace.value = false
  searchQuery.value = ''
  replaceText.value = ''
  searchAll.value = false
  allChapterMatches.value = []
  clearSearchHighlights()
  editor.value?.commands.focus()
}

/** 从编辑器选中文本填充搜索框 */
function fillSearchFromSelection() {
  if (!editor.value) return
  const { from, to } = editor.value.state.selection
  if (from !== to) {
    searchQuery.value = editor.value.state.doc.textBetween(from, to).slice(0, 200)
    nextTick(() => updateSearch())
  }
}

// === 键盘快捷键 ===
function handleKeydown(e: KeyboardEvent) {
  const isCtrl = e.ctrlKey || e.metaKey

  // Ctrl+F：开关搜索栏（不影响替换栏）
  if (isCtrl && e.key === 'f' && !e.shiftKey) {
    e.preventDefault()
    if (showSearch.value) {
      // 有选中文本时填入搜索框而不再关闭
      if (editor.value) {
        const { from, to } = editor.value.state.selection
        if (from !== to) { fillSearchFromSelection(); return }
      }
      closeSearch(); return
    }
    showSearch.value = true
    // 自动将选中文本填入搜索框
    fillSearchFromSelection()
    nextTick(() => (document.querySelector('.search-input input') as HTMLInputElement)?.focus())
    return
  }

  // Ctrl+H：开关替换栏（自动补开搜索栏，关闭时保留搜索栏）
  if (isCtrl && e.key === 'h') {
    e.preventDefault()
    if (showReplace.value) {
      // 有选中文本时填入搜索框而不再关闭替换栏
      if (editor.value) {
        const { from, to } = editor.value.state.selection
        if (from !== to) { fillSearchFromSelection(); return }
      }
      showReplace.value = false; return
    }
    showSearch.value = true
    showReplace.value = true
    // 自动将选中文本填入搜索框
    fillSearchFromSelection()
    nextTick(() => (document.querySelector('.replace-input input') as HTMLInputElement)?.focus())
    return
  }

  // Ctrl+Shift+F：格式化
  if (isCtrl && e.shiftKey && e.key === 'F') {
    e.preventDefault()
    if (canFormat.value) formatContent()
    return
  }

  // Ctrl+B：全局粗体
  if (isCtrl && e.key === 'b') {
    e.preventDefault()
    isBold.value = !isBold.value
    return
  }

  // Ctrl+I：全局斜体
  if (isCtrl && e.key === 'i') {
    e.preventDefault()
    isItalic.value = !isItalic.value
    return
  }

  // Ctrl+S：手动保存当前章节
  if (isCtrl && e.key === 's') {
    e.preventDefault()
    doSave()
    return
  }

  if (!showSearch.value) return

  // F3 / Ctrl+G / Enter（搜索框内）：下一个 / Shift+上一个
  if (e.key === 'F3' || (isCtrl && e.key === 'g') || (e.key === 'Enter' && (e.target as HTMLElement).closest('.search-bar'))) {
    e.preventDefault()
    if (e.shiftKey) findPrev(); else findNext()
    return
  }

  // Escape：关闭搜索
  if (e.key === 'Escape') { closeSearch(); e.preventDefault(); return }
}

// === 窄窗口自动收起侧边栏 ===
let resizeTimer: ReturnType<typeof setTimeout> | null = null
function handleWindowResize() {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => {
    if (window.innerWidth < 960) siderCollapsed.value = true
  }, 150)
}

// === AI 弹框状态 ===
const showContinueWrite = ref(false)
const showAIEdit = ref(false)
const showAISetup = ref(false)
const showNovelInfo = ref(false)
/** AI 提取元数据结果弹框 */
const showExtractResult = ref(false)
const extractResult = ref<ExtractionResult | null>(null)
const extractLoading = ref(false)
const aiEditMode = ref<'polish' | 'expand'>('polish')
/** 续写结果传入 AIEditDialog 二次处理时的待处理文本 */
const pendingEditContent = ref('')
/** 从 AIEditDialog 返回的精炼结果（润色/扩写后），传递给续写框 */
const refinedContinueResult = ref('')
const aiConfigured = computed(() => settingsStore.settings?.aiConfigured ?? false)

function openAIEdit(mode: 'polish' | 'expand') {
  aiEditMode.value = mode
  pendingEditContent.value = ''
  showAIEdit.value = true
}

/** 从续写结果进入二次处理（润色/扩写） */
function handleContinueEdit(mode: 'polish' | 'expand', text: string) {
  pendingEditContent.value = text
  aiEditMode.value = mode
  showAIEdit.value = true
  // 每次进入 AIEditDialog 时清空上次的精炼结果，避免重复应用
  refinedContinueResult.value = ''
}

/** AIEditDialog 返回精炼结果（润色/扩写后），填入续写框 */
function handleReplaceExternal(text: string) {
  refinedContinueResult.value = text
}


// === 生命周期 ===
onMounted(async () => {
  if (!settingsStore.settings) await settingsStore.load()
  initFromSettings()
  await novelStore.loadNovels()
  const n = novelStore.novels.find(n => n.id === novelId)
  if (n) {
    novelStore.selectNovel(n)
    // 进入小说时设置窗口标题为小说名
    setWindowTitle(n.title)
    await novelStore.loadChapters(n.id)
    if (novelStore.chapters.length > 0) {
      novelStore.selectChapter(novelStore.chapters[0])
      await nextTick()
      // 用 addToHistory=false 的 transaction 设内容，避免产生 undo 记录
      const { state, view, schema } = editor.value!
      const html = toTiptapHtml(novelStore.chapters[0].content)
      const doc = createDocument(html, schema)
      const tr = state.tr.replaceWith(0, state.doc.content.size, doc)
      tr.setMeta('addToHistory', false)
      view.dispatch(tr)
      editContent.value = novelStore.chapters[0].content
      contentChanged.value = false
      hasUserEdited.value = false
    }
  }
  searchPlugin = createSearchPluginInst()
  editor.value?.registerPlugin(searchPlugin)
  document.addEventListener('keydown', handleKeydown)
  window.addEventListener('resize', handleWindowResize)
  startPolling()
  isInitializing.value = false
})

onUnmounted(() => {
  // 退出编辑器时恢复默认窗口标题
  setWindowTitle('AI 小说编辑器')
  editor.value?.destroy()
  if (searchPlugin) editor.value?.unregisterPlugin(searchPluginKey)
  document.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', handleWindowResize)
  if (resizeTimer) clearTimeout(resizeTimer)
  stopAutoSave()
})
</script>

<template>
  <n-layout has-sider style="height: 100vh">
    <ChapterSidebar v-model:siderCollapsed="siderCollapsed" @goBack="goBack" @saveBeforeSwitch="doSave" />

    <div class="main-area">
      <EditorToolbar
        :undoable="undoable" :redoable="redoable"
        :fontSize="fontSize" :lineHeight="lineHeight" :paragraphSpacing="paragraphSpacing"
        :isBold="isBold" :isItalic="isItalic" :fontFamily="fontFamily"
        :showSearch="showSearch" :canFormat="canFormat"
        :fontOptions="fontOptions"
        :contentChanged="contentChanged"
        :autoSaveEnabled="!!settingsStore.settings?.autoSave"
        @undo="undo" @redo="redo"
        @update:fontSize="fontSize = $event" @update:lineHeight="lineHeight = $event"
        @update:paragraphSpacing="paragraphSpacing = $event"
        @update:isBold="isBold = $event" @update:isItalic="isItalic = $event"
        @update:fontFamily="fontFamily = $event"
        @update:showSearch="showSearch = $event"
        @formatContent="formatContent" @save="doSave" />

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
            <!-- 导航按钮：所有模式下均可用 -->
            <n-button quaternary size="tiny" :disabled="totalMatches === 0" @click="findPrev" title="上一个 (Shift+F3)">
              <template #icon><n-icon size="14"><PrevIcon/></n-icon></template>
            </n-button>
            <n-button quaternary size="tiny" :disabled="totalMatches === 0" @click="findNext" title="下一个 (F3)">
              <template #icon><n-icon size="14"><NextIcon/></n-icon></template>
            </n-button>
            <!-- 匹配计数：本章 + 全书（搜索模式下同时显示） -->
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
            <!-- 与搜索行范围切换完全相同结构的不可见占位，确保替换输入框与搜索输入框始终对齐 -->
            <span class="search-scope-placeholder" aria-hidden="true">
              <span class="scope-btn">本章</span>
              <span class="scope-divider">|</span>
              <span class="scope-btn">全书</span>
            </span>
            <n-input :value="replaceText" placeholder="替换为..." size="small"
              class="replace-input" style="width:200px"
              @update:value="(v: string) => replaceText = v" />
            <!-- 本章模式：替换当前 + 全部替换 -->
            <template v-if="!searchAll">
              <n-button size="tiny" :disabled="totalMatches === 0 || !replaceText" @click="replaceCurrent">
                替换
              </n-button>
              <n-button size="tiny" :disabled="totalMatches === 0 || !replaceText" @click="replaceAll">
                全部替换
              </n-button>
            </template>
            <!-- 全书模式：仅全书替换，避免单个替换逻辑混淆 -->
            <template v-else>
              <n-button size="tiny" :disabled="allSearchTotal === 0 || !replaceText" @click="replaceAllInBook">
                全书替换
              </n-button>
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

      <div class="editor-area">
        <div v-if="currentChapter" class="editor-content" :style="editorStyles">
          <div class="editor-page">
            <editor-content :editor="editor" class="content-editable" />
            <!-- 拆分章节：选中文本后亮起，点击将所选内容拆出为独立章节 -->
            <div class="split-bar">
              <n-button size="tiny" round
                :type="hasSelection ? 'primary' : 'default'"
                :disabled="!hasSelection"
                @click="handleSplitClick"
                title="将选中内容拆分为新章节">
                <template #icon><n-icon size="16"><SplitIcon/></n-icon></template>
                拆分为新章节
              </n-button>
            </div>
          </div>
        </div>
        <div v-else class="editor-empty">
          <n-text depth="3">还没有章节，请创建第一章</n-text>
        </div>
      </div>

      <EditorStatusBar
        :wordCount="wordCount" :aiConfigured="aiConfigured" :contentChanged="contentChanged"
        :extractLoading="extractLoading"
        @continue="showContinueWrite = true"
        @polish="openAIEdit('polish')"
        @expand="openAIEdit('expand')"
        @extract="handleExtractSupplement"
        @setupAI="showAISetup = true"
        @showInfo="showNovelInfo = true" />
    </div>

    <AIContinueDialog v-model:show="showContinueWrite" :refinedContent="refinedContinueResult"
      @polishResult="(t: string) => handleContinueEdit('polish', t)"
      @expandResult="(t: string) => handleContinueEdit('expand', t)" />
    <AIEditDialog v-model:show="showAIEdit" :mode="aiEditMode" :externalContent="pendingEditContent"
      @replaceExternal="handleReplaceExternal" />
    <AISetupDialog v-model:show="showAISetup" />
    <NovelInfoDialog v-model:show="showNovelInfo" />

    <!-- 拆分章节弹框：输入新章节名 -->
    <n-modal class="dialog-modal" v-model:show="showSplitDialog" title="拆分为新章节" preset="card"
      style="width: 380px" :mask-closable="false">
      <n-form label-placement="top">
        <n-form-item label="新章节标题" required>
          <n-input v-model:value="splitChapterTitle" placeholder="输入章节标题"
            @keyup.enter="confirmSplit" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="cancelSplit">取消</n-button>
          <n-button type="primary" :loading="splittingChapter" @click="confirmSplit">拆分</n-button>
        </n-space>
      </template>
    </n-modal>

    <ExtractResultDialog v-model:show="showExtractResult" :extractResult="extractResult"
      :currentNovelId="novelId"
      :existingOutline="novelStore.currentNovel?.outline"
      :existingCharacters="novelStore.currentNovel?.characters"
      :existingRelationships="novelStore.currentNovel?.relationships"
      :existingEvents="novelStore.currentNovel?.events" />
  </n-layout>
</template>

<style scoped>
.main-area {
  flex: 1; min-width: 0; min-height: 0;
  display: flex; flex-direction: column; overflow: hidden;
}

/* 独立搜索栏：居中布局 */
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
  padding: 0 64px; /* 与编辑器正文对齐 */
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

/* 本章/全书切换 */
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

/* 替换行占位：与搜索范围切换完全相同结构，使两行输入框对齐 */
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
.all-search-chapter { }
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

.editor-area {
  flex: 1; overflow: hidden; min-height: 0; display: flex;
}

.editor-content { flex: 1; overflow-y: auto; background: #f0f2f5; }

/* Word 风格的"页面"容器：白底、阴影、四角对齐标记 */
.editor-page {
  width: 100%; max-width: 960px; min-height: 100%;
  margin: 0 auto; background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  position: relative;
  /* flex 撑满页面高度，使底部两角标对齐到页面底部 */
  display: flex; flex-direction: column;
}

.content-editable {
  padding: 48px 64px 64px;
  outline: none;
  white-space: pre-wrap; box-sizing: border-box;
  position: relative; /* 四角标记定位锚点 */
  flex: 1; /* 撑满 .editor-page 剩余空间 */
  display: flex; flex-direction: column; /* 让 .ProseMirror 撑满可点击区域 */
}
.content-editable:focus,
.content-editable:focus-visible,
.content-editable:focus-within {
  outline: none;
}

/* ── 四角对齐标记（Word 风格 crop marks）── */
/* 拐角位于文字区域边界，短边朝外延伸 */
/* 左上角 ┘ */
.editor-page::before {
  content: ''; position: absolute; pointer-events: none; z-index: 1;
  /* 文字区域左上角 (padding: 48 64)，┘ 拐角在盒子右下 */
  top: 28px; left: 44px;
  width: 20px; height: 20px;
  border-right: 2px solid #c0c0c0;
  border-bottom: 2px solid #c0c0c0;
}
/* 右上角 └ */
.editor-page::after {
  content: ''; position: absolute; pointer-events: none; z-index: 1;
  top: 28px; right: 44px;
  width: 20px; height: 20px;
  border-left: 2px solid #c0c0c0;
  border-bottom: 2px solid #c0c0c0;
}
/* 左下角 ┐ */
.content-editable::before {
  content: ''; position: absolute; pointer-events: none; z-index: 1;
  bottom: 44px; left: 44px;
  width: 20px; height: 20px;
  border-right: 2px solid #c0c0c0;
  border-top: 2px solid #c0c0c0;
}
/* 右下角 ┌ */
.content-editable::after {
  content: ''; position: absolute; pointer-events: none; z-index: 1;
  bottom: 44px; right: 44px;
  width: 20px; height: 20px;
  border-left: 2px solid #c0c0c0;
  border-top: 2px solid #c0c0c0;
}

.content-editable :deep(p) {
  text-indent: 2em;
  margin-bottom: var(--p-gap, 16px);
}
.content-editable :deep(p:last-child) {
  margin-bottom: 0;
}

/* 拆分章节按钮栏：位于页面底部，选中文本后按钮亮起 */
.split-bar {
  flex-shrink: 0;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 8px 64px 12px;
  gap: 8px;
}

.editor-empty { flex: 1; display: flex; align-items: center; justify-content: center; background: #f0f2f5; }

/* 搜索高亮 */
.content-editable :deep(.search-highlight) {
  background-color: #fde68a;
  border-radius: 2px;
  padding: 0 1px;
}
.content-editable :deep(.search-current) {
  background-color: #f59e0b;
  border-radius: 2px;
  padding: 0 1px;
}

.content-editable :deep(.ProseMirror),
.content-editable :deep(.ProseMirror-focused),
.content-editable :deep(.ProseMirror:focus) {
  outline: none !important;
  border: none !important;
  box-shadow: none !important;
  flex: 1; /* 撑满 .content-editable，使整页可点击编辑 */
}

</style>

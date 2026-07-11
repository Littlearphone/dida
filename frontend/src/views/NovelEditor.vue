<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, provide, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useNovelStore } from '../stores/novel'
import { useSettingsStore } from '../stores/settings'
import { EDITOR_ACTIONS_KEY } from '../types/editor'
import { useEditorAppearance } from '../composables/useEditorAppearance'
import { useAutoSave } from '../composables/useAutoSave'
import ChapterSidebar from '../components/editor/ChapterSidebar.vue'
import EditorToolbar from '../components/editor/EditorToolbar.vue'
import EditorStatusBar from '../components/editor/EditorStatusBar.vue'
import AIContinueDialog from '../components/editor/AIContinueDialog.vue'
import AIEditDialog from '../components/editor/AIEditDialog.vue'
import AISetupDialog from '../components/editor/AISetupDialog.vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { createDocument } from '@tiptap/core'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import StarterKit from '@tiptap/starter-kit'
import { NLayout, NText, NInput, NButton, NIcon, useMessage } from 'naive-ui'
import {
  ChevronUpOutline as PrevIcon,
  ChevronDownOutline as NextIcon,
  CloseOutline as CloseIcon,
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
  onUpdate: ({ editor: ed, transaction }) => {
    editContent.value = ed.getHTML()
    if (transaction.docChanged) hasUserEdited.value = true
    if (!contentChanged.value) contentChanged.value = true
    triggerAutoSave()
  },
})

function undo() { editor.value?.chain().focus().undo().run() }
function redo() { editor.value?.chain().focus().redo().run() }
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
    // 直接构造并派发单次事务，避免 chain().focus() 产生两次独立事务
    const tr = from === to
      ? state.tr.insertText(text, state.doc.content.size)
      : state.tr.replaceWith(from, to, state.schema.text(text))
    view.dispatch(tr)
    view.focus()
    contentChanged.value = true
  },
})

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
    return html.split(/\r?\n/).map(s => s.trim()).filter(Boolean).map(s => `<p>${s}</p>`).join('')
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
watch(currentChapter, (ch) => {
  if (ch && editor.value && !isInitializing.value) {
    editor.value.commands.setContent(toTiptapHtml(ch.content))
    editContent.value = ch.content
    contentChanged.value = false
    hasUserEdited.value = false
  }
})

// === 返回小说列表 ===
function goBack() {
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

// === 搜索 / 替换（ProseMirror Decoration 高亮） ===
const showSearch = ref(false)
const showReplace = ref(false)
const searchQuery = ref('')
const replaceText = ref('')
const currentMatchIndex = ref(0)
const totalMatches = ref(0)

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

/** 创建 Decoration 插件 */
function createSearchPluginInst() {
  return new Plugin({
    key: searchPluginKey,
    state: {
      init() { return DecorationSet.empty },
      apply(tr) {
        const meta = tr.getMeta(searchPluginKey)
        if (!meta) return DecorationSet.empty
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

/** 扫描并高亮 */
function updateSearch() {
  if (!editor.value) return
  const query = searchQuery.value
  if (!query) { clearSearchHighlights(); return }
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

/** 替换当前 */
function replaceCurrent() {
  if (!editor.value || !replaceText.value) return
  const idx = currentMatchIndex.value - 1
  const m = matches[idx]
  if (!m) return
  editor.value.chain()
    .focus()
    .command(({ tr }) => {
      tr.replaceWith(m.from, m.to, editor.value!.schema.text(replaceText.value))
      return true
    })
    .run()
  nextTick(() => updateSearch())
}

/** 全部替换 */
function replaceAll() {
  if (!editor.value || !replaceText.value || matches.length === 0) return
  const sorted = [...matches].sort((a, b) => b.from - a.from)
  editor.value.chain()
    .focus()
    .command(({ tr }) => {
      for (const m of sorted) tr.replaceWith(m.from, m.to, editor.value!.schema.text(replaceText.value))
      return true
    })
    .run()
  nextTick(() => updateSearch())
}

/** 关闭搜索 */
function closeSearch() {
  showSearch.value = false
  showReplace.value = false
  searchQuery.value = ''
  replaceText.value = ''
  clearSearchHighlights()
  editor.value?.commands.focus()
}

// === 键盘快捷键 ===
function handleKeydown(e: KeyboardEvent) {
  const isCtrl = e.ctrlKey || e.metaKey

  // Ctrl+F：开关搜索栏（不影响替换栏）
  if (isCtrl && e.key === 'f' && !e.shiftKey) {
    e.preventDefault()
    if (showSearch.value) { closeSearch(); return }
    showSearch.value = true
    nextTick(() => (document.querySelector('.search-input input') as HTMLInputElement)?.focus())
    return
  }

  // Ctrl+H：开关替换栏（自动补开搜索栏，关闭时保留搜索栏）
  if (isCtrl && e.key === 'h') {
    e.preventDefault()
    if (showReplace.value) { showReplace.value = false; return }
    showSearch.value = true
    showReplace.value = true
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
const aiEditMode = ref<'polish' | 'expand'>('polish')
const aiConfigured = computed(() => settingsStore.settings?.aiConfigured ?? false)

function openAIEdit(mode: 'polish' | 'expand') {
  aiEditMode.value = mode
  showAIEdit.value = true
}

// === 生命周期 ===
onMounted(async () => {
  if (!settingsStore.settings) await settingsStore.load()
  initFromSettings()
  await novelStore.loadNovels()
  const n = novelStore.novels.find(n => n.id === novelId)
  if (n) {
    novelStore.selectNovel(n)
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
    <ChapterSidebar v-model:siderCollapsed="siderCollapsed" @goBack="goBack" />

    <div class="main-area">
      <EditorToolbar
        :undoable="undoable" :redoable="redoable"
        :fontSize="fontSize" :lineHeight="lineHeight" :paragraphSpacing="paragraphSpacing"
        :isBold="isBold" :isItalic="isItalic" :fontFamily="fontFamily"
        :showSearch="showSearch" :canFormat="canFormat"
        :fontOptions="fontOptions"
        :contentChanged="contentChanged" :showSavedIndicator="showSavedIndicator"
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
            <n-input :value="searchQuery" placeholder="搜索正文..." size="small"
              class="search-input" style="width:220px"
              @update:value="(v: string) => { searchQuery = v; updateSearch() }"
              @keydown.enter="findNext" />
            <n-button quaternary size="tiny" :disabled="totalMatches === 0" @click="findPrev"
              title="上一个 (Shift+F3)">
              <template #icon><n-icon size="14"><PrevIcon/></n-icon></template>
            </n-button>
            <n-button quaternary size="tiny" :disabled="totalMatches === 0" @click="findNext"
              title="下一个 (F3)">
              <template #icon><n-icon size="14"><NextIcon/></n-icon></template>
            </n-button>
            <n-text v-if="totalMatches > 0" class="match-counter">{{ currentMatchIndex }}/{{ totalMatches }}</n-text>
            <n-text v-else depth="3" class="match-counter">无结果</n-text>
            <n-button quaternary size="tiny" class="search-close" @click="closeSearch" title="关闭搜索 (Esc)">
              <template #icon><n-icon size="14"><CloseIcon/></n-icon></template>
            </n-button>
          </div>
          <div v-if="showReplace" class="search-row">
            <n-input :value="replaceText" placeholder="替换为..." size="small"
              class="replace-input" style="width:220px"
              @update:value="(v: string) => replaceText = v" />
            <n-button size="tiny" :disabled="totalMatches === 0 || !replaceText" @click="replaceCurrent">
              替换
            </n-button>
            <n-button size="tiny" :disabled="totalMatches === 0 || !replaceText" @click="replaceAll">
              全部替换
            </n-button>
          </div>
        </div>
      </div>

      <div class="editor-area">
        <div v-if="currentChapter" class="editor-content" :style="editorStyles">
          <div class="editor-page">
            <editor-content :editor="editor" class="content-editable" />
          </div>
        </div>
        <div v-else class="editor-empty">
          <n-text depth="3">还没有章节，请创建第一章</n-text>
        </div>
      </div>

      <EditorStatusBar
        :wordCount="wordCount" :aiConfigured="aiConfigured"
        @continue="showContinueWrite = true"
        @polish="openAIEdit('polish')"
        @expand="openAIEdit('expand')"
        @setupAI="showAISetup = true" />
    </div>

    <AIContinueDialog v-model:show="showContinueWrite" />
    <AIEditDialog v-model:show="showAIEdit" :mode="aiEditMode" />
    <AISetupDialog v-model:show="showAISetup" />
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
}
.match-counter {
  font-size: 12px; min-width: 44px;
  text-align: center; white-space: nowrap;
}
.search-close {
  margin-left: 4px;
}

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

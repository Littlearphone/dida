<script setup lang="ts">
/**
 * 小说编辑器 — 核心视图
 * 组合子组件：ChapterSidebar / EditorToolbar / EditorContentArea / SearchReplacePanel / EditorStatusBar + AI 弹框
 */
import { computed, nextTick, onMounted, onUnmounted, provide, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useNovelStore } from '../stores/novel'
import { useSettingsStore } from '../stores/settings'
import { EDITOR_ACTIONS_KEY } from '../types/editor'
import { useEditorAppearance } from '../composables/useEditorAppearance'
import { useAutoSave } from '../composables/useAutoSave'
import { useChapterSplit } from '../composables/useChapterSplit'
import { useExport } from '../composables/useExport'
import { useAIDialogs } from '../composables/useAIDialogs'
import { toTiptapHtml, wordCount as wc } from '../utils/editor'
import { setWindowTitle } from '../utils/windowTitle'
import ChapterSidebar from '../components/editor/ChapterSidebar.vue'
import EditorToolbar from '../components/editor/EditorToolbar.vue'
import EditorContentArea from '../components/editor/EditorContentArea.vue'
import EditorStatusBar from '../components/editor/EditorStatusBar.vue'
import SearchReplacePanel from '../components/editor/SearchReplacePanel.vue'
import { useEditor } from '@tiptap/vue-3'
import { createDocument } from '@tiptap/core'
import { DOMParser, Slice } from '@tiptap/pm/model'
import StarterKit from '@tiptap/starter-kit'
import { NLayout, useMessage } from 'naive-ui'

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
/** 用户是否有过实际编辑（用于判断 undo 是否可用，避免初始加载后 undo 误亮） */
const hasUserEdited = ref(false)
const searchPanelRef = ref<InstanceType<typeof SearchReplacePanel> | null>(null)

const editor = useEditor({
  content: '',
  extensions: [StarterKit],
  editorProps: {
    // 为搜索导航留出顶部间距，避免被搜索栏遮挡
    scrollMargin: { top: 80, right: 0, bottom: 40, left: 0 },
    /** 粘贴纯文本：无换行直接插入（不破坏段落），有换行按段落拆分 */
    handlePaste: (view, event) => {
      const text = (event.clipboardData?.getData('text/plain') || '').trim()
      if (!text) return false // 有 HTML 时让默认处理器处理

      event.preventDefault()
      // 无换行 → 纯文本插入，避免产生额外分段（#粘贴换行bug）
      if (!/\n/.test(text)) {
        view.dispatch(view.state.tr.insertText(text))
        return true
      }
      // 有换行 → 按双换行（段落）或单换行拆分
      let paragraphs = text.split(/\n\s*\n/).filter(p => p.trim())
      if (paragraphs.length <= 1) {
        paragraphs = text.split(/\n+/).filter(p => p.trim())
      }
      if (paragraphs.length === 0) {
        view.dispatch(view.state.tr.insertText(text))
        return true
      }
      const html = paragraphs
        .map(p => `<p>${p.replace(/\n/g, '<br>')}</p>`)
        .join('')
      const div = document.createElement('div')
      div.innerHTML = html
      const parser = DOMParser.fromSchema(view.state.schema)
      const doc = parser.parse(div)
      view.dispatch(view.state.tr.replaceSelection(new Slice(doc.content, 0, 0)))
      return true
    },
  },
  onUpdate: ({ editor: ed, transaction }) => {
    editContent.value = ed.getHTML()
    if (transaction.docChanged) hasUserEdited.value = true
    if (!contentChanged.value) contentChanged.value = true
    triggerAutoSave()
  },
  // 选区变化时更新选中字数（ProseMirror 选区不触发 computed 重算，需手动追踪）
  onSelectionUpdate: ({ editor: ed }) => {
    const { from, to } = ed.state.selection
    selectedWordCount.value = from !== to ? wc(ed.state.doc.textBetween(from, to)) : 0
  },
})

const selectedWordCount = ref(0)

function undo() {
  const ed = editor.value
  if (!ed) return
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
  setContent: (html: string) => {
    editContent.value = html
    contentChanged.value = true
    editor.value?.commands.setContent(toTiptapHtml(html))
  },
  getContent: () => editContent.value,
  markChanged: () => { contentChanged.value = true },
  getSelectionText: () => {
    if (!editor.value) return ''
    const { from, to } = editor.value.state.selection
    if (from === to) return ''
    return editor.value.state.doc.textBetween(from, to, '\n\n')
  },
  replaceSelection: (text: string) => {
    if (!editor.value) return
    const { state, view } = editor.value
    const { from, to } = state.selection
    if (from === to) {
      editor.value.commands.insertContentAt(state.doc.content.size, toTiptapHtml(text))
    } else {
      editor.value.commands.insertContentAt({ from, to }, toTiptapHtml(text))
    }
    view.focus()
    contentChanged.value = true
  },
  appendContent: (text: string) => {
    if (!editor.value) return
    const html = toTiptapHtml(text)
    editor.value.commands.insertContentAt(editor.value.state.doc.content.size, html)
    contentChanged.value = true
  },
  /** AI 内容插入后自动提取元数据 */
  triggerExtract: () => { nextTick(() => handlePostInsertExtract()) },
})

// === AI 弹框 & 提取 ===
const {
  showContinueWrite, showAIEdit, showAISetup, showNovelInfo,
  showExtractResult, extractResult, extractLoading, extractStreamText,
  aiEditMode, pendingEditContent, refinedContinueResult, aiConfigured,
  openAIEdit, handleContinueEdit, handleReplaceExternal,
  handlePostInsertExtract, handleExtractSupplement, cancelExtract,
} = useAIDialogs(novelStore, settingsStore, editor, contentChanged, doSaveChapter, message)

// === 导出 ===
const { handleExport } = useExport(novelStore, message)

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

// === 切换章节时同步编辑器（初始化时不执行） ===
const isInitializing = ref(true)
watch(currentChapter, (ch, oldCh) => {
  if (ch && editor.value && !isInitializing.value) {
    if (oldCh && ch.id === oldCh.id) {
      editContent.value = ch.content
      return
    }
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
  setWindowTitle('AI 小说编辑器')
  const go = () => { novelStore.selectNovel(null); router.push({ name: 'NovelList' }) }
  if (contentChanged.value) doSave().then(go)
  else go()
}

// === 字数统计 ===
const wordCount = computed(() => wc(editContent.value))

// === 章节记忆：localStorage 持久化上次打开的章节，刷新后恢复 ===
const CHAPTER_MEMORY_PREFIX = 'dida_last_chapter_'

function saveLastChapterId(novelId: string, chapterId: string) {
  try { localStorage.setItem(CHAPTER_MEMORY_PREFIX + novelId, chapterId) } catch { /* quota exceeded 等异常静默忽略 */ }
}

function getLastChapterId(novelId: string): string | null {
  try { return localStorage.getItem(CHAPTER_MEMORY_PREFIX + novelId) } catch { return null }
}

// 章节切换时自动记住（覆盖侧边栏点击、AI 新建章节、拖拽排序后自动切换等所有场景）
watch(currentChapter, (ch) => {
  if (ch && novelStore.currentNovel) {
    saveLastChapterId(novelStore.currentNovel.id, ch.id)
  }
})

// === 格式化解多余空段落 ===
const canFormat = computed(() => {
  const html = editContent.value
  if (!html) return false
  return /<p>\s*<\/p>|<p>\s*<br>\s*<\/p>/i.test(html) && /<p>[^<]+<\/p>/i.test(html)
})

function formatContent() {
  const formatted = editContent.value
    .replace(/<p>\s*<\/p>/gi, '')
    .replace(/<p>\s*<br\s*\/?>\s*<\/p>/gi, '')
  if (formatted !== editContent.value) {
    editContent.value = formatted
    contentChanged.value = true
    editor.value?.commands.setContent(toTiptapHtml(formatted))
  }
}

// === 拆分章节（逻辑由 useChapterSplit composable 管理） ===
const {
  showSplitDialog, splitChapterTitle, splittingChapter,
  hasSelection, handleSplitClick, confirmSplit, cancelSplit,
} = useChapterSplit(editor, doSaveChapter)

// === 键盘快捷键 ===
function handleKeydown(e: KeyboardEvent) {
  const isCtrl = e.ctrlKey || e.metaKey
  const sp = searchPanelRef.value

  // Ctrl+F：开关搜索栏
  if (isCtrl && e.key === 'f' && !e.shiftKey) {
    e.preventDefault()
    if (sp?.isOpen()) {
      // 有选中文本时填入搜索框而不再关闭
      if (editor.value) {
        const { from, to } = editor.value.state.selection
        if (from !== to) {
          sp.fillSearchFromSelection(
            editor.value.state.doc.textBetween(from, to).slice(0, 200),
          )
          return
        }
      }
      sp.closeSearch(); return
    }
    // 打开搜索并自动填入选中文本
    let selText = ''
    if (editor.value) {
      const { from, to } = editor.value.state.selection
      if (from !== to) selText = editor.value.state.doc.textBetween(from, to).slice(0, 200)
    }
    sp?.openSearch(selText)
    return
  }

  // Ctrl+H：开关替换栏
  if (isCtrl && e.key === 'h') {
    e.preventDefault()
    if (sp?.isReplaceOpen()) {
      if (editor.value) {
        const { from, to } = editor.value.state.selection
        if (from !== to) {
          sp.fillSearchFromSelection(
            editor.value.state.doc.textBetween(from, to).slice(0, 200),
          )
          return
        }
      }
      sp?.openSearch() // Ctrl+H 关闭时保留搜索栏
      return
    }
    let selText = ''
    if (editor.value) {
      const { from, to } = editor.value.state.selection
      if (from !== to) selText = editor.value.state.doc.textBetween(from, to).slice(0, 200)
    }
    sp?.openReplace(selText)
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

  // Ctrl+S：手动保存
  if (isCtrl && e.key === 's') {
    e.preventDefault()
    doSave()
    return
  }

  // 搜索栏已打开时，搜索快捷键由 SearchReplacePanel 内部处理
}

// === 窄窗口自动收起侧边栏 ===
let resizeTimer: ReturnType<typeof setTimeout> | null = null
function handleWindowResize() {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => {
    if (window.innerWidth < 960) siderCollapsed.value = true
  }, 150)
}

// === 生命周期 ===
onMounted(async () => {
  if (!settingsStore.settings) await settingsStore.load()
  initFromSettings()
  await novelStore.loadNovels()
  const n = novelStore.novels.find(n => n.id === novelId)
  if (n) {
    novelStore.selectNovel(n)
    setWindowTitle(n.title)
    await novelStore.loadChapters(n.id)
    if (novelStore.chapters.length > 0) {
      // 恢复上次打开的章节（localStorage 记忆），若不存在则回退到第一章
      const savedId = getLastChapterId(n.id)
      const targetChapter = (savedId && novelStore.chapters.find(c => c.id === savedId))
        || novelStore.chapters[0]
      novelStore.selectChapter(targetChapter)
      await nextTick()
      const { state, view, schema } = editor.value!
      const html = toTiptapHtml(targetChapter.content)
      const doc = createDocument(html, schema)
      const tr = state.tr.replaceWith(0, state.doc.content.size, doc)
      tr.setMeta('addToHistory', false)
      view.dispatch(tr)
      editContent.value = targetChapter.content
      contentChanged.value = false
      hasUserEdited.value = false
    }
  }
  document.addEventListener('keydown', handleKeydown)
  window.addEventListener('resize', handleWindowResize)
  startPolling()
  isInitializing.value = false
})

onUnmounted(() => {
  setWindowTitle('AI 小说编辑器')
  editor.value?.destroy()
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
        :showSearch="searchPanelRef?.isOpen() ?? false" :canFormat="canFormat"
        :fontOptions="fontOptions"
        :contentChanged="contentChanged"
        :autoSaveEnabled="!!settingsStore.settings?.autoSave"
        @undo="undo" @redo="redo"
        @update:fontSize="fontSize = $event" @update:lineHeight="lineHeight = $event"
        @update:paragraphSpacing="paragraphSpacing = $event"
        @update:isBold="isBold = $event" @update:isItalic="isItalic = $event"
        @update:fontFamily="fontFamily = $event"
        @update:showSearch="($event) => $event ? searchPanelRef?.openSearch() : searchPanelRef?.closeSearch()"
        @formatContent="formatContent" @save="doSave" />

      <!-- 搜索/替换面板 -->
      <SearchReplacePanel ref="searchPanelRef" :editor="editor" :doSaveChapter="doSaveChapter" />

      <!-- 编辑器内容区 -->
      <EditorContentArea
        :currentChapter="currentChapter"
        :editor="editor"
        :editorStyles="editorStyles"
        :hasSelection="hasSelection"
        @split="handleSplitClick" />

      <EditorStatusBar
        :wordCount="wordCount" :selectedWordCount="selectedWordCount"
        :aiConfigured="aiConfigured" :contentChanged="contentChanged"
        :extractLoading="extractLoading"
        :novelTitle="novelStore.currentNovel?.title || ''"
        @continue="showContinueWrite = true"
        @polish="openAIEdit('polish')"
        @expand="openAIEdit('expand')"
        @extract="handleExtractSupplement"
        @cancelExtract="cancelExtract"
        @setupAI="showAISetup = true"
        @showInfo="showNovelInfo = true"
        @export="handleExport" />
    </div>

    <!-- AI 弹框 -->
    <AIContinueDialog v-model:show="showContinueWrite" :refinedContent="refinedContinueResult"
      @polishResult="(t: string) => handleContinueEdit('polish', t)"
      @expandResult="(t: string) => handleContinueEdit('expand', t)" />
    <AIEditDialog v-model:show="showAIEdit" :mode="aiEditMode" :externalContent="pendingEditContent"
      @replaceExternal="handleReplaceExternal" />
    <AISetupDialog v-model:show="showAISetup" />
    <NovelInfoDialog v-model:show="showNovelInfo" />

    <!-- 拆分章节弹框 -->
    <SplitChapterDialog
      :show="showSplitDialog"
      :loading="splittingChapter"
      :defaultTitle="splitChapterTitle"
      @update:show="showSplitDialog = $event"
      @confirm="confirmSplit"
      @cancel="cancelSplit" />

    <ExtractResultDialog v-model:show="showExtractResult" :extractResult="extractResult"
      :loading="extractLoading" :streamText="extractStreamText"
      :currentNovelId="novelId"
      :existingOutline="novelStore.currentNovel?.outline"
      :existingCharacters="novelStore.currentNovel?.characters"
      :existingRelationships="novelStore.currentNovel?.relationships"
      :existingEvents="novelStore.currentNovel?.events" />
  </n-layout>
</template>

<style lang="scss" scoped>
.main-area {
  flex: 1; min-width: 0; min-height: 0;
  display: flex; flex-direction: column; overflow: hidden;
}

/* 搜索高亮样式（来自 SearchReplacePanel 的 Decoration） */
:deep(.search-highlight) {
  background-color: #fde68a;
  border-radius: 2px;
  padding: 0 1px;
}

:deep(.search-current) {
  background-color: #f59e0b;
  border-radius: 2px;
  padding: 0 1px;
}
</style>

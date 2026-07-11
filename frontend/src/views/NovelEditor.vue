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
import StarterKit from '@tiptap/starter-kit'
import { NLayout, NText, useMessage } from 'naive-ui'

const route = useRoute()
const router = useRouter()
const novelStore = useNovelStore()
const settingsStore = useSettingsStore()

const message = useMessage()
const novelId = route.params.novelId as string
const currentChapter = computed(() => novelStore.currentChapter)
const siderCollapsed = ref(false)

// === Tiptap 编辑器 ===
const editContent = ref('')
const contentChanged = ref(false)
const editor = useEditor({
  content: '',
  extensions: [StarterKit],
  onUpdate: ({ editor: ed }) => {
    editContent.value = ed.getHTML()
    if (!contentChanged.value) contentChanged.value = true
    triggerAutoSave()
  },
})

function undo() { editor.value?.chain().focus().undo().run() }
function redo() { editor.value?.chain().focus().redo().run() }
const undoable = computed(() => editor.value?.can().undo() ?? false)
const redoable = computed(() => editor.value?.can().redo() ?? false)

// === 提供编辑器操作给 AI 弹框 ===
provide(EDITOR_ACTIONS_KEY, {
  setContent: (html: string) => setEditorContent(html),
  getContent: () => editContent.value,
  markChanged: () => { contentChanged.value = true },
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

// === 切换章节时同步编辑器 ===
watch(currentChapter, (ch) => {
  if (ch && editor.value) {
    editor.value.commands.setContent(toTiptapHtml(ch.content))
    editContent.value = ch.content
    contentChanged.value = false
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

// === 工具栏功能 ===
function copySelection() {
  const sel = window.getSelection()
  if (sel?.toString()) {
    navigator.clipboard.writeText(sel.toString()).then(() => message.success('已复制'))
  }
}

const showSearch = ref(false)
const searchQuery = ref('')

function handleSearch() {
  if (!searchQuery.value) return
  const editorEl = document.querySelector('.editor-content')
  if (!editorEl) return
  const text = editorEl.textContent || ''
  const idx = text.toLowerCase().indexOf(searchQuery.value.toLowerCase())
  if (idx === -1) { message.info('未找到匹配内容'); return }
  const tw = document.createTreeWalker(editorEl, NodeFilter.SHOW_TEXT)
  let node: Text | null
  const q = searchQuery.value.toLowerCase()
  while ((node = tw.nextNode() as Text | null)) {
    const t = node.textContent || ''
    const pos = t.toLowerCase().indexOf(q)
    if (pos !== -1) {
      const sel = window.getSelection()
      const r = document.createRange()
      r.setStart(node, pos); r.setEnd(node, pos + q.length)
      sel?.removeAllRanges(); sel?.addRange(r)
      node.parentElement?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      break
    }
  }
}

function formatContent() {
  const formatted = editContent.value.split('\n').filter(l => l.trim().length > 0).join('\n')
  if (formatted !== editContent.value) setEditorContent(formatted)
}

// === AI 弹框状态 ===
const showContinueWrite = ref(false)
const showAIEdit = ref(false)
const showAISetup = ref(false)
const aiEditMode = ref<'polish' | 'expand'>('polish')
const aiEditScope = ref<'selection' | 'chapter'>('chapter')
const aiConfigured = computed(() => settingsStore.settings?.aiConfigured ?? false)

function openAIEdit(mode: 'polish' | 'expand', scope: 'selection' | 'chapter') {
  aiEditMode.value = mode
  aiEditScope.value = scope
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
      editor.value?.commands.setContent(toTiptapHtml(novelStore.chapters[0].content))
    }
  }
  startPolling()
})

onUnmounted(() => {
  editor.value?.destroy()
  stopAutoSave()
})
</script>

<template>
  <n-layout has-sider style="height: 100vh">
    <ChapterSidebar v-model:siderCollapsed="siderCollapsed" @goBack="goBack" />

    <div style="flex: 1; min-width: 0; min-height: 0; display: flex; flex-direction: column; overflow: hidden;">
      <EditorToolbar
        :undoable="undoable" :redoable="redoable"
        :fontSize="fontSize" :lineHeight="lineHeight" :paragraphSpacing="paragraphSpacing"
        :isBold="isBold" :isItalic="isItalic" :fontFamily="fontFamily"
        :showSearch="showSearch" :searchQuery="searchQuery"
        :fontOptions="fontOptions"
        :contentChanged="contentChanged" :showSavedIndicator="showSavedIndicator"
        :autoSaveEnabled="!!settingsStore.settings?.autoSave"
        @undo="undo" @redo="redo"
        @update:fontSize="fontSize = $event" @update:lineHeight="lineHeight = $event"
        @update:paragraphSpacing="paragraphSpacing = $event"
        @update:isBold="isBold = $event" @update:isItalic="isItalic = $event"
        @update:fontFamily="fontFamily = $event"
        @update:showSearch="showSearch = $event" @update:searchQuery="searchQuery = $event"
        @copySelection="copySelection" @search="handleSearch"
        @formatContent="formatContent" @save="doSave" />

      <div style="flex: 1; overflow: hidden; min-height: 0; display: flex;">
        <div v-if="currentChapter" class="editor-content" :style="editorStyles">
          <editor-content :editor="editor" class="content-editable" />
        </div>
        <div v-else class="editor-empty">
          <n-text depth="3">还没有章节，请创建第一章</n-text>
        </div>
      </div>

      <EditorStatusBar
        :wordCount="wordCount" :aiConfigured="aiConfigured"
        @continue="showContinueWrite = true"
        @polish="(s) => openAIEdit('polish', s)"
        @expand="(s) => openAIEdit('expand', s)"
        @setupAI="showAISetup = true" />
    </div>

    <AIContinueDialog v-model:show="showContinueWrite" />
    <AIEditDialog v-model:show="showAIEdit" :mode="aiEditMode" :scope="aiEditScope" />
    <AISetupDialog v-model:show="showAISetup" />
  </n-layout>
</template>

<style scoped>
.editor-content { flex: 1; overflow-y: auto; background: #f0f2f5; }

.content-editable {
  width: 100%; max-width: 960px; min-height: 100%;
  margin: 0 auto; padding: 48px 64px 64px;
  background: #fff; outline: none;
  white-space: pre-wrap; box-sizing: border-box;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}
.content-editable:focus,
.content-editable:focus-visible,
.content-editable:focus-within {
  outline: none;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.content-editable :deep(p) {
  text-indent: 2em;
  margin-bottom: var(--p-gap, 16px);
}
.content-editable :deep(p:last-child) {
  margin-bottom: 0;
}

.editor-empty { flex: 1; display: flex; align-items: center; justify-content: center; background: #f0f2f5; }

.content-editable :deep(.ProseMirror),
.content-editable :deep(.ProseMirror-focused),
.content-editable :deep(.ProseMirror:focus) {
  outline: none !important;
  border: none !important;
  box-shadow: none !important;
}
</style>

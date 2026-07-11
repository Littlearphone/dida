<script setup lang="ts">
import {computed, nextTick, onMounted, onUnmounted, ref, watch} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {useNovelStore} from '../stores/novel'
import {useSettingsStore} from '../stores/settings'
import * as aiApi from '../api/ai'
import type {Chapter} from '../types'
import {EditorContent, useEditor} from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import {
  NAlert, NButton, NDivider, NDropdown, NForm, NFormItem, NGi, NGrid,
  NIcon, NInput, NInputNumber, NLayout, NLayoutSider, NModal, NScrollbar,
  NSelect, NSpace, NText, useMessage,
} from 'naive-ui'
import {
  AddCircleOutline as AddChapterIcon,
  ChevronBackOutline as BackIcon,
  CopyOutline as CopyIcon,
  CutOutline as CutIcon,
  SaveOutline as SaveIcon,
  SearchOutline as SearchIcon,
  SettingsOutline as AISetupIcon,
  SparklesOutline as SparklesIcon,
  TrashOutline as DeleteIcon,
  ArrowUndoOutline as UndoIcon,
  ArrowRedoOutline as RedoIcon,
} from '@vicons/ionicons5'

const route = useRoute()
const router = useRouter()
const novelStore = useNovelStore()
const settingsStore = useSettingsStore()
const message = useMessage()

// === 状态 ===
const novelId = route.params.novelId as string
const novel = computed(() => novelStore.currentNovel)
const chapters = computed(() => novelStore.chapters)
const currentChapter = computed(() => novelStore.currentChapter)
const editContent = ref('')
const autoSaveTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const autoSavePolling = ref<ReturnType<typeof setInterval> | null>(null)
const contentChanged = ref(false)
const showSavedIndicator = ref(false)
const savedTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const siderCollapsed = ref(false)

// Tiptap 编辑器实例（StarterKit 含 History 扩展，Ctrl+Z/Y 开箱即用）
const editor = useEditor({
  content: '',
  extensions: [StarterKit],
  onUpdate: ({ editor: ed }) => {
    editContent.value = ed.getHTML()
    if (!contentChanged.value) {
      contentChanged.value = true
    }
    triggerAutoSave()
  },
})

function undo() { editor.value?.chain().focus().undo().run() }
function redo() { editor.value?.chain().focus().redo().run() }
const undoable = computed(() => editor.value?.can().undo() ?? false)
const redoable = computed(() => editor.value?.can().redo() ?? false)

// === 右键菜单 ===
const contextMenuShow = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextMenuChapter = ref<Chapter | null>(null)
const contextMenuOptions = [
  {label: '重命名', key: 'rename'},
  {label: '删除', key: 'delete'},
]

function openContextMenu(e: MouseEvent, ch: Chapter) {
  e.preventDefault()
  e.stopPropagation()
  contextMenuShow.value = false
  nextTick(() => {
    contextMenuX.value = e.clientX
    contextMenuY.value = e.clientY
    contextMenuChapter.value = ch
    contextMenuShow.value = true
  })
}

function closeContextMenu() {
  contextMenuShow.value = false
  contextMenuChapter.value = null
}

function handleContextMenuSelect(key: string) {
  const ch = contextMenuChapter.value
  if (!ch) return
  closeContextMenu()
  if (key === 'rename') startRenameChapter(ch)
  else if (key === 'delete') confirmDeleteChapter(ch)
}

function onMenuGlobalClose(e: Event) {
  if (!contextMenuShow.value) return
  if (e.type === 'keydown' && (e as KeyboardEvent).key !== 'Escape') return
  closeContextMenu()
}

// 编辑设置
const fontSize = ref(settingsStore.settings?.defaultFontSize || 16)
const lineHeight = ref(settingsStore.settings?.defaultLineSpacing || 1.8)
const isBold = ref(false)
const isItalic = ref(false)
const fontFamily = ref('')
const paragraphSpacing = ref(16)
const showSearch = ref(false)
const searchQuery = ref('')

const fontOptions = [
  { label: '系统默认', value: '' },
  { label: '宋体', value: 'SimSun, serif' },
  { label: '黑体', value: 'SimHei, sans-serif' },
  { label: '微软雅黑', value: '"Microsoft YaHei", sans-serif' },
  { label: '楷体', value: 'KaiTi, serif' },
  { label: '仿宋', value: 'FangSong, serif' },
]

// === 初始化 ===
onMounted(async () => {
  if (!settingsStore.settings) await settingsStore.load()
  if (settingsStore.settings) {
    fontSize.value = settingsStore.settings.defaultFontSize || 16
    lineHeight.value = settingsStore.settings.defaultLineSpacing || 1.8
  }
  document.addEventListener('click', onMenuGlobalClose)
  document.addEventListener('contextmenu', onMenuGlobalClose)
  document.addEventListener('keydown', onMenuGlobalClose)
  document.addEventListener('wheel', onMenuGlobalClose, {passive: true})
  await novelStore.loadNovels()
  const n = novelStore.novels.find(n => n.id === novelId)
  if (n) {
    novelStore.selectNovel(n)
    await novelStore.loadChapters(n.id)
    if (chapters.value.length === 0) {
      const ch = await novelStore.createChapter({
        novelId: n.id, title: '第一章', content: '', order: 1,
      })
      if (ch) {
        novelStore.selectChapter(ch)
        await nextTick()
        editor.value?.commands.setContent('')
      }
    } else {
      selectChapter(chapters.value[0])
      await nextTick()
      if (chapters.value[0]) {
        editor.value?.commands.setContent(toTiptapHtml(chapters.value[0].content))
      }
    }
  }
  startAutoSave()
})

onUnmounted(() => {
  editor.value?.destroy()
  if (autoSaveTimer.value) clearTimeout(autoSaveTimer.value)
  stopAutoSave()
  if (savedTimer.value) clearTimeout(savedTimer.value)
  document.removeEventListener('click', onMenuGlobalClose)
  document.removeEventListener('contextmenu', onMenuGlobalClose)
  document.removeEventListener('keydown', onMenuGlobalClose)
  document.removeEventListener('wheel', onMenuGlobalClose)
})

// === 章节选择 ===
function selectChapter(chapter: Chapter) {
  if (contentChanged.value) saveCurrentChapter()
  novelStore.selectChapter(chapter)
  editContent.value = chapter.content
  contentChanged.value = false
}

// 切换章节时同步内容到编辑区
watch(currentChapter, (ch) => {
  if (ch && editor.value) {
    editor.value.commands.setContent(toTiptapHtml(ch.content))
    editContent.value = ch.content
    contentChanged.value = false
  }
})

/** 兼容旧数据：将 <div> 段落转为 Tiptap 能识别的 <p>，纯文本自动按换行分段 */
function toTiptapHtml(html: string): string {
  if (!html) return ''
  // 纯文本检测：无 HTML 标记时按换行分段落（中文 TXT 每行一段，兼容 \r\n）
  if (!/<\/?[a-z][\s\S]*?>/i.test(html)) {
    return html.split(/\r?\n/).map(s => s.trim()).filter(Boolean).map(s => `<p>${s}</p>`).join('')
  }
  // 将 <div> 标签替换为 <p>（Tiptap schema 用 p 表示段落）
  let out = html.replace(/<div/gi, '<p').replace(/<\/div\s*>/gi, '</p>')
  // 处理 <br><br> 段落分隔（旧 contenteditable 以 pre-wrap 保存的数据）
  if (/(?:<br\s*\/?>\s*){2,}/i.test(out)) {
    const parts = out.split(/(?:<br\s*\/?>\s*){2,}/i).filter(s => s.trim())
    if (parts.length > 1) {
      out = parts.map(s => `<p>${s.trim()}</p>`).join('')
    }
  }
  return out
}

/** 程序化设置编辑区内容（AI操作等外部触发时同步到DOM） */
function setEditorContent(text: string) {
  editContent.value = text
  contentChanged.value = true
  if (editor.value) {
    editor.value.commands.setContent(toTiptapHtml(text))
  }
}

// === 重命名章节（弹框） ===
const showRenameModal = ref(false)
const renameChapterId = ref<string | null>(null)
const renameChapterTitle = ref('')

function startRenameChapter(ch: Chapter) {
  renameChapterId.value = ch.id
  renameChapterTitle.value = ch.title || ''
  showRenameModal.value = true
}

async function saveRenameTitle() {
  if (!renameChapterId.value) return
  await novelStore.updateChapter(renameChapterId.value, {
    title: renameChapterTitle.value.trim(),
  })
  showRenameModal.value = false
}

// === 删除章节 ===
const showDeleteModal = ref(false)
const deleteTargetChapter = ref<Chapter | null>(null)
const deletingChapter = ref(false)

function confirmDeleteChapter(ch: Chapter) {
  deleteTargetChapter.value = ch
  showDeleteModal.value = true
}

async function handleDeleteConfirm() {
  const ch = deleteTargetChapter.value
  if (!ch) return
  deletingChapter.value = true
  await novelStore.deleteChapter(ch.id)
  if (currentChapter.value?.id === ch.id) {
    editContent.value = ''
    contentChanged.value = false
  }
  deletingChapter.value = false
  showDeleteModal.value = false
  deleteTargetChapter.value = null
}

// === 拖拽排序 ===
const dragIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)

function handleDragStart(i: number) { dragIndex.value = i }

function handleDragOver(e: DragEvent, i: number) {
  e.preventDefault()
  if (i === chapters.value.length - 1) {
    const target = e.currentTarget as HTMLElement
    if (target) {
      const rect = target.getBoundingClientRect()
      if (e.clientY - rect.top > rect.height * 0.55) {
        dragOverIndex.value = chapters.value.length; return
      }
    }
  }
  dragOverIndex.value = i
}

function handleDragLeave() { dragOverIndex.value = null }

async function handleDrop(e: DragEvent) {
  e.preventDefault()
  if (dragIndex.value === null || dragIndex.value === dragOverIndex.value) {
    dragIndex.value = null; dragOverIndex.value = null; return
  }
  const from = dragIndex.value
  const to = dragOverIndex.value ?? chapters.value.length - 1
  if (from === to) { dragIndex.value = null; dragOverIndex.value = null; return }
  const list = [...chapters.value]
  const [moved] = list.splice(from, 1)
  const adjustedTo = to > from ? to - 1 : to
  list.splice(adjustedTo, 0, moved)
  const nid = novel.value?.id
  if (!nid) return
  if (await novelStore.reorderChapters(nid, list.map(ch => ch.id))) {
    await novelStore.loadChapters(nid)
  }
  dragIndex.value = null; dragOverIndex.value = null
}

function handleDragEnd() { dragIndex.value = null; dragOverIndex.value = null }

// === 自动保存（防抖 + 轮询兜底） ===
function triggerAutoSave() {
  if (autoSaveTimer.value) clearTimeout(autoSaveTimer.value)
  const ms = settingsStore.settings?.autoSaveMs || 2000
  autoSaveTimer.value = setTimeout(() => saveCurrentChapter(), ms)
}

function startAutoSave() {
  if (autoSavePolling.value) return
  autoSavePolling.value = setInterval(() => {
    if (contentChanged.value) saveCurrentChapter()
  }, 5000)
}

function stopAutoSave() {
  if (autoSaveTimer.value) clearTimeout(autoSaveTimer.value)
  autoSaveTimer.value = null
  if (autoSavePolling.value) { clearInterval(autoSavePolling.value); autoSavePolling.value = null }
}

async function saveCurrentChapter() {
  if (!currentChapter.value || !contentChanged.value) return
  const content = editor.value?.getHTML() || ''
  const ok = await novelStore.updateChapter(currentChapter.value.id, {content})
  if (ok) {
    contentChanged.value = false
    showSavedIndicator.value = true
    if (savedTimer.value) clearTimeout(savedTimer.value)
    savedTimer.value = setTimeout(() => { showSavedIndicator.value = false }, 2000)
  }
}

// === 添加章节 ===
const showAddChapterModal = ref(false)
const newChapterTitle = ref('')
const creatingChapter = ref(false)

async function handleAddChapter() {
  if (!novel.value || !newChapterTitle.value.trim()) {
    message.warning('请输入章节标题'); return
  }
  creatingChapter.value = true
  const ch = await novelStore.createChapter({
    novelId: novel.value.id,
    title: newChapterTitle.value.trim(),
    content: '',
    order: chapters.value.length + 1,
  })
  creatingChapter.value = false
  if (ch) {
    message.success('章节创建成功')
    showAddChapterModal.value = false
    newChapterTitle.value = ''
    novelStore.selectChapter(ch)
    await nextTick()
    editor.value?.commands.setContent(toTiptapHtml(''))
    editContent.value = ''
    contentChanged.value = false
  }
}

// === 格式化正文 ===
function formatContent() {
  const formatted = editContent.value.split('\n').filter(l => l.trim().length > 0).join('\n')
  if (formatted !== editContent.value) setEditorContent(formatted)
}

// === 字数统计 ===
const wordCount = computed(() => {
  return (editContent.value.replace(/<[^>]*>/g, '').replace(/\s/g, '')).length
})

// === 复制选中内容 ===
function copySelection() {
  const sel = window.getSelection()
  if (sel?.toString()) {
    navigator.clipboard.writeText(sel.toString()).then(() => message.success('已复制'))
  } else {
    message.warning('请先选择要复制的内容')
  }
}

// === 搜索 ===
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
      node.parentElement?.scrollIntoView({behavior: 'smooth', block: 'center'})
      break
    }
  }
}

// === 返回小说列表 ===
function goBack() {
  const go = () => { novelStore.selectNovel(null); router.push({name: 'NovelList'}) }
  if (contentChanged.value) saveCurrentChapter().then(go)
  else go()
}

// === AI 功能 ===
const aiConfigured = computed(() => settingsStore.settings?.aiConfigured ?? false)

// AI 续写
const showContinueWrite = ref(false)
const continueRequirement = ref('')
const continueLoading = ref(false)
const continueResult = ref('')
const showContinueResult = ref(false)

async function handleContinueWrite() {
  if (!currentChapter.value) return
  continueLoading.value = true
  try {
    const res = await aiApi.continueWrite({
      chapterContent: currentChapter.value.content,
      outline: novel.value?.outline || '',
      requirement: continueRequirement.value,
    })
    continueResult.value = res.result
    showContinueResult.value = true
  } catch (e: any) { message.error(`续写失败: ${e.message}`) }
  finally { continueLoading.value = false }
}

function insertToChapterEnd() {
  if (!currentChapter.value) return
  setEditorContent(editContent.value + '\n\n' + continueResult.value)
  showContinueWrite.value = false; showContinueResult.value = false
  continueResult.value = ''; continueRequirement.value = ''
  message.success('已插入到章节末尾')
}

async function insertAsNewChapter() {
  if (!novel.value) return
  const ch = await novelStore.createChapter({
    novelId: novel.value.id,
    title: `${currentChapter.value?.title || '续写'} (续)`,
    content: continueResult.value,
    order: chapters.value.length + 1,
  })
  if (ch) { message.success('已创建新章节'); selectChapter(ch) }
  showContinueWrite.value = false; showContinueResult.value = false
  continueResult.value = ''; continueRequirement.value = ''
}

function copyContinueResult() {
  navigator.clipboard.writeText(continueResult.value); message.success('已复制')
}

// AI 润色
const showPolish = ref(false)
const polishScope = ref<'selection' | 'chapter'>('chapter')
const polishRequirement = ref('')
const polishLoading = ref(false)
const polishResult = ref<{ original: string; result: string } | null>(null)

function openPolish(scope: 'selection' | 'chapter') {
  polishScope.value = scope
  if (scope === 'selection' && !window.getSelection()?.toString()) {
    message.warning('请先选中要润色的内容'); return
  }
  showPolish.value = true; polishResult.value = null
}

async function handlePolish() {
  if (!currentChapter.value) return
  polishLoading.value = true
  try {
    const res = await aiApi.polish({
      content: polishScope.value === 'selection' ? (window.getSelection()?.toString() || '') : currentChapter.value.content,
      isSelection: polishScope.value === 'selection',
      outline: novel.value?.outline || '',
      requirement: polishRequirement.value,
    })
    polishResult.value = res
  } catch (e: any) { message.error(`润色失败: ${e.message}`) }
  finally { polishLoading.value = false }
}

function replaceWithPolish() {
  if (!polishResult.value) return
  if (polishScope.value === 'selection') {
    const sel = window.getSelection()
    if (sel?.rangeCount) {
      sel.getRangeAt(0).deleteContents()
      sel.getRangeAt(0).insertNode(document.createTextNode(polishResult.value.result))
      contentChanged.value = true
    }
  } else { setEditorContent(polishResult.value.result) }
  message.success('已替换原文'); showPolish.value = false
}

function copyPolishResult() {
  if (polishResult.value) navigator.clipboard.writeText(polishResult.value.result).then(() => message.success('已复制'))
}

// AI 扩写
const showExpand = ref(false)
const expandScope = ref<'selection' | 'chapter'>('chapter')
const expandRequirement = ref('')
const expandLoading = ref(false)
const expandResult = ref<{ original: string; result: string } | null>(null)

function openExpand(scope: 'selection' | 'chapter') {
  expandScope.value = scope
  if (scope === 'selection' && !window.getSelection()?.toString()) {
    message.warning('请先选中要扩写的内容'); return
  }
  showExpand.value = true; expandResult.value = null
}

async function handleExpand() {
  if (!currentChapter.value) return
  expandLoading.value = true
  try {
    const res = await aiApi.expand({
      content: expandScope.value === 'selection' ? (window.getSelection()?.toString() || '') : currentChapter.value.content,
      isSelection: expandScope.value === 'selection',
      outline: novel.value?.outline || '',
      requirement: expandRequirement.value,
    })
    expandResult.value = res
  } catch (e: any) { message.error(`扩写失败: ${e.message}`) }
  finally { expandLoading.value = false }
}

function replaceWithExpand() {
  if (!expandResult.value) return
  if (expandScope.value === 'selection') {
    const sel = window.getSelection()
    if (sel?.rangeCount) {
      sel.getRangeAt(0).deleteContents()
      sel.getRangeAt(0).insertNode(document.createTextNode(expandResult.value.result))
      contentChanged.value = true
    }
  } else { setEditorContent(expandResult.value.result) }
  message.success('已替换原文'); showExpand.value = false
}

function copyExpandResult() {
  if (expandResult.value) navigator.clipboard.writeText(expandResult.value.result).then(() => message.success('已复制'))
}

// === 设置 AI ===
const showAISetup = ref(false)
const setupEndpoint = ref('https://api.deepseek.com')
const setupApiKey = ref('')
const setupModel = ref('deepseek-chat')
const setupHasKey = ref(false)

const modelOptions = [
  { label: 'DeepSeek Chat', value: 'deepseek-chat' },
  { label: 'DeepSeek V3', value: 'deepseek-chat' },
  { label: 'DeepSeek R1', value: 'deepseek-reasoner' },
  { label: 'GPT-4o', value: 'gpt-4o' },
  { label: 'GPT-4o Mini', value: 'gpt-4o-mini' },
  { label: 'Claude Sonnet', value: 'claude-sonnet-4-20250514' },
  { label: '自定义', value: '__custom__' },
]

const showCustomModel = ref(false)
const showChangeKey = ref(false)

watch(setupModel, (val) => { showCustomModel.value = val === '__custom__' })

watch(showAISetup, async (open) => {
  if (!open) return
  if (!settingsStore.settings) await settingsStore.load()
  if (settingsStore.settings) {
    setupEndpoint.value = settingsStore.settings.endpoint || 'https://api.deepseek.com'
    setupModel.value = settingsStore.settings.aiModel || 'deepseek-chat'
    setupHasKey.value = settingsStore.settings.hasApiKey
  }
  setupApiKey.value = ''
  showCustomModel.value = !modelOptions.some(o => o.value === setupModel.value)
  showChangeKey.value = false
})

async function saveAISetup() {
  const form: Record<string, any> = { endpoint: setupEndpoint.value, aiModel: setupModel.value }
  if (setupApiKey.value) form.apiKey = setupApiKey.value
  const ok = await settingsStore.update(form as any)
  if (ok) {
    if (setupApiKey.value) setupHasKey.value = true
    message.success('AI 设置已保存'); showAISetup.value = false
  }
}
</script>

<template>
  <n-layout has-sider style="height: 100vh">
    <!-- 左侧章节列表 -->
    <n-layout-sider
      bordered :width="240" :collapsed-width="48"
      show-trigger="arrow-circle" collapse-mode="width"
      :collapsed="siderCollapsed"
      @update:collapsed="siderCollapsed = $event"
      style="height: 100vh; background: #fafafa;"
    >
      <div class="chapter-sidebar" :class="{ collapsed: siderCollapsed }">
        <div class="chapter-header">
          <div class="header-back-btn" title="返回小说列表" @click="goBack">
            <n-icon size="22"><BackIcon/></n-icon>
            <span class="back-label">返回</span>
          </div>
          <n-text class="header-title" :title="novel?.title">
            {{ novel?.title || '加载中...' }}
          </n-text>
        </div>
        <n-divider v-if="!siderCollapsed" style="margin: 8px 0"/>
        <n-scrollbar style="flex: 1;">
          <div class="chapter-list" :class="{ dragging: dragIndex !== null }">
            <div
              v-for="(ch, i) in chapters" :key="ch.id"
              class="chapter-item"
              :class="{
                active: currentChapter?.id === ch.id,
                'drag-over': dragOverIndex === i,
                'drag-to-end': dragOverIndex === chapters.length && i === chapters.length - 1,
                dragging: dragIndex === i,
              }"
              :draggable="true"
              @click="selectChapter(ch)"
              @contextmenu="(e) => openContextMenu(e, ch)"
              @dragstart="handleDragStart(i)"
              @dragover="(e) => handleDragOver(e, i)"
              @dragleave="handleDragLeave"
              @drop="handleDrop"
              @dragend="handleDragEnd"
            >
              <div class="chapter-index">{{ i + 1 }}</div>
              <div class="chapter-info">
                <n-text ellipsis style="font-size: 13px;" @dblclick.stop="startRenameChapter(ch)">
                  {{ ch.title || `第${i + 1}章` }}
                </n-text>
                <n-text depth="3" style="font-size: 11px; line-height: 1.4;">{{ ch.wordCount }} 字</n-text>
              </div>
              <n-button text size="tiny" class="chapter-del-btn" @click.stop="confirmDeleteChapter(ch)">
                <template #icon><n-icon size="14"><DeleteIcon/></n-icon></template>
              </n-button>
            </div>
            <div
              class="chapter-drop-zone"
              :class="{ 'drag-over': dragOverIndex === chapters.length }"
              @dragover="(e) => handleDragOver(e, chapters.length)"
              @dragleave="handleDragLeave" @drop="handleDrop"
            />
          </div>
        </n-scrollbar>
        <div class="chapter-footer">
          <n-button class="add-chapter-btn" :block="!siderCollapsed" :ghost="!siderCollapsed" size="small" @click="showAddChapterModal = true" title="添加章节">
            <template #icon><n-icon size="18"><AddChapterIcon/></n-icon></template>
            <span v-if="!siderCollapsed">添加章节</span>
          </n-button>
        </div>
      </div>
      <n-dropdown trigger="manual" placement="bottom-start" :show="contextMenuShow" :x="contextMenuX" :y="contextMenuY" :options="contextMenuOptions" @select="handleContextMenuSelect"/>
    </n-layout-sider>

    <!-- 右侧编辑区 -->
    <div style="flex: 1; min-width: 0; min-height: 0; display: flex; flex-direction: column; overflow: hidden;">
      <!-- 工具栏 -->
      <div style="border-bottom: 1px solid #eee; padding: 4px 16px; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; flex-shrink: 0;">
        <n-button quaternary size="tiny" :disabled="!undoable" @click="undo">
          <template #icon><n-icon size="16"><UndoIcon/></n-icon></template>
        </n-button>
        <n-button quaternary size="tiny" :disabled="!redoable" @click="redo">
          <template #icon><n-icon size="16"><RedoIcon/></n-icon></template>
        </n-button>
        <n-divider vertical/>
        <n-select v-model:value="fontFamily" :options="fontOptions" size="small" style="width: 120px;" placeholder="字体"/>
        <n-divider vertical/>
        <n-text depth="3" style="font-size: 12px; white-space: nowrap;">字号</n-text>
        <n-input-number v-model:value="fontSize" :min="12" :max="32" size="small" style="width: 82px;"/>
        <n-divider vertical/>
        <n-text depth="3" style="font-size: 12px; white-space: nowrap;">行距</n-text>
        <n-input-number v-model:value="lineHeight" :min="1" :max="3" :step="0.1" size="small" style="width: 82px;"/>
        <n-divider vertical/>
        <n-text depth="3" style="font-size: 12px; white-space: nowrap;">段距</n-text>
        <n-input-number v-model:value="paragraphSpacing" :min="0" :max="48" :step="4" size="small" style="width: 82px;"/>
        <n-divider vertical/>
        <n-button quaternary size="tiny" :type="isBold ? 'primary' : 'default'" @click="isBold = !isBold"><b>B</b></n-button>
        <n-button quaternary size="tiny" :type="isItalic ? 'primary' : 'default'" @click="isItalic = !isItalic"><i>I</i></n-button>
        <n-divider vertical/>
        <n-button quaternary size="tiny" @click="copySelection">
          <template #icon><n-icon size="16"><CopyIcon/></n-icon></template>复制
        </n-button>
        <n-button quaternary size="tiny" @click="showSearch = !showSearch">
          <template #icon><n-icon size="16"><SearchIcon/></n-icon></template>搜索
        </n-button>
        <n-button quaternary size="tiny" @click="formatContent">
          <template #icon><n-icon size="16"><CutIcon/></n-icon></template>格式化
        </n-button>
        <template v-if="!settingsStore.settings?.autoSave">
          <n-button type="warning" size="tiny" :disabled="!contentChanged" @click="saveCurrentChapter">
            <template #icon><n-icon size="16"><SaveIcon/></n-icon></template>保存
          </n-button>
          <n-divider vertical/>
        </template>
        <template v-if="showSearch">
          <n-divider vertical/>
          <n-input v-model:value="searchQuery" placeholder="搜索正文..." size="small" style="width: 200px" @keyup.enter="handleSearch"/>
        </template>
        <div style="flex: 1"/>
        <n-text v-if="contentChanged || showSavedIndicator" :style="{ fontSize: '12px', color: contentChanged ? '#e6a23c' : '#18a058' }">
          {{ contentChanged ? '未保存' : '已保存' }}
        </n-text>
      </div>

      <!-- 正文编辑区 -->
      <div style="flex: 1; overflow: hidden; min-height: 0; display: flex;">
        <div
          v-if="currentChapter"
          class="editor-content"
          :style="{
            fontSize: fontSize + 'px',
            lineHeight: lineHeight,
            fontWeight: isBold ? 'bold' : 'normal',
            fontStyle: isItalic ? 'italic' : 'normal',
            fontFamily: fontFamily || undefined,
            '--p-gap': paragraphSpacing + 'px',
          }"
        >
          <editor-content :editor="editor" class="content-editable" />
        </div>
        <div v-else class="editor-empty">
          <n-text depth="3">还没有章节，请创建第一章</n-text>
        </div>
      </div>

      <!-- 底栏 -->
      <div style="border-top: 1px solid #eee; padding: 6px 16px; display: flex; align-items: center; gap: 8px; flex-shrink: 0;">
        <n-text depth="3" style="font-size: 12px;">共 {{ wordCount }} 字</n-text>
        <div style="flex: 1"/>
        <template v-if="aiConfigured">
          <n-button size="tiny" secondary @click="showContinueWrite = true">
            <template #icon><n-icon size="14"><SparklesIcon/></n-icon></template>AI续写
          </n-button>
          <n-dropdown trigger="click" :options="[
            { label: '润色选中内容', key: 'polish-selection' },
            { label: '润色整章', key: 'polish-chapter' },
          ]" @select="(key) => key === 'polish-selection' ? openPolish('selection') : openPolish('chapter')">
            <n-button size="tiny" secondary>AI润色</n-button>
          </n-dropdown>
          <n-dropdown trigger="click" :options="[
            { label: '扩写选中内容', key: 'expand-selection' },
            { label: '扩写整章', key: 'expand-chapter' },
          ]" @select="(key) => key === 'expand-selection' ? openExpand('selection') : openExpand('chapter')">
            <n-button size="tiny" secondary>AI扩写</n-button>
          </n-dropdown>
        </template>
        <template v-else>
          <n-button size="tiny" secondary @click="showAISetup = true">
            <template #icon><n-icon size="14"><AISetupIcon/></n-icon></template>设置 AI
          </n-button>
        </template>
      </div>
    </div>

    <!-- AI 续写弹框 -->
    <n-modal class="dialog-modal" v-model:show="showContinueWrite" preset="card" title="AI 续写" style="width: 500px" :mask-closable="false">
      <div v-if="!showContinueResult">
        <n-form label-placement="top">
          <n-form-item label="续写要求（可选）">
            <n-input v-model:value="continueRequirement" type="textarea" placeholder="输入对续写内容的要求、方向或风格..." :rows="4"/>
          </n-form-item>
        </n-form>
      </div>
      <div v-else>
        <n-alert type="success" :bordered="false" style="margin-bottom: 12px">续写完成，共 {{ continueResult.length }} 字</n-alert>
        <n-scrollbar style="max-height: 300px; border: 1px solid #eee; border-radius: 4px; padding: 12px;">
          <n-text>{{ continueResult }}</n-text>
        </n-scrollbar>
      </div>
      <template #footer>
        <n-space justify="end">
          <template v-if="!showContinueResult">
            <n-button quaternary @click="showContinueWrite = false">取消</n-button>
            <n-button type="primary" :loading="continueLoading" @click="handleContinueWrite">开始续写</n-button>
          </template>
          <template v-else>
            <n-button quaternary @click="copyContinueResult"><template #icon><n-icon><CopyIcon/></n-icon></template>复制</n-button>
            <n-button quaternary @click="insertToChapterEnd">插入当前章节末尾</n-button>
            <n-button type="primary" @click="insertAsNewChapter">新建章节</n-button>
          </template>
        </n-space>
      </template>
    </n-modal>

    <!-- AI 润色弹框 -->
    <n-modal class="dialog-modal" v-model:show="showPolish" preset="card" title="AI 润色" style="width: 800px" :mask-closable="false">
      <div v-if="polishScope === 'selection' && !polishResult" style="margin-bottom: 12px">
        <n-alert type="info" :bordered="false">当前选中内容将被润色</n-alert>
      </div>
      <n-form v-if="!polishResult" label-placement="top">
        <n-form-item label="润色要求（可选）">
          <n-input v-model:value="polishRequirement" type="textarea" placeholder="输入润色方向、风格要求..." :rows="3"/>
        </n-form-item>
      </n-form>
      <div v-if="polishResult">
        <n-grid :cols="2" :x-gap="12">
          <n-gi>
            <n-text strong depth="2">原文</n-text>
            <div style="max-height: 300px; border: 1px solid #eee; border-radius: 4px; padding: 12px; margin-top: 8px; overflow-y: auto;"><n-text>{{ polishResult.original }}</n-text></div>
          </n-gi>
          <n-gi>
            <n-text strong depth="2">润色后</n-text>
            <div style="max-height: 300px; border: 1px solid #2080f0; border-radius: 4px; padding: 12px; margin-top: 8px; overflow-y: auto;">
              <pre contenteditable="true" @input="polishResult.result = ($event.target as HTMLElement).innerText" style="outline: none; white-space: pre-wrap; font-family: inherit; margin: 0;">{{ polishResult.result }}</pre>
            </div>
          </n-gi>
        </n-grid>
      </div>
      <template #footer>
        <n-space justify="end">
          <template v-if="!polishResult">
            <n-button quaternary @click="showPolish = false">取消</n-button>
            <n-button type="primary" :loading="polishLoading" @click="handlePolish">开始润色</n-button>
          </template>
          <template v-else>
            <n-button quaternary @click="showPolish = false">关闭</n-button>
            <n-button quaternary @click="copyPolishResult"><template #icon><n-icon><CopyIcon/></n-icon></template>复制</n-button>
            <n-button type="primary" @click="replaceWithPolish">替换原文</n-button>
          </template>
        </n-space>
      </template>
    </n-modal>

    <!-- AI 扩写弹框 -->
    <n-modal class="dialog-modal" v-model:show="showExpand" preset="card" title="AI 扩写" style="width: 800px" :mask-closable="false">
      <n-form v-if="!expandResult" label-placement="top">
        <n-form-item label="扩写要求（可选）">
          <n-input v-model:value="expandRequirement" type="textarea" placeholder="输入扩写方向、需要丰富的内容..." :rows="3"/>
        </n-form-item>
      </n-form>
      <div v-if="expandResult">
        <n-grid :cols="2" :x-gap="12">
          <n-gi>
            <n-text strong depth="2">原文</n-text>
            <div style="max-height: 300px; border: 1px solid #eee; border-radius: 4px; padding: 12px; margin-top: 8px; overflow-y: auto;"><n-text>{{ expandResult.original }}</n-text></div>
          </n-gi>
          <n-gi>
            <n-text strong depth="2">扩写后</n-text>
            <div style="max-height: 300px; border: 1px solid #2080f0; border-radius: 4px; padding: 12px; margin-top: 8px; overflow-y: auto;">
              <pre contenteditable="true" @input="expandResult.result = ($event.target as HTMLElement).innerText" style="outline: none; white-space: pre-wrap; font-family: inherit; margin: 0;">{{ expandResult.result }}</pre>
            </div>
          </n-gi>
        </n-grid>
      </div>
      <template #footer>
        <n-space justify="end">
          <template v-if="!expandResult">
            <n-button quaternary @click="showExpand = false">取消</n-button>
            <n-button type="primary" :loading="expandLoading" @click="handleExpand">开始扩写</n-button>
          </template>
          <template v-else>
            <n-button quaternary @click="showExpand = false">关闭</n-button>
            <n-button quaternary @click="copyExpandResult"><template #icon><n-icon><CopyIcon/></n-icon></template>复制</n-button>
            <n-button type="primary" @click="replaceWithExpand">替换原文</n-button>
          </template>
        </n-space>
      </template>
    </n-modal>

    <!-- 添加章节弹框 -->
    <n-modal class="dialog-modal" v-model:show="showAddChapterModal" title="添加章节" preset="card" style="width: 360px" :mask-closable="false">
      <n-form label-placement="top">
        <n-form-item label="章节标题" required>
          <n-input v-model:value="newChapterTitle" placeholder="输入章节标题" @keyup.enter="handleAddChapter"/>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showAddChapterModal = false">取消</n-button>
          <n-button type="primary" :loading="creatingChapter" @click="handleAddChapter">创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 重命名章节弹框 -->
    <n-modal class="dialog-modal" v-model:show="showRenameModal" title="重命名章节" preset="card" style="width: 360px" :mask-closable="false">
      <n-form label-placement="top">
        <n-form-item label="章节标题" required>
          <n-input v-model:value="renameChapterTitle" placeholder="输入章节标题" @keyup.enter="saveRenameTitle"/>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showRenameModal = false">取消</n-button>
          <n-button type="primary" @click="saveRenameTitle">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 删除章节确认弹框 -->
    <n-modal class="dialog-modal" :show="showDeleteModal" title="删除章节" preset="card" style="width: 360px" :mask-closable="false" @update:show="showDeleteModal = $event">
      <n-text>确定删除「{{ deleteTargetChapter?.title || (deleteTargetChapter ? `第${chapters.indexOf(deleteTargetChapter) + 1}章` : '') }}」吗？此操作不可撤销。</n-text>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showDeleteModal = false">取消</n-button>
          <n-button type="error" :loading="deletingChapter" @click="handleDeleteConfirm">删除</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- AI 设置弹框 -->
    <n-modal class="dialog-modal" v-model:show="showAISetup" title="配置 AI 提供商" preset="card" style="width: 480px" :mask-closable="false">
      <n-alert type="info" :bordered="false" style="margin-bottom: 12px">
        需要配置 AI 接口才能使用智能续写、润色等功能。目前支持 DeepSeek 兼容接口。
      </n-alert>
      <n-form label-placement="top">
        <n-form-item label="接口地址" required>
          <n-input v-model:value="setupEndpoint" placeholder="https://api.deepseek.com"/>
        </n-form-item>
        <n-form-item label="模型">
          <n-select v-model:value="setupModel" :options="modelOptions" placeholder="选择模型"/>
        </n-form-item>
        <n-form-item v-if="showCustomModel" label="自定义模型名">
          <n-input v-model:value="setupModel" placeholder="输入模型名称"/>
        </n-form-item>
        <n-form-item v-if="setupHasKey && !showChangeKey" label="API Key">
          <div style="display: flex; align-items: center; gap: 8px; width: 100%;">
            <n-text style="color: #18a058;">✓ API Key 已配置</n-text>
            <n-button size="tiny" quaternary @click="showChangeKey = true">更换</n-button>
          </div>
        </n-form-item>
        <n-form-item v-else label="API Key">
          <n-input v-model:value="setupApiKey" type="password" show-password-on="click" placeholder="输入新的 API Key"/>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-button quaternary @click="showAISetup = false">暂不配置</n-button>
        <n-button type="primary" @click="saveAISetup">保存</n-button>
      </template>
    </n-modal>
  </n-layout>
</template>

<style scoped>
.chapter-sidebar {
  height: 100%; display: flex; flex-direction: column; min-height: 0;
  .chapter-header {
    padding: 14px 16px; display: flex; align-items: center; gap: 10px; flex-shrink: 0;
    border-bottom: 1px solid #eee; background: #fff;
    .header-back-btn {
      display: flex; align-items: center; gap: 4px; padding: 6px 12px; border-radius: 8px;
      cursor: pointer; color: #666; transition: all 0.2s; user-select: none; flex-shrink: 0;
      &:hover { background: #f0f0f0; color: #333; }
      &:active { background: #e8e8e8; }
      .back-label { font-size: 13px; }
    }
    .header-title {
      font-size: 16px; font-weight: 600; overflow: hidden; text-overflow: ellipsis;
      white-space: nowrap; color: #333;
    }
  }
  .chapter-footer { padding: 8px; flex-shrink: 0; border-top: 1px solid #eee; }
  .add-chapter-btn { font-size: 13px; }
  .chapter-list { padding: 4px 8px;
    &.dragging .chapter-drop-zone { height: 6px; background: rgba(32,128,240,0.04); border: 1px dashed rgba(32,128,240,0.2); }
  }
  .chapter-item {
    display: flex; align-items: center; gap: 8px; padding: 8px 10px;
    border-radius: 6px; cursor: pointer; transition: background 0.15s;
    &:hover { background: #e8f0fe; }
    &.active { background: #d4e8ff; }
    &.dragging { opacity: 0.4; }
    &.drag-over { border-top: 2px solid #2080f0; }
    &.drag-to-end { border-bottom: 2px solid #2080f0; }
    .chapter-index { width: 28px; height: 28px; border-radius: 50%; background: #e8e8e8; display: flex; align-items: center; justify-content: center; font-size: 12px; color: #666; flex-shrink: 0; }
    &.active .chapter-index { background: #2080f0; color: #fff; }
    .chapter-info { overflow: hidden; flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
    .chapter-del-btn { opacity: 0; transition: opacity 0.15s; }
    &:hover .chapter-del-btn { opacity: 1; }
  }
  .chapter-drop-zone { height: 4px; margin: 0 8px; border-radius: 4px; transition: all 0.15s;
    &.drag-over { height: 32px; margin: 4px 8px; background: rgba(32,128,240,0.06); border: 2px dashed #2080f0; border-radius: 6px; display: flex; align-items: center; justify-content: center; color: #2080f0; font-size: 12px;
      &::after { content: '移至末尾'; }
    }
  }
  &.collapsed {
    .chapter-header { justify-content: center; padding: 12px 0;
      .header-title { display: none; }
      .back-label { display: none; }
    }
    .chapter-list { padding: 0; }
    .chapter-item { justify-content: center; padding: 6px 0; gap: 0; }
    .chapter-info, .chapter-del-btn { display: none; }
    .chapter-footer { display: flex; padding: 6px 0; border-top: none; justify-content: center; .n-button { padding-inline: 8px; } }
    .add-chapter-btn { padding: 0 4px; height: 32px; border: none; }
    .chapter-index { width: 28px; height: 28px; font-size: 12px; }
    .chapter-drop-zone, .chapter-list.dragging .chapter-drop-zone { display: none; }
  }
}
.chapter-index { font-variant-numeric: tabular-nums; min-width: 28px; }

.editor-content { flex: 1; overflow-y: auto; background: #f0f2f5; }

/* Tiptap 编辑器容器 */
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

/* 段首缩进（适用于 Tiptap 生成的 <p> 元素） */
.content-editable :deep(p) {
  text-indent: 2em;
  margin-bottom: var(--p-gap, 16px);
}
.content-editable :deep(p:last-child) {
  margin-bottom: 0;
}

.editor-empty { flex: 1; display: flex; align-items: center; justify-content: center; background: #f0f2f5; }

/* 去掉 ProseMirror 默认 focus 黑框 */
.content-editable :deep(.ProseMirror),
.content-editable :deep(.ProseMirror-focused),
.content-editable :deep(.ProseMirror:focus) {
  outline: none !important;
  border: none !important;
  box-shadow: none !important;
}
</style>

<style>
/* 弹框样式统一嵌套 */
.dialog-modal {
  .n-card-header { border-bottom: 1px solid rgba(0,0,0,0.14); padding: 0 0 0 16px !important; height: 40px; }
  .n-card-header__close { height: 40px; width: 40px; border-radius: 0; color: #999; transition: all 0.15s;
    &:hover { background: #d03050;
      &::before { --n-close-color-hover: #d03050; --n-close-color-pressed: #d03050; }
    }
    &:hover, &:hover * { color: #fff; opacity: 1 !important; }
  }
  .n-card-content { padding: 20px 24px !important; }
  .n-card__footer { padding: 0 !important; border-top: 1px solid #eee;
    .n-space { width: 100%; gap: 1px !important; flex-flow: nowrap !important; & > div { width: 100%; } }
    .n-space-item { flex: 1; display: flex; }
    .n-button { width: 100%; border-radius: 0; height: 36px;
      &:not(:last-child) { border-right: 1px solid #eee; }
      /* 去掉 NaiveUI 默认的 focus 蓝框，避免误以为按钮处于激活态 */
      &:focus, &:focus-visible,
      &:focus-within { box-shadow: none !important; outline: none !important; }
      /* quaternary 按钮在 dialog 中的显式 hover 态 */
      &.n-button--quaternary-type {
        &:hover { background: #f5f5f5; }
        &:active { background: #ebebeb; }
      }
    }
  }
}
</style>

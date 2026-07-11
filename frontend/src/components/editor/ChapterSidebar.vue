<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted } from 'vue'
import { useNovelStore } from '../../stores/novel'
import type { Chapter } from '../../types'
import {
  NButton, NDivider, NText, NIcon, NLayoutSider, NModal, NForm, NFormItem,
  NInput, NScrollbar, NDropdown, NSpace, useMessage,
} from 'naive-ui'
import {
  ChevronBackOutline as BackIcon,
  AddCircleOutline as AddChapterIcon,
  TrashOutline as DeleteIcon,
} from '@vicons/ionicons5'

defineProps<{ siderCollapsed: boolean }>()
const emit = defineEmits<{
  'update:siderCollapsed': [v: boolean]
  goBack: []
}>()

const novelStore = useNovelStore()
const message = useMessage()

// === 章节选择 ===
function selectChapter(chapter: Chapter) {
  if (contentChanged.value) saveCurrentChapter()
  novelStore.selectChapter(chapter)
}

// === 右键菜单 ===
const contextMenuShow = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextMenuChapter = ref<Chapter | null>(null)
const contextMenuOptions = [
  { label: '重命名', key: 'rename' },
  { label: '删除', key: 'delete' },
]

function openContextMenu(e: MouseEvent, ch: Chapter) {
  e.preventDefault(); e.stopPropagation()
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
  if (!ch) return; closeContextMenu()
  if (key === 'rename') startRenameChapter(ch)
  else if (key === 'delete') confirmDeleteChapter(ch)
}

function onMenuGlobalClose(e: Event) {
  if (!contextMenuShow.value) return
  if (e.type === 'keydown' && (e as KeyboardEvent).key !== 'Escape') return
  closeContextMenu()
}

// === 自动保存（章节切换时） ===
const contentChanged = ref(false)

async function saveCurrentChapter() {
  const ch = novelStore.currentChapter
  if (!ch || !contentChanged.value) return
  const ok = await novelStore.updateChapter(ch.id, { content: '' })
  if (ok) contentChanged.value = false
}

// === 添加章节 ===
const showAddChapterModal = ref(false)
const newChapterTitle = ref('')
const creatingChapter = ref(false)

async function handleAddChapter() {
  if (!novelStore.currentNovel || !newChapterTitle.value.trim()) {
    message.warning('请输入章节标题'); return
  }
  creatingChapter.value = true
  const ch = await novelStore.createChapter({
    novelId: novelStore.currentNovel.id,
    title: newChapterTitle.value.trim(),
    content: '',
    order: novelStore.chapters.length + 1,
  })
  creatingChapter.value = false
  if (ch) {
    message.success('章节创建成功')
    showAddChapterModal.value = false
    newChapterTitle.value = ''
    novelStore.selectChapter(ch)
  }
}

// === 重命名章节 ===
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
  await novelStore.updateChapter(renameChapterId.value, { title: renameChapterTitle.value.trim() })
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
  if (i === novelStore.chapters.length - 1) {
    const target = e.currentTarget as HTMLElement
    if (target) {
      const rect = target.getBoundingClientRect()
      if (e.clientY - rect.top > rect.height * 0.55) {
        dragOverIndex.value = novelStore.chapters.length; return
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
  const to = dragOverIndex.value ?? novelStore.chapters.length - 1
  if (from === to) { dragIndex.value = null; dragOverIndex.value = null; return }
  const list = [...novelStore.chapters]
  const [moved] = list.splice(from, 1)
  const adjustedTo = to > from ? to - 1 : to
  list.splice(adjustedTo, 0, moved)
  const nid = novelStore.currentNovel?.id
  if (!nid) return
  if (await novelStore.reorderChapters(nid, list.map(ch => ch.id))) {
    await novelStore.loadChapters(nid)
  }
  dragIndex.value = null; dragOverIndex.value = null
}

function handleDragEnd() { dragIndex.value = null; dragOverIndex.value = null }

// === 生命周期 ===
onMounted(() => {
  document.addEventListener('click', onMenuGlobalClose)
  document.addEventListener('contextmenu', onMenuGlobalClose)
  document.addEventListener('keydown', onMenuGlobalClose)
  document.addEventListener('wheel', onMenuGlobalClose, { passive: true })
})

onUnmounted(() => {
  document.removeEventListener('click', onMenuGlobalClose)
  document.removeEventListener('contextmenu', onMenuGlobalClose)
  document.removeEventListener('keydown', onMenuGlobalClose)
  document.removeEventListener('wheel', onMenuGlobalClose as EventListener)
})
</script>

<template>
  <n-layout-sider bordered :width="240" :collapsed-width="48"
    show-trigger="arrow-circle" collapse-mode="width"
    :collapsed="siderCollapsed"
    @update:collapsed="emit('update:siderCollapsed', $event)"
    style="height: 100vh; background: #fafafa;">
    <div class="chapter-sidebar" :class="{ collapsed: siderCollapsed }">
      <div class="chapter-header">
        <div class="header-back-btn" title="返回小说列表" @click="emit('goBack')">
          <n-icon size="22"><BackIcon /></n-icon>
          <span class="back-label">返回</span>
        </div>
        <n-text class="header-title" :title="novelStore.currentNovel?.title">
          {{ novelStore.currentNovel?.title || '加载中...' }}
        </n-text>
      </div>
      <n-divider v-if="!siderCollapsed" style="margin: 8px 0" />
      <n-scrollbar style="flex: 1;">
        <div class="chapter-list" :class="{ dragging: dragIndex !== null }">
          <div v-for="(ch, i) in novelStore.chapters" :key="ch.id"
            class="chapter-item"
            :class="{
              active: novelStore.currentChapter?.id === ch.id,
              'drag-over': dragOverIndex === i,
              'drag-to-end': dragOverIndex === novelStore.chapters.length && i === novelStore.chapters.length - 1,
              dragging: dragIndex === i,
            }"
            :draggable="true"
            @click="selectChapter(ch)"
            @contextmenu="(e) => openContextMenu(e, ch)"
            @dragstart="handleDragStart(i)"
            @dragover="(e) => handleDragOver(e, i)"
            @dragleave="handleDragLeave"
            @drop="handleDrop"
            @dragend="handleDragEnd">
            <div class="chapter-index">{{ i + 1 }}</div>
            <div class="chapter-info">
              <n-text ellipsis style="font-size: 13px; white-space: nowrap;" @dblclick.stop="startRenameChapter(ch)">
                {{ ch.title || `第${i + 1}章` }}
              </n-text>
              <n-text depth="3" style="font-size: 11px; line-height: 1.4; white-space: nowrap;">{{ ch.wordCount }} 字</n-text>
            </div>
            <n-button text size="tiny" class="chapter-del-btn" @click.stop="confirmDeleteChapter(ch)">
              <template #icon><n-icon size="14"><DeleteIcon /></n-icon></template>
            </n-button>
          </div>
          <div class="chapter-drop-zone"
            :class="{ 'drag-over': dragOverIndex === novelStore.chapters.length }"
            @dragover="(e) => handleDragOver(e, novelStore.chapters.length)"
            @dragleave="handleDragLeave" @drop="handleDrop" />
        </div>
      </n-scrollbar>
      <div class="chapter-footer">
        <n-button class="add-chapter-btn" :block="!siderCollapsed" :ghost="!siderCollapsed"
          size="small" @click="showAddChapterModal = true" title="添加章节">
          <template #icon><n-icon size="18"><AddChapterIcon /></n-icon></template>
          <span v-if="!siderCollapsed">添加章节</span>
        </n-button>
      </div>
    </div>

    <!-- 右键菜单 -->
    <n-dropdown trigger="manual" placement="bottom-start" :show="contextMenuShow"
      :x="contextMenuX" :y="contextMenuY" :options="contextMenuOptions"
      @select="handleContextMenuSelect" />

    <!-- 添加章节弹框 -->
    <n-modal class="dialog-modal" v-model:show="showAddChapterModal" title="添加章节" preset="card"
      style="width: 360px" :mask-closable="false">
      <n-form label-placement="top">
        <n-form-item label="章节标题" required>
          <n-input v-model:value="newChapterTitle" placeholder="输入章节标题" @keyup.enter="handleAddChapter" />
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
    <n-modal class="dialog-modal" v-model:show="showRenameModal" title="重命名章节" preset="card"
      style="width: 360px" :mask-closable="false">
      <n-form label-placement="top">
        <n-form-item label="章节标题" required>
          <n-input v-model:value="renameChapterTitle" placeholder="输入章节标题" @keyup.enter="saveRenameTitle" />
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
    <n-modal class="dialog-modal" :show="showDeleteModal" title="删除章节" preset="card"
      style="width: 360px" :mask-closable="false"
      @update:show="showDeleteModal = $event">
      <n-text>确定删除「{{ deleteTargetChapter?.title || (deleteTargetChapter ? `第${novelStore.chapters.indexOf(deleteTargetChapter) + 1}章` : '') }}」吗？此操作不可撤销。</n-text>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showDeleteModal = false">取消</n-button>
          <n-button type="error" :loading="deletingChapter" @click="handleDeleteConfirm">删除</n-button>
        </n-space>
      </template>
    </n-modal>
  </n-layout-sider>
</template>

<style scoped>
.chapter-sidebar {
  height: 100%; display: flex; flex-direction: column; min-height: 0;
  overflow: hidden; /* 防止伸缩时内容被挤压形变 */
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
</style>

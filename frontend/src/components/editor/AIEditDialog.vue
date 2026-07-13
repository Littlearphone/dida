<script setup lang="ts">
import { ref, inject, watch, computed, onUnmounted } from 'vue'
import {
  NModal, NAlert, NForm, NFormItem, NInput, NButton, NGrid, NGi,
  NSpace, NIcon, NProgress, useMessage,
} from 'naive-ui'
import { CopyOutline as CopyIcon, SyncOutline as ReplaceIcon, CloseOutline as CloseIcon } from '@vicons/ionicons5'
import { useNovelStore } from '../../stores/novel'
import { useAIStream } from '../../composables/useAIStream'
import { htmlToPlainText, normalizeParagraphs } from '../../utils/editor'
import * as aiApi from '../../api/ai'
import { EDITOR_ACTIONS_KEY } from '../../types/editor'

const props = defineProps<{
  show: boolean
  mode: 'polish' | 'expand'
  /** 外部传入待处理内容（如续写结果的二次处理），此时不再使用编辑器选区/章节内容 */
  externalContent?: string
}>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  /** 外部传入内容的处理结果，由父组件转发回源头（如续写框） */
  'replaceExternal': [text: string]
}>()

const novelStore = useNovelStore()
const message = useMessage()
const editorActions = inject(EDITOR_ACTIONS_KEY)!

const requirement = ref('')
const originalContent = ref('')
const editResult = ref('')
const showResult = ref(false)
const editingResult = ref(false)
const hasSelection = ref(false)
const savedSelectionText = ref('')

const {
  loading, progress, progressText,
  MAX_RETRIES,
  getAbortSignal: getSignal, cleanupRequest, resetRetry,
  shouldRetry, isCanceled, incrementRetry,
  startProgressSimulation, stopProgressSimulation,
  completeProgress, errorProgress,
} = useAIStream()

function resetState() {
  requirement.value = ''
  editResult.value = ''
  originalContent.value = ''
  showResult.value = false
  editingResult.value = false
  hasSelection.value = false
  savedSelectionText.value = ''
}

const title = computed(() => props.mode === 'polish' ? 'AI 润色' : 'AI 扩写')
const resultLabel = computed(() => props.mode === 'polish' ? '润色后' : '扩写后')

// 弹框打开时通过编辑器保存选中文本（基于 ProseMirror 选区，不受焦点影响）
function checkSelection() {
  savedSelectionText.value = editorActions.getSelectionText?.()?.trim() || ''
  hasSelection.value = !!savedSelectionText.value
}

/** 使用外部传入内容作为处理对象（如续写结果的二次处理） */
function useExternalContent(text: string) {
  savedSelectionText.value = text
  hasSelection.value = true
  originalContent.value = text
  editResult.value = ''
  showResult.value = false
}

/** 清除选中状态，改为使用整章内容 */
function clearSelection() {
  hasSelection.value = false
  savedSelectionText.value = ''
}

function startProgress(label: string) {
  startProgressSimulation(props.mode === 'polish' ? '润色' : '扩写')
}

function cancelEdit() {
  cleanupRequest()
}

function stopProgress() {
  stopProgressSimulation()
}

async function handleEdit() {
  loading.value = true
  resetRetry()

  const rawContent = savedSelectionText.value
    ? savedSelectionText.value
    : (editorActions.getContent() || novelStore.currentChapter?.content || '')
  const plainContent = htmlToPlainText(rawContent)
  originalContent.value = plainContent

  editResult.value = ''
  showResult.value = true

  const apiFn = props.mode === 'polish' ? aiApi.polish : aiApi.expand
  const modeLabel = props.mode === 'polish' ? '润色' : '扩写'

  while (shouldRetry()) {
    if (isCanceled()) { closeDialog(); return }

    editResult.value = ''
    progress.value = 0
    startProgress(modeLabel)

    const signal = getSignal()

    try {
      const chapters = novelStore.chapters
      const currentCh = novelStore.currentChapter; const curIdx = currentCh
        ? chapters.findIndex(c => c.id === currentCh.id)
        : -1
      const prevCh = curIdx > 0 ? chapters[curIdx - 1] : null

      const fullText = await apiFn(
        {
          content: plainContent,
          isSelection: !!savedSelectionText.value,
          previousChapterContent: htmlToPlainText(prevCh?.content || ''),
          outline: novelStore.currentNovel?.outline || '',
          requirement: requirement.value,
          characters: novelStore.currentNovel?.characters,
          relationships: novelStore.currentNovel?.relationships,
          events: novelStore.currentNovel?.events,
        },
        (text: string) => {
          editResult.value = normalizeParagraphs(text)
          if (progress.value < 95) {
            progress.value = Math.max(progress.value, 70)
          }
        },
        signal,
      )
      stopProgressSimulation()

      if (fullText) {
        editResult.value = normalizeParagraphs(fullText)
        completeProgress(modeLabel)
        loading.value = false
        return
      }

      incrementRetry()
    } catch (e: any) {
      stopProgressSimulation()
      if (e.name === 'AbortError') { closeDialog(); return }
      if (shouldRetry() && !isCanceled()) { incrementRetry(); continue }
      message.error(`${modeLabel}失败: ${e.message}`)
      if (!editResult.value) { showResult.value = false }
      else { errorProgress(modeLabel) }
      loading.value = false
      return
    }
  }

  message.warning(`AI 连续返回空内容，${modeLabel}失败`)
  showResult.value = false
  loading.value = false
}

function replaceContent() {
  if (!editResult.value) return
  if (props.externalContent) {
    // 续写结果二次处理（润色/扩写）：将 AI 结果发回父组件，由父组件转填回续写框
    emit('replaceExternal', editResult.value)
    closeDialog()
    message.success('已替换续写框中的内容')
  } else if (savedSelectionText.value && editorActions.replaceSelection) {
    // 有选中 → 通过编辑器替换选区
    editorActions.replaceSelection(editResult.value)
    message.success('已替换原文')
    closeDialog()
  } else {
    // 整章 → 替换全部
    editorActions.setContent(editResult.value)
    message.success('已替换原文')
    closeDialog()
  }
}

function copyResult() {
  if (editResult.value) {
    navigator.clipboard.writeText(editResult.value)
    message.success('已复制')
  }
}

function closeDialog() {
  cleanupRequest()
  emit('update:show', false)
  resetState()
}

// 弹框打开时检测选中状态；关闭时重置状态（覆盖 Escape 等非按钮关闭途径）
watch(() => props.show, (open) => {
  if (open) {
    if (props.externalContent) {
      // 外部传入内容（续写二次处理）覆盖编辑器选区
      useExternalContent(props.externalContent)
    } else {
      checkSelection()
    }
  } else {
    cleanupRequest()
    resetState()
  }
})

onUnmounted(() => {
  stopProgressSimulation()
  cleanupRequest()
})
</script>

<template>
  <n-modal class="dialog-modal" :show="show" :title="title" preset="card" style="width: 80vw; height: 80vh;"
    :mask-closable="false" draggable @update:show="emit('update:show', $event)">
    <!-- 输入阶段：包裹在 div 中以匹配全局 CSS ".n-card-content > div" -->
    <div v-if="!showResult" style="display: flex; flex-direction: column; flex: 1; min-height: 0;">
      <!-- 有选中时显示提示和选中内容预览（可关闭取消选中，外部传入内容时禁用） -->
      <div v-if="hasSelection" style="flex-shrink: 0; margin-bottom: 12px">
        <n-alert type="info" :bordered="false" style="line-height: 1.6;">
          <template #header>
            <div style="display: flex; align-items: center; justify-content: space-between; width: 100%;">
              <span v-if="externalContent">将对外部内容进行{{ mode === 'polish' ? '润色' : '扩写' }}</span>
              <span v-else>当前选中内容将被{{ mode === 'polish' ? '润色' : '扩写' }}</span>
              <n-button v-if="!externalContent" tertiary circle size="tiny" @click.stop="clearSelection" title="取消选中，改用整章内容">
                <template #icon><n-icon size="14"><CloseIcon/></n-icon></template>
              </n-button>
            </div>
          </template>
          <div style="font-size: 12px; color: #666; white-space: pre-wrap; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical;">
            {{ savedSelectionText }}
          </div>
        </n-alert>
      </div>
      <n-form label-placement="top" style="display: flex; flex-direction: column; flex: 1; min-height: 0;">
        <n-form-item :label="`${mode === 'polish' ? '润色' : '扩写'}要求（可选）`" style="flex: 1; align-items: flex-start;">
          <n-input v-model:value="requirement" type="textarea"
            :placeholder="`输入${mode === 'polish' ? '润色' : '扩写'}方向、风格要求...`" :resizable="false" :maxlength="500" show-count
            style="height: 100%; min-height: 80px;" />
        </n-form-item>
      </n-form>
    </div>

    <!-- 结果阶段：始终双栏对比（原文 + AI 结果），扩写模式下也能对照查看 -->
    <div v-if="showResult" style="display: flex; flex-direction: column; flex: 1; min-height: 0;">
      <!-- 进度条 -->
      <div v-if="loading" style="flex-shrink: 0; margin-bottom: 12px;">
        <n-progress type="line" :percentage="Math.round(progress)" :indicator-placement="'inside'"
          :height="20" :border-radius="4" processing>
          {{ progressText }}
        </n-progress>
      </div>

      <!-- 双栏对比：左侧原文，右侧 AI 结果 -->
      <n-grid :cols="2" :x-gap="12" style="flex: 1; min-height: 0;">
        <n-gi style="display: flex; flex-direction: column; min-height: 0;">
          <n-alert type="info" :bordered="false" style="flex-shrink: 0; margin-bottom: 8px;">原文</n-alert>
          <div style="flex: 1; min-height: 0; overflow-y: auto; border: 1px solid #e0e0e0; border-radius: 6px; background: #fafafa;">
            <div style="padding: 16px; white-space: pre-wrap; line-height: 1.8; font-size: 14px; color: #666;">
              {{ originalContent }}
            </div>
          </div>
        </n-gi>
        <n-gi style="display: flex; flex-direction: column; min-height: 0;">
          <n-alert type="success" :bordered="false" style="flex-shrink: 0; margin-bottom: 8px;">{{ resultLabel }}</n-alert>
          <!-- 加载中/编辑模式/无内容 → 可编辑输入框，否则格式化预览 -->
          <n-input v-if="loading || editingResult || !editResult"
            v-model:value="editResult"
            type="textarea"
            :disabled="loading"
            placeholder="等待 AI 响应..."
            :resizable="true"
            class="result-textarea"
            style="flex: 1; min-height: 60px;"
          />
          <div v-else class="result-text-display" @click="editingResult = true">
            {{ editResult }}
            <div class="result-edit-hint">点击编辑结果</div>
          </div>
        </n-gi>
      </n-grid>
    </div>

    <template #footer>
      <n-space justify="end">
        <template v-if="!showResult">
          <n-button quaternary @click="closeDialog">取消</n-button>
          <n-button type="primary" :loading="loading" :disabled="loading" @click="handleEdit">
            {{ mode === 'polish' ? '开始润色' : '开始扩写' }}
          </n-button>
        </template>
        <template v-else-if="loading">
          <n-button quaternary @click="cancelEdit">取消生成</n-button>
        </template>
        <template v-else>
          <n-button quaternary @click="closeDialog">
            <template #icon><n-icon><CloseIcon/></n-icon></template>关闭
          </n-button>
          <n-button quaternary @click="copyResult">
            <template #icon><n-icon><CopyIcon/></n-icon></template>复制
          </n-button>
          <n-button type="primary" @click="replaceContent">
            <template #icon><n-icon><ReplaceIcon/></n-icon></template>替换原文
          </n-button>
        </template>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
/* 确保卡片的 content 区是 flex 容器，使子元素 flex:1 生效 */
:deep(.n-card > div:nth-child(2)) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

/* 结果预览：保留换行，清晰区分段落 */
.result-text-display {
  flex: 1;
  min-height: 60px;
  overflow-y: auto;
  white-space: pre-wrap;
  padding: 12px 14px;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  line-height: 1.9;
  font-size: 14px;
  color: #333;
  cursor: text;
  background: #fff;
}
.result-text-display:hover {
  border-color: #c0c0c0;
}
.result-edit-hint {
  margin-top: 8px;
  font-size: 11px;
  color: #bbb;
  text-align: center;
  border-top: 1px dashed #eee;
  padding-top: 6px;
}
.result-text-display:hover .result-edit-hint {
  color: #888;
}
/* 结果区 textarea 保留换行显示 */
:deep(.result-textarea textarea) {
  white-space: pre-wrap !important;
}
</style>

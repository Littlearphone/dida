<script setup lang="ts">
import { ref, inject, onUnmounted } from 'vue'
import {
  NModal, NAlert, NForm, NFormItem, NInput, NButton, NGrid, NGi,
  NSpace, NIcon, NProgress, useMessage,
} from 'naive-ui'
import { CopyOutline as CopyIcon, SyncOutline as ReplaceIcon, CloseOutline as CloseIcon } from '@vicons/ionicons5'
import { useNovelStore } from '../../stores/novel'
import * as aiApi from '../../api/ai'
import { EDITOR_ACTIONS_KEY } from '../../types/editor'

const props = defineProps<{
  show: boolean
  mode: 'polish' | 'expand'
  scope: 'selection' | 'chapter'
}>()
const emit = defineEmits<{ 'update:show': [value: boolean] }>()

const novelStore = useNovelStore()
const message = useMessage()
const editorActions = inject(EDITOR_ACTIONS_KEY)!

const requirement = ref('')
const loading = ref(false)
const originalContent = ref('')
const editResult = ref('')
const showResult = ref(false)
const progress = ref(0)
const progressText = ref('')

let abortController: AbortController | null = null
let progressTimer: ReturnType<typeof setInterval> | null = null

const title = props.mode === 'polish' ? 'AI 润色' : 'AI 扩写'
const resultLabel = props.mode === 'polish' ? '润色后' : '扩写后'

function startProgressSimulation() {
  progress.value = 0
  progressText.value = `正在准备${props.mode === 'polish' ? '润色' : '扩写'}请求...`
  progressTimer = setInterval(() => {
    const remaining = 90 - progress.value
    if (remaining > 0) {
      progress.value += Math.max(0.5, remaining * 0.08)
    }
    if (progress.value < 20) {
      progressText.value = `正在准备${props.mode === 'polish' ? '润色' : '扩写'}请求...`
    } else if (progress.value < 50) {
      progressText.value = '正在请求 AI 服务...'
    } else {
      progressText.value = `AI 正在${props.mode === 'polish' ? '润色' : '扩写'}...`
    }
  }, 200)
}
function stopProgressSimulation() {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
}

async function handleEdit() {
  loading.value = true
  originalContent.value = props.scope === 'selection'
    ? (window.getSelection()?.toString() || '')
    : (editorActions.getContent() || novelStore.currentChapter?.content || '')

  editResult.value = ''
  showResult.value = true

  abortController = new AbortController()
  startProgressSimulation()

  try {
    const apiFn = props.mode === 'polish' ? aiApi.polish : aiApi.expand
    const fullText = await apiFn(
      {
        content: originalContent.value,
        isSelection: props.scope === 'selection',
        outline: novelStore.currentNovel?.outline || '',
        requirement: requirement.value,
      },
      (fullText: string) => {
        editResult.value = fullText
        if (progress.value < 95) {
          progress.value = Math.max(progress.value, 70)
          progressText.value = `AI 正在${props.mode === 'polish' ? '润色' : '扩写'}...`
        }
      },
      abortController.signal,
    )
    editResult.value = fullText
    progress.value = 100
    progressText.value = `${props.mode === 'polish' ? '润色' : '扩写'}完成`
    stopProgressSimulation()
  } catch (e: any) {
    stopProgressSimulation()
    if (e.name === 'AbortError') {
      message.info('已取消')
      closeDialog()
      return
    }
    message.error(`${props.mode === 'polish' ? '润色' : '扩写'}失败: ${e.message}`)
    if (!editResult.value) {
      showResult.value = false
    } else {
      progress.value = 100
      progressText.value = `${props.mode === 'polish' ? '润色' : '扩写'}出错（已保留部分内容）`
    }
  } finally {
    loading.value = false
    abortController = null
  }
}

function cancelEdit() {
  if (abortController) {
    abortController.abort()
  }
}

function replaceContent() {
  if (!editResult.value) return
  if (props.scope === 'selection') {
    const sel = window.getSelection()
    if (sel?.rangeCount) {
      sel.getRangeAt(0).deleteContents()
      sel.getRangeAt(0).insertNode(document.createTextNode(editResult.value))
    }
  } else {
    editorActions.setContent(editResult.value)
  }
  message.success('已替换原文')
  closeDialog()
}

function copyResult() {
  if (editResult.value) {
    navigator.clipboard.writeText(editResult.value)
    message.success('已复制')
  }
}

function closeDialog() {
  if (loading.value) cancelEdit()
  stopProgressSimulation()
  abortController = null
  emit('update:show', false)
  requirement.value = ''
  editResult.value = ''
  originalContent.value = ''
  showResult.value = false
  progress.value = 0
  progressText.value = ''
}

onUnmounted(() => {
  stopProgressSimulation()
  if (abortController) abortController.abort()
})
</script>

<template>
  <n-modal class="dialog-modal" :show="show" :title="title" preset="card" style="width: 80vw; height: 80vh;"
    :mask-closable="false" draggable @update:show="emit('update:show', $event)">
    <!-- 提示：选中内容模式 -->
    <div v-if="scope === 'selection' && !showResult" style="margin-bottom: 12px">
      <n-alert type="info" :bordered="false">当前选中内容将被{{ mode === 'polish' ? '润色' : '扩写' }}</n-alert>
    </div>

    <!-- 输入阶段 -->
    <n-form v-if="!showResult" label-placement="top"
      style="display: flex; flex-direction: column; flex: 1; min-height: 0;">
      <n-form-item :label="`${mode === 'polish' ? '润色' : '扩写'}要求（可选）`" style="flex: 1; align-items: flex-start;">
        <n-input v-model:value="requirement" type="textarea"
          :placeholder="`输入${mode === 'polish' ? '润色' : '扩写'}方向、风格要求...`" :resizable="false" :maxlength="500" show-count
          style="height: 100%; min-height: 80px;" />
      </n-form-item>
    </n-form>

    <!-- 结果阶段：flex 自适应高度 -->
    <div v-if="showResult" style="display: flex; flex-direction: column; flex: 1; min-height: 0;">
      <!-- 进度条 -->
      <div v-if="loading" style="flex-shrink: 0; margin-bottom: 12px;">
        <n-progress type="line" :percentage="Math.round(progress)" :indicator-placement="'inside'"
          :height="20" :border-radius="4" processing>
          {{ progressText }}
        </n-progress>
      </div>

      <n-grid :cols="2" :x-gap="12" style="flex: 1; min-height: 0;">
        <n-gi style="display: flex; flex-direction: column; min-height: 0;">
          <n-alert type="info" :bordered="false" style="flex-shrink: 0; margin-bottom: 8px;">原文</n-alert>
          <!-- 原文预览：flex:1 自适应 -->
          <div style="flex: 1; min-height: 0; overflow-y: auto; border: 1px solid #e0e0e0; border-radius: 6px; background: #fafafa;">
            <div style="padding: 16px; white-space: pre-wrap; line-height: 1.8; font-size: 14px; color: #666;">
              {{ originalContent }}
            </div>
          </div>
        </n-gi>
        <n-gi style="display: flex; flex-direction: column; min-height: 0;">
          <n-alert type="success" :bordered="false" style="flex-shrink: 0; margin-bottom: 8px;">{{ resultLabel }}</n-alert>
          <!-- 结果预览（可编辑）：flex:1 自适应 -->
          <div style="flex: 1; min-height: 0; overflow-y: auto; border: 1px solid #2080f0; border-radius: 6px; background: #fafafa;">
            <div
              contenteditable="true"
              @input="editResult = ($event.target as HTMLElement).innerText"
              style="padding: 16px; white-space: pre-wrap; line-height: 1.8; font-size: 14px; outline: none; font-family: inherit; color: #333; min-height: 80px;">
              {{ editResult || (loading ? '等待 AI 响应...' : '') }}
            </div>
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

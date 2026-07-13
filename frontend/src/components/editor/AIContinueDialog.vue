<script setup lang="ts">
import { ref, inject, watch } from 'vue'
import { NModal, NAlert, NForm, NFormItem, NInput, NButton, NSpace, NIcon, NProgress, useMessage } from 'naive-ui'
import {
  CopyOutline as CopyIcon, ArrowDownOutline as InsertIcon,
  AddCircleOutline as NewChapterIcon,
  ColorWandOutline as PolishIcon, CreateOutline as ExpandIcon,
  RefreshOutline as RefreshIcon,
} from '@vicons/ionicons5'
import { useNovelStore } from '../../stores/novel'
import { useAIStream } from '../../composables/useAIStream'
import { htmlToPlainText, normalizeParagraphs } from '../../utils/editor'
import * as aiApi from '../../api/ai'
import { EDITOR_ACTIONS_KEY } from '../../types/editor'

const props = defineProps<{
  show: boolean
  refinedContent?: string
}>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  /** 对续写结果进行二次处理（润色/扩写） */
  'polishResult': [text: string]
  'expandResult': [text: string]
}>()

const novelStore = useNovelStore()
const message = useMessage()
const editorActions = inject(EDITOR_ACTIONS_KEY)!

const continueRequirement = ref('')
const continueResult = ref('')
const showContinueResult = ref(false)

const {
  loading: continueLoading, progress: generateProgress, progressText: generateStatus,
  MAX_RETRIES,
  getAbortSignal: getSignal, cleanupRequest, resetRetry,
  shouldRetry, isCanceled, incrementRetry,
  startProgressSimulation: startProgress, stopProgressSimulation: stopProgress,
  completeProgress, errorProgress,
} = useAIStream()

/** 归一化段落换行：当 AI 返回的内容没有双换行但存在单换行时，将单换行视为段落分隔 */
async function handleContinueWrite() {
  if (!novelStore.currentChapter) return
  resetRetry()
  showContinueResult.value = true
  continueLoading.value = true

  while (shouldRetry()) {
    if (isCanceled()) { closeDialog(); return }

    continueResult.value = ''
    generateProgress.value = 0
    startProgress('续写')
    const signal = getSignal()

    try {
      const chapters = novelStore.chapters
      const currentCh = novelStore.currentChapter; const curIdx = currentCh
        ? chapters.findIndex(c => c.id === currentCh.id)
        : -1
      const prevCh = curIdx > 0 ? chapters[curIdx - 1] : null

      const fullText = await aiApi.continueWrite(
        {
          chapterContent: htmlToPlainText(novelStore.currentChapter.content),
          previousChapterContent: htmlToPlainText(prevCh?.content || ''),
          outline: novelStore.currentNovel?.outline || '',
          requirement: continueRequirement.value,
          characters: novelStore.currentNovel?.characters,
          relationships: novelStore.currentNovel?.relationships,
          events: novelStore.currentNovel?.events,
        },
        (text: string) => {
          continueResult.value = normalizeParagraphs(text)
          if (generateProgress.value < 95) {
            generateProgress.value = Math.max(generateProgress.value, 70)
          }
        },
        signal,
      )
      stopProgress()

      if (fullText) {
        continueResult.value = normalizeParagraphs(fullText)
        completeProgress('续写')
        continueLoading.value = false
        return
      }

      incrementRetry()
    } catch (e: any) {
      stopProgress()
      if (e.name === 'AbortError') { closeDialog(); return }
      if (shouldRetry() && !isCanceled()) { incrementRetry(); continue }
      message.error(`续写失败: ${e.message}`)
      if (!continueResult.value) { showContinueResult.value = false }
      else { errorProgress('续写') }
      continueLoading.value = false
      return
    }
  }

  message.warning('AI 连续返回空内容，续写失败')
  showContinueResult.value = false
  continueLoading.value = false
}

function cancelContinue() {
  cleanupRequest()
}

function insertToChapterEnd() {
  if (!novelStore.currentChapter) return
  // 使用 appendContent 将续写文本转段落后追加到文档末尾，避免 HTML+纯文本拼接丢失换行分段
  editorActions.appendContent?.(continueResult.value)
  closeDialog()
  // 内容插入后自动提取元数据（大纲/角色/关系/事件）
  editorActions.triggerExtract?.()
  message.success('已插入到章节末尾')
}

function insertAsNewChapter() {
  if (!novelStore.currentNovel) return
  novelStore.createChapter({
    novelId: novelStore.currentNovel.id,
    title: `${novelStore.currentChapter?.title || '续写'} (续)`,
    content: continueResult.value,
    order: novelStore.chapters.length + 1,
  }).then(ch => {
    if (ch) {
      novelStore.selectChapter(ch)
      // 新建章节后自动提取元数据
      editorActions.triggerExtract?.()
      message.success('已创建新章节')
    }
  })
  closeDialog()
}

function copyResult() {
  navigator.clipboard.writeText(continueResult.value)
  message.success('已复制')
}

function closeDialog() {
  if (continueLoading.value) cancelContinue()
  stopProgress()
  emit('update:show', false)
  continueRequirement.value = ''
  continueResult.value = ''
  showContinueResult.value = false
  generateProgress.value = 0
  generateStatus.value = ''
}

/** 接收外部传入的精炼结果（润色/扩写后），替换续写框结果 */
watch(() => props.refinedContent, (val: string | undefined) => {
  if (val) {
    continueResult.value = val
    showContinueResult.value = true
  }
})

function retryContinue() {
  showContinueResult.value = false
  continueResult.value = ''
  generateProgress.value = 0
  generateStatus.value = ''
}
</script>

<template>
  <n-modal class="dialog-modal" :show="show" title="AI 续写" preset="card" style="width: 80vw; height: 80vh;"
    :mask-closable="false" draggable @update:show="emit('update:show', $event)">
    <!-- 输入阶段：填写续写要求 -->
    <div v-if="!showContinueResult" style="display: flex; flex-direction: column; flex: 1; min-height: 0;">
      <n-form label-placement="top" style="display: flex; flex-direction: column; flex: 1; min-height: 0;">
        <n-form-item label="续写要求（可选）" style="flex: 1; align-items: flex-start;">
          <n-input v-model:value="continueRequirement" type="textarea"
            placeholder="输入对续写内容的要求、方向或风格..." :resizable="false" :maxlength="500" show-count
            style="height: 100%; min-height: 80px;" />
        </n-form-item>
      </n-form>
    </div>
    <!-- 结果阶段 -->
    <div v-else style="display: flex; flex-direction: column; flex: 1; min-height: 0;">
      <n-alert type="success" :bordered="false" style="flex-shrink: 0; margin-bottom: 12px">
        续写完成，共 {{ continueResult.length }} 字
      </n-alert>
      <!-- 生成进度条 -->
      <div v-if="continueLoading" style="flex-shrink: 0; margin-bottom: 12px;">
        <n-progress type="line" :percentage="Math.round(generateProgress)" :indicator-placement="'inside'"
          :height="20" :border-radius="4" processing>
          {{ generateStatus }}
        </n-progress>
      </div>
      <!-- 预览区域：可编辑文本区域，加载时禁用 -->
      <n-input
        v-model:value="continueResult"
        type="textarea"
        :disabled="continueLoading"
        placeholder="等待 AI 响应..."
        :resizable="false"
        style="flex: 1; min-height: 60px;"
      />
    </div>
    <template #footer>
      <n-space justify="end">
        <template v-if="!showContinueResult">
          <n-button quaternary @click="closeDialog">取消</n-button>
          <n-button type="primary" :loading="continueLoading" :disabled="continueLoading" @click="handleContinueWrite">开始续写</n-button>
        </template>
        <template v-else-if="continueLoading">
          <n-button quaternary @click="cancelContinue">取消生成</n-button>
        </template>
        <template v-else>
          <n-button quaternary @click="retryContinue">
            <template #icon><n-icon><RefreshIcon/></n-icon></template>重新续写
          </n-button>
          <n-button quaternary @click="copyResult">
            <template #icon><n-icon><CopyIcon/></n-icon></template>复制
          </n-button>
          <!-- 二次处理：将续写结果送入润色或扩写 -->
          <n-button secondary @click="emit('polishResult', continueResult)" :disabled="!continueResult">
            <template #icon><n-icon><PolishIcon/></n-icon></template>润色
          </n-button>
          <n-button secondary @click="emit('expandResult', continueResult)" :disabled="!continueResult">
            <template #icon><n-icon><ExpandIcon/></n-icon></template>扩写
          </n-button>
          <n-button quaternary @click="insertToChapterEnd">
            <template #icon><n-icon><InsertIcon/></n-icon></template>插入当前章节末尾
          </n-button>
          <n-button type="primary" @click="insertAsNewChapter">
            <template #icon><n-icon><NewChapterIcon/></n-icon></template>新建章节
          </n-button>
        </template>
      </n-space>
    </template>
  </n-modal>
</template>

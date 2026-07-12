<script setup lang="ts">
import { ref, inject, onUnmounted } from 'vue'
import { NModal, NAlert, NForm, NFormItem, NInput, NButton, NSpace, NIcon, NProgress, useMessage } from 'naive-ui'
import {
  CopyOutline as CopyIcon, ArrowDownOutline as InsertIcon,
  AddCircleOutline as NewChapterIcon,
  ColorWandOutline as PolishIcon, CreateOutline as ExpandIcon,
} from '@vicons/ionicons5'
import { useNovelStore } from '../../stores/novel'
import * as aiApi from '../../api/ai'
import { EDITOR_ACTIONS_KEY } from '../../types/editor'

defineProps<{ show: boolean }>()
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
const continueLoading = ref(false)
const continueResult = ref('')
const showContinueResult = ref(false)
const generateProgress = ref(0) // 模拟进度 0-100
const generateStatus = ref('') // 当前状态文字
// 用于取消流式请求的 AbortController
let abortController: AbortController | null = null
/** 自动重试：AI 返回空内容时自动重试 */
const MAX_RETRIES = 2
let retries = 0
let cancelRequested = false

/** 模拟进度的定时器，让进度条平滑递增到 90% */
let progressTimer: ReturnType<typeof setInterval> | null = null
function startProgressSimulation() {
  generateProgress.value = 0
  generateStatus.value = '正在准备续写请求...'
  progressTimer = setInterval(() => {
    // 进度越接近 90% 增长越慢（模拟真实等待体验）
    const remaining = 90 - generateProgress.value
    if (remaining > 0) {
      generateProgress.value += Math.max(0.5, remaining * 0.08)
    }
    // 根据进度更新状态文字
    if (generateProgress.value < 20) {
      generateStatus.value = '正在准备续写请求...'
    } else if (generateProgress.value < 50) {
      generateStatus.value = '正在请求 AI 服务...'
    } else {
      generateStatus.value = 'AI 正在生成内容...'
    }
  }, 200)
}
function stopProgressSimulation() {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
}

async function handleContinueWrite() {
  if (!novelStore.currentChapter) return
  continueLoading.value = true
  retries = 0
  cancelRequested = false
  showContinueResult.value = true

  while (retries <= MAX_RETRIES) {
    if (cancelRequested) {
      closeDialog()
      return
    }

    // 每次尝试清除上次残留内容
    continueResult.value = ''
    generateProgress.value = 0

    if (retries > 0) {
      generateStatus.value = `续写返回为空，自动重试 (${retries}/${MAX_RETRIES})...`
    } else {
      generateStatus.value = '正在准备续写请求...'
    }

    abortController = new AbortController()
    startProgressSimulation()

    try {
      // 流式续写：onChunk 回调实时更新内容和进度
      const fullText = await aiApi.continueWrite(
        {
          chapterContent: novelStore.currentChapter.content,
          outline: novelStore.currentNovel?.outline || '',
          requirement: continueRequirement.value,
        },
        (fullText: string) => {
          continueResult.value = fullText
          if (generateProgress.value < 95) {
            generateProgress.value = Math.max(generateProgress.value, 70)
            generateStatus.value = 'AI 正在生成内容...'
          }
        },
        abortController.signal,
      )
      stopProgressSimulation()

      if (fullText) {
        // 成功获取内容
        continueResult.value = fullText
        generateProgress.value = 100
        generateStatus.value = '续写完成'
        continueLoading.value = false
        abortController = null
        return
      }

      // 返回空内容 → 重试
      retries++

    } catch (e: any) {
      stopProgressSimulation()
      if (e.name === 'AbortError') {
        // 取消时不弹提示
        closeDialog()
        return
      }

      // 请求出错，还有重试次数则继续
      if (retries < MAX_RETRIES && !cancelRequested) {
        retries++
        generateStatus.value = `续写出错，自动重试 (${retries}/${MAX_RETRIES})...`
        continue
      }

      // 重试用完，报错（如有部分内容则保留）
      message.error(`续写失败: ${e.message}`)
      if (!continueResult.value) {
        showContinueResult.value = false
      } else {
        generateProgress.value = 100
        generateStatus.value = '续写出错（已保留部分内容）'
      }
      continueLoading.value = false
      abortController = null
      return
    }
  }

  // 自动重试用完仍为空
  message.warning('AI 连续返回空内容，续写失败')
  showContinueResult.value = false
  continueLoading.value = false
  abortController = null
}

function cancelContinue() {
  cancelRequested = true
  if (abortController) {
    abortController.abort()
  }
}

function insertToChapterEnd() {
  if (!novelStore.currentChapter) return
  // 使用 appendContent 将续写文本转段落后追加到文档末尾，避免 HTML+纯文本拼接丢失换行分段
  editorActions.appendContent?.(continueResult.value)
  closeDialog()
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
    if (ch) { novelStore.selectChapter(ch); message.success('已创建新章节') }
  })
  closeDialog()
}

function copyResult() {
  navigator.clipboard.writeText(continueResult.value)
  message.success('已复制')
}

function closeDialog() {
  // 如果正在加载，取消请求
  if (continueLoading.value) {
    cancelContinue()
  }
  stopProgressSimulation()
  abortController = null
  emit('update:show', false)
  continueRequirement.value = ''
  continueResult.value = ''
  showContinueResult.value = false
  generateProgress.value = 0
  generateStatus.value = ''
}

onUnmounted(() => {
  stopProgressSimulation()
  if (abortController) abortController.abort()
})
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
      <!-- 预览区域：flex:1 自适应填满剩余空间，保留换行 -->
      <div style="flex: 1; min-height: 0; overflow-y: auto; border: 1px solid #e0e0e0; border-radius: 6px; background: #fafafa;">
        <div style="padding: 20px; white-space: pre-wrap; line-height: 1.8; font-size: 15px; color: #333;">
          {{ continueResult || (continueLoading ? '等待 AI 响应...' : '') }}
        </div>
      </div>
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

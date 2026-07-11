<script setup lang="ts">
import { ref, inject } from 'vue'
import {
  NModal, NAlert, NForm, NFormItem, NInput, NButton, NGrid, NGi,
  NScrollbar, NSpace, NIcon, NText, useMessage,
} from 'naive-ui'
import { CopyOutline as CopyIcon } from '@vicons/ionicons5'
import { useNovelStore } from '../../stores/novel'
import * as aiApi from '../../api/ai'
import { EDITOR_ACTIONS_KEY, type EditorActions } from '../../types/editor'

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
const result = ref<{ original: string; result: string } | null>(null)

const title = props.mode === 'polish' ? 'AI 润色' : 'AI 扩写'
const resultLabel = props.mode === 'polish' ? '润色后' : '扩写后'

async function handleEdit() {
  loading.value = true
  const content = props.scope === 'selection'
    ? (window.getSelection()?.toString() || '')
    : (editorActions.getContent() || novelStore.currentChapter?.content || '')
  try {
    const apiFn = props.mode === 'polish' ? aiApi.polish : aiApi.expand
    const res = await apiFn({
      content,
      isSelection: props.scope === 'selection',
      outline: novelStore.currentNovel?.outline || '',
      requirement: requirement.value,
    })
    result.value = res
  } catch (e: any) {
    message.error(`${props.mode === 'polish' ? '润色' : '扩写'}失败: ${e.message}`)
  } finally {
    loading.value = false
  }
}

function replaceContent() {
  if (!result.value) return
  if (props.scope === 'selection') {
    const sel = window.getSelection()
    if (sel?.rangeCount) {
      sel.getRangeAt(0).deleteContents()
      sel.getRangeAt(0).insertNode(document.createTextNode(result.value.result))
    }
  } else {
    editorActions.setContent(result.value.result)
  }
  message.success('已替换原文')
  closeDialog()
}

function copyResult() {
  if (result.value) {
    navigator.clipboard.writeText(result.value.result)
    message.success('已复制')
  }
}

function closeDialog() {
  emit('update:show', false)
  requirement.value = ''
  result.value = null
}
</script>

<template>
  <n-modal class="dialog-modal" :show="show" :title="title" preset="card" style="width: 800px"
    :mask-closable="false" @update:show="emit('update:show', $event)">
    <div v-if="scope === 'selection' && !result" style="margin-bottom: 12px">
      <n-alert type="info" :bordered="false">当前选中内容将被{{ mode === 'polish' ? '润色' : '扩写' }}</n-alert>
    </div>
    <n-form v-if="!result" label-placement="top">
      <n-form-item :label="`${mode === 'polish' ? '润色' : '扩写'}要求（可选）`">
        <n-input v-model:value="requirement" type="textarea"
          :placeholder="`输入${mode === 'polish' ? '润色' : '扩写'}方向、风格要求...`" :rows="3" />
      </n-form-item>
    </n-form>
    <div v-if="result">
      <n-grid :cols="2" :x-gap="12">
        <n-gi>
          <n-text strong depth="2">原文</n-text>
          <div style="max-height: 300px; border: 1px solid #eee; border-radius: 4px; padding: 12px; margin-top: 8px; overflow-y: auto;">
            <n-text>{{ result.original }}</n-text>
          </div>
        </n-gi>
        <n-gi>
          <n-text strong depth="2">{{ resultLabel }}</n-text>
          <div style="max-height: 300px; border: 1px solid #2080f0; border-radius: 4px; padding: 12px; margin-top: 8px; overflow-y: auto;">
            <pre contenteditable="true"
              @input="result.result = ($event.target as HTMLElement).innerText"
              style="outline: none; white-space: pre-wrap; font-family: inherit; margin: 0;">{{ result.result }}</pre>
          </div>
        </n-gi>
      </n-grid>
    </div>
    <template #footer>
      <n-space justify="end">
        <template v-if="!result">
          <n-button quaternary @click="closeDialog">取消</n-button>
          <n-button type="primary" :loading="loading" @click="handleEdit">
            {{ mode === 'polish' ? '开始润色' : '开始扩写' }}
          </n-button>
        </template>
        <template v-else>
          <n-button quaternary @click="closeDialog">关闭</n-button>
          <n-button quaternary @click="copyResult">
            <template #icon><n-icon><CopyIcon/></n-icon></template>复制
          </n-button>
          <n-button type="primary" @click="replaceContent">替换原文</n-button>
        </template>
      </n-space>
    </template>
  </n-modal>
</template>

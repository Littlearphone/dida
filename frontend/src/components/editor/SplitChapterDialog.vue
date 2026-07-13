<script setup lang="ts">
/**
 * 拆分章节确认弹框 — 输入新章节标题后确认拆分
 */
import { ref, watch } from 'vue'
import { NButton, NForm, NFormItem, NInput, NModal, NSpace } from 'naive-ui'

const props = defineProps<{
  show: boolean
  loading: boolean
  defaultTitle: string
}>()

const emit = defineEmits<{
  'update:show': [v: boolean]
  confirm: [title: string]
  cancel: []
}>()

const chapterTitle = ref(props.defaultTitle || '')

// dialog 打开时同步默认标题
watch(() => props.show, (v) => {
  if (v) chapterTitle.value = props.defaultTitle || ''
})

function handleConfirm() {
  const title = chapterTitle.value.trim()
  if (!title) return
  emit('confirm', title)
}

function handleCancel() {
  chapterTitle.value = ''
  emit('cancel')
  emit('update:show', false)
}
</script>

<template>
  <n-modal class="dialog-modal" :show="show" title="拆分为新章节" preset="card"
    style="width: 380px" :mask-closable="false"
    @update:show="(v: boolean) => emit('update:show', v)">
    <n-form label-placement="top">
      <n-form-item label="新章节标题" required>
        <n-input v-model:value="chapterTitle" placeholder="输入章节标题"
          @keyup.enter="handleConfirm" />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button quaternary @click="handleCancel">取消</n-button>
        <n-button type="primary" :loading="loading" @click="handleConfirm">拆分</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

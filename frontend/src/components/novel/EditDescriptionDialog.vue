<script setup lang="ts">
import { ref, watch } from 'vue'
import { NModal, NForm, NFormItem, NInput, NButton, NSpace, useMessage } from 'naive-ui'
import { useNovelStore } from '../../stores/novel'
import type { Novel } from '../../types'

const props = defineProps<{ show: boolean; novel: Novel | null }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  saved: []
}>()

const novelStore = useNovelStore()
const message = useMessage()

const editDescContent = ref('')

// 弹框打开时用当前小说简介初始化输入框
watch(() => props.show, (open) => {
  if (open && props.novel) editDescContent.value = props.novel.description || ''
})

async function handleSave() {
  const novel = props.novel
  if (!novel) return
  const ok = await novelStore.updateNovelMeta(novel.id, { description: editDescContent.value.trim() })
  if (ok) {
    novel.description = editDescContent.value.trim()
    message.success('简介已更新')
    emit('update:show', false)
    emit('saved')
  } else {
    message.error('更新失败')
  }
}
</script>

<template>
  <n-modal class="dialog-modal" :show="show" title="修改简介" preset="card" style="width: 480px" :mask-closable="false"
    @update:show="emit('update:show', $event)">
    <n-form label-placement="top">
      <n-form-item label="小说简介">
        <n-input v-model:value="editDescContent" type="textarea" placeholder="输入小说简介" :rows="4" />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button quaternary @click="emit('update:show', false)">取消</n-button>
        <n-button type="primary" @click="handleSave">保存</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

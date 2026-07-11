<script setup lang="ts">
import { ref, watch } from 'vue'
import { NModal, NForm, NFormItem, NInput, NButton, NSpace, useMessage } from 'naive-ui'
import { useNovelStore } from '../../stores/novel'
import type { Novel } from '../../types'

const props = defineProps<{ show: boolean; novel: Novel | null }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  renamed: []
}>()

const novelStore = useNovelStore()
const message = useMessage()

const renameNovelTitle = ref('')

// 弹框打开时用当前小说标题初始化输入框
watch(() => props.show, (open) => {
  if (open && props.novel) renameNovelTitle.value = props.novel.title
})

async function handleSave() {
  const novel = props.novel
  if (!novel || !renameNovelTitle.value.trim()) { message.warning('请输入小说标题'); return }
  const ok = await novelStore.updateNovelMeta(novel.id, { title: renameNovelTitle.value.trim() })
  if (ok) {
    novel.title = renameNovelTitle.value.trim()
    message.success('已重命名')
    emit('update:show', false)
    emit('renamed')
  } else {
    message.error('重命名失败')
  }
}
</script>

<template>
  <n-modal class="dialog-modal" :show="show" title="重命名小说" preset="card" style="width: 360px" :mask-closable="false"
    @update:show="emit('update:show', $event)">
    <n-form label-placement="top">
      <n-form-item label="小说标题" required>
        <n-input v-model:value="renameNovelTitle" placeholder="输入新标题" @keyup.enter="handleSave" />
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

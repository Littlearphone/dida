<script setup lang="ts">
import { ref } from 'vue'
import { NModal, NForm, NFormItem, NInput, NButton, NSpace, useMessage } from 'naive-ui'
import { useNovelStore } from '../../stores/novel'
import type { Novel } from '../../types'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  created: [novel: Novel]
}>()

const novelStore = useNovelStore()
const message = useMessage()

const newTitle = ref('')
const newAuthor = ref('')
const creating = ref(false)

async function handleCreate() {
  if (!newTitle.value.trim()) { message.warning('请输入小说标题'); return }
  creating.value = true
  const novel = await novelStore.createNovel(newTitle.value.trim(), newAuthor.value.trim())
  creating.value = false
  if (novel) {
    message.success('创建成功')
    newTitle.value = ''
    newAuthor.value = ''
    emit('update:show', false)
    emit('created', novel)
  }
}
</script>

<template>
  <n-modal class="dialog-modal" :show="show" title="创建新小说" preset="card" style="width: 420px" :mask-closable="false"
    @update:show="emit('update:show', $event)"
    @after-leave="() => { newTitle = ''; newAuthor = '' }">
    <n-form label-placement="top">
      <n-form-item label="小说标题" required>
        <n-input v-model:value="newTitle" placeholder="输入小说标题" />
      </n-form-item>
      <n-form-item label="作者（可选）">
        <n-input v-model:value="newAuthor" placeholder="输入作者名" />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button quaternary @click="emit('update:show', false)">取消</n-button>
        <n-button type="primary" :loading="creating" @click="handleCreate">创建</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { NModal, NButton, NSpace, NText, useMessage } from 'naive-ui'
import { useNovelStore } from '../../stores/novel'
import type { Novel } from '../../types'

const props = defineProps<{ show: boolean; novel: Novel | null }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  deleted: []
}>()

const novelStore = useNovelStore()
const message = useMessage()

const deletingNovel = ref(false)

async function handleDelete() {
  const novel = props.novel
  if (!novel) return
  deletingNovel.value = true
  const ok = await novelStore.deleteNovel(novel.id)
  deletingNovel.value = false
  if (ok) message.success('已删除')
  else message.error('删除失败')
  emit('update:show', false)
  emit('deleted')
}
</script>

<template>
  <n-modal class="dialog-modal" :show="show" title="删除小说" preset="card" style="width: 360px" :mask-closable="false" draggable
    @update:show="emit('update:show', $event)">
    <n-text>确定删除《{{ novel?.title }}》吗？此操作将删除所有章节且不可撤销。</n-text>
    <template #footer>
      <n-space justify="end">
        <n-button quaternary @click="emit('update:show', false)">取消</n-button>
        <n-button type="error" :loading="deletingNovel" @click="handleDelete">删除</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

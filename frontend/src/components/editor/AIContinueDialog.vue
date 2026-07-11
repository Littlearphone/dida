<script setup lang="ts">
import { ref, inject } from 'vue'
import { NModal, NAlert, NForm, NFormItem, NInput, NButton, NScrollbar, NSpace, NIcon, NText, useMessage } from 'naive-ui'
import { CopyOutline as CopyIcon } from '@vicons/ionicons5'
import { useNovelStore } from '../../stores/novel'
import * as aiApi from '../../api/ai'
import { EDITOR_ACTIONS_KEY, type EditorActions } from '../../types/editor'

defineProps<{ show: boolean }>()
const emit = defineEmits<{ 'update:show': [value: boolean] }>()

const novelStore = useNovelStore()
const message = useMessage()
const editorActions = inject(EDITOR_ACTIONS_KEY)!

const continueRequirement = ref('')
const continueLoading = ref(false)
const continueResult = ref('')
const showContinueResult = ref(false)

async function handleContinueWrite() {
  if (!novelStore.currentChapter) return
  continueLoading.value = true
  try {
    const res = await aiApi.continueWrite({
      chapterContent: novelStore.currentChapter.content,
      outline: novelStore.currentNovel?.outline || '',
      requirement: continueRequirement.value,
    })
    continueResult.value = res.result
    showContinueResult.value = true
  } catch (e: any) {
    message.error(`续写失败: ${e.message}`)
  } finally {
    continueLoading.value = false
  }
}

function insertToChapterEnd() {
  if (!novelStore.currentChapter) return
  const current = editorActions.getContent?.() || novelStore.currentChapter.content
  editorActions.setContent(current + '\n\n' + continueResult.value)
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
  emit('update:show', false)
  continueRequirement.value = ''
  continueResult.value = ''
  showContinueResult.value = false
}
</script>

<template>
  <n-modal class="dialog-modal" :show="show" title="AI 续写" preset="card" style="width: 500px"
    :mask-closable="false" @update:show="emit('update:show', $event)">
    <div v-if="!showContinueResult">
      <n-form label-placement="top">
        <n-form-item label="续写要求（可选）">
          <n-input v-model:value="continueRequirement" type="textarea"
            placeholder="输入对续写内容的要求、方向或风格..." :rows="4" />
        </n-form-item>
      </n-form>
    </div>
    <div v-else>
      <n-alert type="success" :bordered="false" style="margin-bottom: 12px">
        续写完成，共 {{ continueResult.length }} 字
      </n-alert>
      <n-scrollbar style="max-height: 300px; border: 1px solid #eee; border-radius: 4px; padding: 12px;">
        <n-text>{{ continueResult }}</n-text>
      </n-scrollbar>
    </div>
    <template #footer>
      <n-space justify="end">
        <template v-if="!showContinueResult">
          <n-button quaternary @click="closeDialog">取消</n-button>
          <n-button type="primary" :loading="continueLoading" @click="handleContinueWrite">开始续写</n-button>
        </template>
        <template v-else>
          <n-button quaternary @click="copyResult">
            <template #icon><n-icon><CopyIcon/></n-icon></template>复制
          </n-button>
          <n-button quaternary @click="insertToChapterEnd">插入当前章节末尾</n-button>
          <n-button type="primary" @click="insertAsNewChapter">新建章节</n-button>
        </template>
      </n-space>
    </template>
  </n-modal>
</template>

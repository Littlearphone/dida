<script setup lang="ts">
import { ref } from 'vue'
import {
  NModal, NForm, NFormItem, NInput, NButton, NSpace, NUpload, NAlert,
  NScrollbar, NCard, NText, useMessage,
} from 'naive-ui'
import { useNovelStore } from '../../stores/novel'
import { useSettingsStore } from '../../stores/settings'
import * as aiApi from '../../api/ai'
import type { Novel, SplitResult } from '../../types'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  imported: [novel: Novel]
}>()

const novelStore = useNovelStore()
const settingsStore = useSettingsStore()
const message = useMessage()

const importTitle = ref('')
const importContent = ref('')
const importing = ref(false)
const importStatus = ref('')
const showSplitPreview = ref(false)
const splitResult = ref<SplitResult | null>(null)

async function handleFileSelect(file: File) {
  const text = await file.text()
  importContent.value = text
  if (!importTitle.value) {
    importTitle.value = file.name.replace(/\.[^.]+$/, '')
  }
  return true
}

async function handleAISplit() {
  if (!importContent.value.trim()) { message.warning('请先选择或粘贴小说内容'); return }
  importing.value = true
  importStatus.value = '正在连接 AI 服务...'
  try {
    importStatus.value = 'AI 正在分析章节结构，请稍候...'
    const result = await aiApi.splitChapters(importContent.value)
    importStatus.value = `拆分完成，共 ${result.chapters.length} 章`
    splitResult.value = result
    if (result.title) importTitle.value = result.title
    showSplitPreview.value = true
  } catch (e: any) {
    message.warning(`AI拆分失败: ${e.message}，您可以手动导入`)
    splitResult.value = {
      chapters: [{ title: importTitle.value || '第一章', content: importContent.value }],
      characters: [], events: [], outline: '',
    }
    showSplitPreview.value = true
  } finally {
    importing.value = false
    importStatus.value = ''
  }
}

async function handleManualImport() {
  if (!importContent.value.trim()) { message.warning('请粘贴小说内容'); return }
  importing.value = true
  const novel = await novelStore.importNovel({
    title: importTitle.value || '未命名小说',
    skipAISplit: true,
    chapters: [{ title: '第一章', content: importContent.value }],
  })
  importing.value = false
  if (novel) {
    message.success('导入成功')
    resetImport()
    emit('update:show', false)
    emit('imported', novel)
  }
}

async function confirmImport() {
  if (!splitResult.value) return
  importing.value = true
  const novel = await novelStore.importNovel({
    title: importTitle.value || '未命名小说',
    skipAISplit: true,
    chapters: splitResult.value.chapters.map((ch, i) => ({
      title: ch.title || `第${i + 1}章`,
      content: ch.content,
    })),
  })
  importing.value = false
  if (novel) {
    await saveExtractedInfo(novel, splitResult.value)
    message.success('导入成功')
    showSplitPreview.value = false
    resetImport()
    emit('update:show', false)
    emit('imported', novel)
  }
}

async function saveExtractedInfo(novel: Novel, split: SplitResult) {
  novel.outline = split.outline
  novel.description = split.description
  novel.characters = split.characters
  novel.relationships = split.relationships
  novel.events = split.events
  try {
    const response = await fetch(`/api/novels/${novel.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        outline: split.outline,
        description: split.description,
        characters: split.characters,
        relationships: split.relationships,
        events: split.events,
      }),
    })
    if (!response.ok) console.warn('保存提取信息失败')
  } catch (e) { console.warn('保存提取信息失败:', e) }
}

function resetImport() {
  importTitle.value = ''
  importContent.value = ''
  splitResult.value = null
  showSplitPreview.value = false
}

const aiConfigured = () => settingsStore.settings?.aiConfigured ?? false
</script>

<template>
  <n-modal class="dialog-modal" :show="show" title="导入小说" preset="card" style="width: 600px" :mask-closable="false" draggable
    @update:show="emit('update:show', $event)" @after-leave="resetImport">
    <div v-if="!showSplitPreview">
      <n-form label-placement="top">
        <n-form-item label="小说标题">
          <n-input v-model:value="importTitle" placeholder="输入标题（可选，默认用文件名）" :disabled="importing" />
        </n-form-item>
        <n-form-item label="选择文件">
          <n-upload :max="1" accept=".txt,.md,.json" :disabled="importing"
            @change="(e: any) => e.file.file && handleFileSelect(e.file.file)">
            <n-button :disabled="importing">选择文件</n-button>
          </n-upload>
        </n-form-item>
        <n-form-item label="或粘贴小说内容">
          <n-input v-model:value="importContent" type="textarea" placeholder="粘贴小说全文..." :rows="8" :disabled="importing" />
        </n-form-item>
      </n-form>
      <n-alert v-if="!aiConfigured()" type="info" :bordered="false" style="margin-bottom: 12px">
        <template #header>未配置 AI 接口</template>
        未检测到 AI 配置，将无法智能拆分章节和提取大纲。您可以在设置中配置 AI 接口，或直接导入为单章节。
      </n-alert>
      <n-alert v-if="importing && importStatus" type="info" :bordered="false" style="margin-bottom: 12px">
        <template #header>AI 处理中</template>
        {{ importStatus }}
      </n-alert>
    </div>
    <div v-else>
      <n-alert type="success" :bordered="false" style="margin-bottom: 12px">
        已识别 {{ splitResult?.chapters.length || 0 }} 个章节
      </n-alert>
      <n-space v-if="splitResult?.author || splitResult?.description" vertical
        style="margin-bottom: 12px; padding: 8px 12px; background: #fafafa; border-radius: 6px;">
        <n-text v-if="splitResult?.author" depth="2" style="font-size: 13px;">作者：{{ splitResult.author }}</n-text>
        <n-text v-if="splitResult?.description" depth="3" style="font-size: 12px; line-height: 1.6;">{{ splitResult.description }}</n-text>
      </n-space>
      <n-scrollbar style="max-height: 300px">
        <n-card v-for="(ch, i) in splitResult?.chapters || []" :key="i"
          :title="ch.title || `第${i+1}章`" size="small" style="margin-bottom: 8px">
          <n-text depth="3">{{ ch.content.slice(0, 100) }}{{ ch.content.length > 100 ? '...' : '' }}</n-text>
        </n-card>
      </n-scrollbar>
      <n-space v-if="splitResult?.outline" style="margin-top: 12px">
        <n-text style="font-weight: 500;">大纲摘要：</n-text>
        <n-text>{{ splitResult.outline.slice(0, 200) }}</n-text>
      </n-space>
    </div>
    <template #footer>
      <n-space justify="end">
        <template v-if="!showSplitPreview">
          <n-button quaternary @click="emit('update:show', false); resetImport()" :disabled="importing">取消</n-button>
          <n-button quaternary :disabled="importing" :loading="importing" @click="handleManualImport">直接导入</n-button>
          <n-button v-if="aiConfigured()" :disabled="importing" type="primary" :loading="importing" @click="handleAISplit">AI智能拆分</n-button>
        </template>
        <template v-else>
          <n-button quaternary @click="showSplitPreview = false" :disabled="importing">返回修改</n-button>
          <n-button type="primary" :disabled="importing" :loading="importing" @click="confirmImport">确认导入</n-button>
        </template>
      </n-space>
    </template>
  </n-modal>
</template>

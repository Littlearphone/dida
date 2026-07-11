<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { useNovelStore } from '../stores/novel'
import { useSettingsStore } from '../stores/settings'
import NovelCard from '../components/NovelCard.vue'
import * as aiApi from '../api/ai'
import type { SplitResult, Novel } from '../types'
import {
  NButton, NCard, NModal, NInput, NForm, NFormItem, NIcon, NText,
  NEmpty, NSpace, NGrid, NGi, NDivider, NUpload, useMessage, NAlert,
  NSpin, NScrollbar, useDialog,
} from 'naive-ui'
import { AddOutline as AddIcon, DocumentTextOutline as ImportIcon } from '@vicons/ionicons5'

const router = useRouter()
const novelStore = useNovelStore()
const settingsStore = useSettingsStore()
const message = useMessage()
const dialog = useDialog()

// === 创建小说弹框 ===
const showCreateModal = ref(false)
const newTitle = ref('')
const newAuthor = ref('')
const creating = ref(false)

async function handleCreate() {
  if (!newTitle.value.trim()) {
    message.warning('请输入小说标题')
    return
  }
  creating.value = true
  const novel = await novelStore.createNovel(newTitle.value.trim(), newAuthor.value.trim())
  creating.value = false
  if (novel) {
    message.success('创建成功')
    showCreateModal.value = false
    newTitle.value = ''
    newAuthor.value = ''
    openNovelEditor(novel)
  }
}

// === 导入小说 ===
const showImportModal = ref(false)
const importTitle = ref('')
const importContent = ref('')
const importing = ref(false)
const importStatus = ref('') // 导入进度提示
const showSplitPreview = ref(false)
const splitResult = ref<SplitResult | null>(null)

async function handleFileSelect(file: File) {
  const text = await file.text()
  importContent.value = text
  // 自动用文件名作为标题
  if (!importTitle.value) {
    importTitle.value = file.name.replace(/\.[^.]+$/, '')
  }
  return true
}

async function handleAISplit() {
  if (!importContent.value.trim()) {
    message.warning('请先选择或粘贴小说内容')
    return
  }
  importing.value = true
  importStatus.value = '正在连接 AI 服务...'
  const startTime = Date.now()
  console.log(`[导入] AI拆分开始 | 内容长度=${importContent.value.length} 字符`)

  try {
    importStatus.value = 'AI 正在分析章节结构，请稍候...'
    const result = await aiApi.splitChapters(importContent.value)
    const elapsed = ((Date.now() - startTime) / 1000).toFixed(1)
    console.log(`[导入] AI拆分完成 | 耗时=${elapsed}s | 章节数=${result.chapters.length} | 角色数=${result.characters?.length || 0}`)
    importStatus.value = `拆分完成，共 ${result.chapters.length} 章（${elapsed}s）`
    splitResult.value = result
    // 如果AI识别出了标题，自动回填
    if (result.title) {
      importTitle.value = result.title
    }
    // 显示识别到的额外信息
    const extras: string[] = []
    if (result.title) extras.push(`标题: ${result.title}`)
    if (result.author) extras.push(`作者: ${result.author}`)
    if (result.description) extras.push(`简介: ${result.description.slice(0, 60)}${result.description.length > 60 ? '...' : ''}`)
    if (extras.length > 0) console.log(`[导入] AI识别到: ${extras.join(' | ')}`)
    showSplitPreview.value = true
  } catch (e: any) {
    const elapsed = ((Date.now() - startTime) / 1000).toFixed(1)
    console.error(`[导入] AI拆分失败 | 耗时=${elapsed}s | 错误=`, e.message)
    message.warning(`AI拆分失败: ${e.message}，您可以手动导入`)
    // 回退：整篇内容作为一章
    splitResult.value = {
      chapters: [{ title: importTitle.value || '第一章', content: importContent.value }],
      characters: [],
      events: [],
      outline: '',
    }
    showSplitPreview.value = true
  } finally {
    importing.value = false
    importStatus.value = ''
  }
}

async function handleManualImport() {
  // 不经过 AI，直接作为一章导入
  if (!importContent.value.trim()) {
    message.warning('请粘贴小说内容')
    return
  }
  importing.value = true
  const novel = await novelStore.importNovel({
    title: importTitle.value || '未命名小说',
    skipAISplit: true,
    chapters: [{ title: '第一章', content: importContent.value }],
  })
  importing.value = false
  if (novel) {
    message.success('导入成功')
    showImportModal.value = false
    resetImport()
    openNovelEditor(novel)
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
    // 保存提取的大纲和角色信息
    await saveExtractedInfo(novel, splitResult.value)
    message.success('导入成功')
    showImportModal.value = false
    showSplitPreview.value = false
    resetImport()
    openNovelEditor(novel)
  }
}

/** 保存AI提取的大纲、角色信息到小说元数据 */
async function saveExtractedInfo(novel: Novel, split: SplitResult) {
  novel.outline = split.outline
  novel.description = split.description
  novel.characters = split.characters
  novel.events = split.events
  // 通过 API 更新小说信息
  try {
    const response = await fetch(`/api/novels/${novel.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        outline: split.outline,
        description: split.description,
        characters: split.characters,
        events: split.events,
      }),
    })
    if (!response.ok) console.warn('保存提取信息失败')
  } catch (e) {
    console.warn('保存提取信息失败:', e)
  }
}

function resetImport() {
  importTitle.value = ''
  importContent.value = ''
  splitResult.value = null
  showSplitPreview.value = false
}

// === 删除小说 ===
function confirmDeleteNovel(novel: Novel) {
  const d = dialog.warning({
    title: '删除小说',
    content: `确定删除《${novel.title}》吗？\n该操作将删除所有章节且不可撤销。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: () => {
      d.loading = true
      novelStore.deleteNovel(novel.id).then((ok) => {
        if (ok) message.success('已删除')
        else message.error('删除失败')
      })
    },
  })
}

// === 重命名小说 ===
const showRenameNovelModal = ref(false)
const renameNovelTarget = ref<Novel | null>(null)
const renameNovelTitle = ref('')

function startRenameNovel(novel: Novel) {
  renameNovelTarget.value = novel
  renameNovelTitle.value = novel.title
  showRenameNovelModal.value = true
}

async function saveRenameNovel() {
  const novel = renameNovelTarget.value
  if (!novel || !renameNovelTitle.value.trim()) {
    message.warning('请输入小说标题')
    return
  }
  const ok = await novelStore.updateNovelMeta(novel.id, { title: renameNovelTitle.value.trim() })
  if (ok) {
    novel.title = renameNovelTitle.value.trim()
    message.success('已重命名')
    showRenameNovelModal.value = false
  } else {
    message.error('重命名失败')
  }
}

// === 修改简介 ===
const showEditDescModal = ref(false)
const editDescTarget = ref<Novel | null>(null)
const editDescContent = ref('')

function startEditDesc(novel: Novel) {
  editDescTarget.value = novel
  editDescContent.value = novel.description || ''
  showEditDescModal.value = true
}

async function saveEditDesc() {
  const novel = editDescTarget.value
  if (!novel) return
  const ok = await novelStore.updateNovelMeta(novel.id, { description: editDescContent.value.trim() })
  if (ok) {
    novel.description = editDescContent.value.trim()
    message.success('简介已更新')
    showEditDescModal.value = false
  } else {
    message.error('更新失败')
  }
}

// === 小说编辑 ===
function openNovelEditor(novel: Novel) {
  // 在当前窗口中导航到编辑器
  router.push({ name: 'NovelEditor', params: { novelId: novel.id } })
}

// === 生命周期 ===
onMounted(() => {
  novelStore.loadNovels()
})

// === 检查AI配置 ===
function checkAIConfig(): boolean {
  return settingsStore.settings?.aiConfigured ?? false
}
</script>

<template>
  <div class="novel-list-container">
    <!-- 顶部操作栏 -->
    <div class="toolbar">
      <n-text class="page-title" style="font-size: 20px; font-weight: 600;">
        我的小说
      </n-text>
      <n-space v-if="novelStore.novels.length > 0">
        <n-button type="primary" @click="showCreateModal = true">
          <template #icon>
            <n-icon><AddIcon /></n-icon>
          </template>
          创建新小说
        </n-button>
        <n-button @click="showImportModal = true">
          <template #icon>
            <n-icon><ImportIcon /></n-icon>
          </template>
          导入小说
        </n-button>
      </n-space>
    </div>

    <!-- 小说网格 -->
    <n-scrollbar style="flex: 1; padding: 16px">
      <template v-if="novelStore.novels.length > 0">
        <n-grid :cols="5" :x-gap="16" :y-gap="16" style="padding-top: 4px">
          <n-gi v-for="novel in novelStore.novels" :key="novel.id">
            <novel-card :novel="novel" @click="openNovelEditor(novel)" @rename="startRenameNovel" @edit-desc="startEditDesc" @delete="confirmDeleteNovel" />
          </n-gi>
        </n-grid>
      </template>

      <!-- 空状态 -->
      <template v-else>
        <div class="empty-state">
          <n-empty description="还没有小说">
            <template #extra>
              <n-space>
                <n-button type="primary" @click="showCreateModal = true">
                  创建一本新小说
                </n-button>
                <n-button @click="showImportModal = true">
                  导入已有小说
                </n-button>
              </n-space>
            </template>
          </n-empty>
        </div>
      </template>
    </n-scrollbar>

    <!-- 创建小说弹框 -->
    <n-modal class="dialog-modal" v-model:show="showCreateModal" title="创建新小说" preset="card" style="width: 420px" :mask-closable="false" @after-leave="() => { newTitle = ''; newAuthor = '' }">
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
          <n-button @click="showCreateModal = false">取消</n-button>
          <n-button type="primary" :loading="creating" @click="handleCreate">创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 导入小说弹框 -->
    <n-modal class="dialog-modal" v-model:show="showImportModal" title="导入小说" preset="card" style="width: 600px" :mask-closable="false" @after-leave="resetImport">
      <div v-if="!showSplitPreview">
        <n-form label-placement="top">
          <n-form-item label="小说标题">
            <n-input v-model:value="importTitle" placeholder="输入标题（可选，默认用文件名）" :disabled="importing" />
          </n-form-item>
          <n-form-item label="选择文件">
            <n-upload :max="1" accept=".txt,.md,.json" :disabled="importing" @change="(e) => e.file.file && handleFileSelect(e.file.file)">
              <n-button :disabled="importing">选择文件</n-button>
            </n-upload>
          </n-form-item>
          <n-form-item label="或粘贴小说内容">
            <n-input v-model:value="importContent" type="textarea" placeholder="粘贴小说全文..." :rows="8" :disabled="importing" />
          </n-form-item>
        </n-form>
        <n-alert v-if="!checkAIConfig()" type="info" :bordered="false" style="margin-bottom: 12px">
          <template #header>未配置 AI 接口</template>
          未检测到 AI 配置，将无法智能拆分章节和提取大纲。您可以在设置中配置 AI 接口，或直接导入为单章节。
        </n-alert>
        <!-- AI 处理进度提示 -->
        <n-alert v-if="importing && importStatus" type="info" :bordered="false" style="margin-bottom: 12px">
          <template #header>AI 处理中</template>
          {{ importStatus }}
        </n-alert>
      </div>
      <div v-else>
        <n-alert type="success" :bordered="false" style="margin-bottom: 12px">已识别 {{ splitResult?.chapters.length || 0 }} 个章节</n-alert>
        <!-- AI 识别到的额外信息 -->
        <n-space v-if="splitResult?.author || splitResult?.description" vertical style="margin-bottom: 12px; padding: 8px 12px; background: #fafafa; border-radius: 6px;">
          <n-text v-if="splitResult?.author" depth="2" style="font-size: 13px;">作者：{{ splitResult.author }}</n-text>
          <n-text v-if="splitResult?.description" depth="3" style="font-size: 12px; line-height: 1.6;">{{ splitResult.description }}</n-text>
        </n-space>
        <n-scrollbar style="max-height: 300px">
          <n-card v-for="(ch, i) in splitResult?.chapters || []" :key="i" :title="ch.title || `第${i+1}章`" size="small" style="margin-bottom: 8px">
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
            <n-button @click="showImportModal = false; resetImport()" :disabled="importing">取消</n-button>
            <n-button :disabled="importing" :loading="importing" @click="handleManualImport">直接导入</n-button>
            <n-button v-if="checkAIConfig()" :disabled="importing" type="primary" :loading="importing" @click="handleAISplit">AI智能拆分</n-button>
          </template>
          <template v-else>
            <n-button @click="showSplitPreview = false" :disabled="importing">返回修改</n-button>
            <n-button type="primary" :disabled="importing" :loading="importing" @click="confirmImport">确认导入</n-button>
          </template>
        </n-space>
      </template>
    </n-modal>

    <!-- 重命名小说弹框 -->
    <n-modal class="dialog-modal" v-model:show="showRenameNovelModal" title="重命名小说" preset="card" style="width: 360px" :mask-closable="false">
      <n-form label-placement="top">
        <n-form-item label="小说标题" required>
          <n-input v-model:value="renameNovelTitle" placeholder="输入新标题" @keyup.enter="saveRenameNovel" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showRenameNovelModal = false">取消</n-button>
          <n-button type="primary" @click="saveRenameNovel">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 修改简介弹框 -->
    <n-modal class="dialog-modal" v-model:show="showEditDescModal" title="修改简介" preset="card" style="width: 480px" :mask-closable="false">
      <n-form label-placement="top">
        <n-form-item label="小说简介">
          <n-input v-model:value="editDescContent" type="textarea" placeholder="输入小说简介" :rows="4" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditDescModal = false">取消</n-button>
          <n-button type="primary" @click="saveEditDesc">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.novel-list-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fff;

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px 0;
    flex-shrink: 0;
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 400px;
  }
}
</style>

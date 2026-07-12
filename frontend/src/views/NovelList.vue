<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useNovelStore } from '../stores/novel'
import NovelCard from '../components/NovelCard.vue'
import CreateNovelDialog from '../components/novel/CreateNovelDialog.vue'
import ImportNovelDialog from '../components/novel/ImportNovelDialog.vue'
import RenameNovelDialog from '../components/novel/RenameNovelDialog.vue'
import NovelInfoDialog from '../components/editor/NovelInfoDialog.vue'
import DeleteNovelConfirmDialog from '../components/novel/DeleteNovelConfirmDialog.vue'
import { NButton, NIcon, NText, NEmpty, NSpace, NGrid, NGi, NScrollbar } from 'naive-ui'
import { AddOutline as AddIcon, DocumentTextOutline as ImportIcon } from '@vicons/ionicons5'
import type { Novel } from '../types'

const router = useRouter()
const novelStore = useNovelStore()

const showCreateModal = ref(false)
const showImportModal = ref(false)
const showRenameModal = ref(false)
const renameTarget = ref<Novel | null>(null)
const showEditDescModal = ref(false)
const descTarget = ref<Novel | null>(null)
const showDeleteModal = ref(false)
const deleteTarget = ref<Novel | null>(null)

function openNovelEditor(novel: Novel) {
  router.push({ name: 'NovelEditor', params: { novelId: novel.id } })
}

function startRenameNovel(novel: Novel) {
  renameTarget.value = novel
  showRenameModal.value = true
}

function startEditDesc(novel: Novel) {
  descTarget.value = novel
  showEditDescModal.value = true
}

function confirmDeleteNovel(novel: Novel) {
  deleteTarget.value = novel
  showDeleteModal.value = true
}

onMounted(() => { novelStore.loadNovels() })
</script>

<template>
  <div class="novel-list-container">
    <div class="toolbar">
      <n-text class="page-title" style="font-size: 20px; font-weight: 600;">我的小说</n-text>
      <n-space v-if="novelStore.novels.length > 0">
        <n-button type="primary" @click="showCreateModal = true">
          <template #icon><n-icon><AddIcon /></n-icon></template>创建新小说
        </n-button>
        <n-button @click="showImportModal = true">
          <template #icon><n-icon><ImportIcon /></n-icon></template>导入小说
        </n-button>
      </n-space>
    </div>

    <n-scrollbar style="flex: 1; padding: 16px">
      <template v-if="novelStore.novels.length > 0">
        <n-grid :cols="5" :x-gap="16" :y-gap="16" style="padding-top: 4px">
          <n-gi v-for="novel in novelStore.novels" :key="novel.id">
            <novel-card :novel="novel" @click="openNovelEditor(novel)"
              @rename="startRenameNovel" @edit-desc="startEditDesc" @delete="confirmDeleteNovel" />
          </n-gi>
        </n-grid>
      </template>
      <template v-else>
        <div class="empty-state">
          <n-empty description="还没有小说">
            <template #extra>
              <n-space>
                <n-button type="primary" @click="showCreateModal = true">创建一本新小说</n-button>
                <n-button @click="showImportModal = true">导入已有小说</n-button>
              </n-space>
            </template>
          </n-empty>
        </div>
      </template>
    </n-scrollbar>

    <CreateNovelDialog v-model:show="showCreateModal" @created="openNovelEditor" />
    <ImportNovelDialog v-model:show="showImportModal" @imported="openNovelEditor" />
    <RenameNovelDialog v-model:show="showRenameModal" :novel="renameTarget" />
    <NovelInfoDialog v-model:show="showEditDescModal" :novel="descTarget" />
    <DeleteNovelConfirmDialog v-model:show="showDeleteModal" :novel="deleteTarget" />
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

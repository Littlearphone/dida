<script setup lang="ts">
import { computed, ref, watch, onUnmounted } from 'vue'
import {
  NModal, NTabs, NTabPane, NInput, NButton, NSpace, NGrid, NGi,
  NCard, NText, NEmpty, NScrollbar, useMessage,
} from 'naive-ui'
import { useNovelStore } from '../../stores/novel'
import type { Novel, Character, Event, NovelRelationship } from '../../types'
import CharacterGraph from './CharacterGraph.vue'
import EventTimeline from './EventTimeline.vue'

const props = defineProps<{
  show: boolean
  /** 传入 novel 表示从列表页打开；不传则使用 store.currentNovel（编辑器内打开） */
  novel?: Novel | null
}>()
const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

const novelStore = useNovelStore()
const message = useMessage()
const saving = ref(false)

// 实际使用的小说对象
const novel = computed(() => props.novel ?? novelStore.currentNovel)

// ------ 本地状态（深拷贝，关闭时不丢失原始数据） ------
const localDescription = ref('')
const localOutline = ref('')
const localCharacters = ref<Character[]>([])
const localRelationships = ref<NovelRelationship[]>([])
const localEvents = ref<Event[]>([])

// ------ 自动保存（防抖） ------
let autoSaveTimer: ReturnType<typeof setTimeout> | null = null
let initialized = false

async function doAutoSave() {
  if (!novel.value) return
  saving.value = true
  await novelStore.updateNovelMeta(novel.value.id, {
    description: localDescription.value.trim() || undefined,
    outline: localOutline.value.trim() || undefined,
    characters: localCharacters.value,
    relationships: localRelationships.value,
    events: localEvents.value,
  })
  saving.value = false
}

function scheduleAutoSave() {
  if (!initialized || !props.show) return
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(doAutoSave, 800)
}

// 初始化 / 重置本地状态
function initLocalState() {
  if (!novel.value) return
  const n = novel.value
  initialized = false
  localDescription.value = n.description || ''
  localOutline.value = n.outline || ''
  localCharacters.value = JSON.parse(JSON.stringify(n.characters || []))
  localRelationships.value = JSON.parse(JSON.stringify(n.relationships || []))
  localEvents.value = JSON.parse(JSON.stringify(n.events || []))
  initialized = true
}

watch(() => props.show, async (open) => {
  if (open) {
    initLocalState()
  } else {
    // 关闭时刷 pending 保存
    if (autoSaveTimer) {
      clearTimeout(autoSaveTimer)
      autoSaveTimer = null
      await doAutoSave()
    }
  }
})

// 各字段变化时自动保存（防抖 800ms）
watch(localDescription, () => scheduleAutoSave())
watch(localOutline, () => scheduleAutoSave())
watch(localEvents, () => scheduleAutoSave(), { deep: true })



onUnmounted(() => {
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
})

// ------ 章节概览数据 ------
const chapterList = computed(() => novelStore.chapters)
const totalWords = computed(() =>
  chapterList.value.reduce((sum, ch) => sum + (ch.wordCount || 0), 0),
)
</script>

<template>
  <n-modal
    class="dialog-modal novel-info-modal"
    :show="show" title="小说信息"
    preset="card"
    style="width: 90vw; max-width: 1200px;"
    :style="{ height: '88vh', maxHeight: '88vh' }"
    :mask-closable="false" draggable
    @update:show="emit('update:show', $event)"
  >
    <!-- 内容区：flex-fill 撑满 n-card 高度 -->
    <div class="info-body">
      <n-tabs type="line" animated class="info-tabs" default-value="overview">
        <!-- Tab 1: 简介与大纲 -->
        <n-tab-pane name="overview" tab="简介与大纲">
          <n-scrollbar class="tab-scroll">
            <div class="tab-pane-inner">
              <div class="field-section">
                <n-text depth="3" class="field-label">小说简介</n-text>
                <n-input
                  v-model:value="localDescription"
                  type="textarea" placeholder="输入小说简介..."
                  :rows="4" :maxlength="2000" show-count
                  :resizable="false"
                />
              </div>
              <div class="field-section fill">
                <n-text depth="3" class="field-label">故事大纲</n-text>
                <n-input
                  v-model:value="localOutline"
                  type="textarea" placeholder="输入故事大纲，支持多段落..."
                  :maxlength="50000" show-count
                  class="outline-input"
                  :resizable="false"
                />
              </div>
            </div>
          </n-scrollbar>
        </n-tab-pane>

        <!-- Tab 2: 人物关系图 -->
        <n-tab-pane name="characters" tab="人物关系图">
          <CharacterGraph
            v-model:characters="localCharacters"
            v-model:relationships="localRelationships"
            :novel-id="novel?.id"
            class="tab-fill"
          />
        </n-tab-pane>

        <!-- Tab 3: 事件时间线 -->
        <n-tab-pane name="events" tab="事件时间线">
          <EventTimeline
            v-model:events="localEvents"
            class="tab-fill"
          />
        </n-tab-pane>

        <!-- Tab 4: 章节概览 -->
        <n-tab-pane name="chapters" tab="章节概览">
          <n-scrollbar class="tab-scroll">
            <div class="tab-pane-inner">
              <n-text depth="3" class="section-summary">
                共 {{ chapterList.length }} 章 · 总计 {{ totalWords.toLocaleString() }} 字
              </n-text>
              <template v-if="chapterList.length > 0">
                <n-grid :cols="4" :x-gap="12" :y-gap="12">
                  <n-gi v-for="(ch, i) in chapterList" :key="ch.id">
                    <n-card :title="`第${i + 1}章`" size="small" hoverable class="chapter-card">
                      <template #header-extra>
                        <n-text depth="3" style="font-size: 12px; margin-right: 4px;">{{ ch.wordCount }} 字</n-text>
                      </template>
                      <n-text>{{ ch.title || '未命名' }}</n-text>
                    </n-card>
                  </n-gi>
                </n-grid>
              </template>
              <n-empty v-else description="暂无章节数据" style="margin-top: 40px;">
                <template #extra>
                  <n-text depth="3">请在编辑器中查看章节概览</n-text>
                </template>
              </n-empty>
            </div>
          </n-scrollbar>
        </n-tab-pane>
      </n-tabs>
    </div>

  </n-modal>
</template>

<style scoped>
/* 弹窗 body 区域 flex-fill 撑满 */
.novel-info-modal :deep(.n-card__content) {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding-bottom: 0;
}

.info-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

/* Tabs 容器 flex-fill */
.info-tabs {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.info-tabs :deep(.n-tabs-nav) {
  flex-shrink: 0;
}
.info-tabs :deep(.n-tabs-content) {
  flex: 1;
  min-height: 0;
  position: relative;
}
.info-tabs :deep(.n-tab-pane) {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* 滚动区域填充 tab-pane */
.tab-scroll {
  flex: 1;
  min-height: 0;
}
.tab-fill {
  flex: 1;
  min-height: 0;
}

/* Tab pane 内部布局 */
.tab-pane-inner {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  padding-right: 8px;
}

/* 字段分区 */
.field-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field-section.fill {
  flex: 1;
  min-height: 0;
}
.field-label {
  font-size: 13px;
  font-weight: 500;
}
.dialog-modal {
  .outline-input {
    flex: 1;
    resize: none;
    min-height: 0;

    & :deep(.n-input__textarea-el) {
      height: 52vh !important;
    }
  }
}

/* 章节概览统计 */
.section-summary {
  font-size: 14px;
  margin-bottom: 12px;
}

/* 章节卡片右侧留白 */
.chapter-card :deep(.n-card-header__extra) {
  margin-right: 4px;
}
</style>

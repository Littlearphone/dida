<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  NButton, NIcon, NCard, NTag, NModal, NForm, NFormItem,
  NInput, NSpace, useMessage, NEmpty,
} from 'naive-ui'
import { AddOutline as AddIcon, CreateOutline as EditIcon, TrashOutline as DeleteIcon } from '@vicons/ionicons5'
import type { Event } from '../../types'

const props = defineProps<{ events: Event[] }>()
const emit = defineEmits<{
  'update:events': [events: Event[]]
}>()

const message = useMessage()

// ------ 编辑弹框状态 ------
const showEdit = ref(false)
const editingIndex = ref(-1)
const editName = ref('')
const editTime = ref('')
const editDesc = ref('')

function openAdd() {
  editingIndex.value = -1
  editName.value = ''
  editTime.value = ''
  editDesc.value = ''
  showEdit.value = true
}

function openEdit(index: number) {
  const e = props.events[index]
  editingIndex.value = index
  editName.value = e.name
  editTime.value = e.timeOrder || ''
  editDesc.value = e.description || ''
  showEdit.value = true
}

function saveEvent() {
  if (!editName.value.trim()) {
    message.warning('请输入事件名称')
    return
  }
  const list = [...props.events]
  const evt: Event = {
    name: editName.value.trim(),
    timeOrder: editTime.value.trim() || undefined,
    description: editDesc.value.trim() || undefined,
  }
  if (editingIndex.value >= 0) {
    list[editingIndex.value] = evt
  } else {
    list.push(evt)
  }
  emit('update:events', list)
  showEdit.value = false
}

function removeEvent(index: number) {
  const list = [...props.events]
  list.splice(index, 1)
  emit('update:events', list)
}

// 左右交替
const timelineEvents = computed(() =>
  props.events.map((e, i) => ({ ...e, _index: i, _side: i % 2 === 0 ? 'left' : 'right' as const })),
)
</script>

<template>
  <div class="timeline-wrapper">
    <!-- 顶部工具栏 -->
    <div class="timeline-toolbar">
      <n-button size="small" type="primary" @click="openAdd">
        <template #icon><n-icon><AddIcon/></n-icon></template>添加事件
      </n-button>
      <n-text v-if="events.length > 0" depth="3" style="font-size: 13px;">共 {{ events.length }} 个事件</n-text>
    </div>

    <!-- 时间线内容 -->
    <div v-if="events.length > 0" class="timeline-container">
      <div class="timeline-line" />
      <div
        v-for="evt in timelineEvents"
        :key="evt._index"
        class="timeline-item"
        :class="evt._side"
      >
        <!-- 中间圆点 -->
        <div class="timeline-dot" />
        <!-- 时间标签 -->
        <div v-if="evt.timeOrder" class="timeline-tag">
          <n-tag size="small" :bordered="false">
            {{ evt.timeOrder }}
          </n-tag>
        </div>
        <!-- 内容卡片 -->
        <n-card class="timeline-card" size="small" hoverable>
          <template #header>
            <div class="card-header">
              <n-text strong>{{ evt.name }}</n-text>
              <div class="card-actions">
                <n-button text size="tiny" @click="openEdit(evt._index)">
                  <template #icon><n-icon size="14"><EditIcon/></n-icon></template>
                </n-button>
                <n-button text size="tiny" style="color: #d03050;" @click="removeEvent(evt._index)">
                  <template #icon><n-icon size="14"><DeleteIcon/></n-icon></template>
                </n-button>
              </div>
            </div>
          </template>
          <n-text v-if="evt.description" depth="3" style="font-size: 13px; line-height: 1.6; white-space: pre-wrap;">
            {{ evt.description }}
          </n-text>
        </n-card>
      </div>
    </div>

    <!-- 空状态 -->
    <n-empty v-else description="还没有事件" class="timeline-empty">
      <template #extra>
        <n-button size="small" @click="openAdd">添加第一个事件</n-button>
      </template>
    </n-empty>

    <!-- 编辑弹框 -->
    <n-modal class="dialog-modal" :show="showEdit" title="编辑事件" preset="card"
      style="width: 480px;" :mask-closable="false" draggable
      @update:show="showEdit = $event">
      <n-form label-placement="top">
        <n-form-item label="事件名称" required>
          <n-input v-model:value="editName" placeholder="输入事件名称" />
        </n-form-item>
        <n-form-item label="时间顺序">
          <n-input v-model:value="editTime" placeholder="例：第1章 / 序章 / 三年前..." />
        </n-form-item>
        <n-form-item label="描述">
          <n-input v-model:value="editDesc" type="textarea" :rows="3" placeholder="事件描述" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showEdit = false">取消</n-button>
          <n-button type="primary" @click="saveEvent">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.timeline-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.timeline-toolbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.timeline-container {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  position: relative;
  padding: 8px 0;
}

/* 中央竖线 */
.timeline-line {
  position: absolute;
  left: 50%;
  top: 0;
  bottom: 0;
  width: 2px;
  background: linear-gradient(to bottom, #e0e0e0, #c0c0c0, #e0e0e0);
  transform: translateX(-50%);
}

/* 每个事件占一行，左右交替 */
.timeline-item {
  display: flex;
  align-items: flex-start;
  position: relative;
  margin-bottom: 24px;
  min-height: 60px;
}

/* 中间圆点 */
.timeline-dot {
  position: absolute;
  left: 50%;
  top: 8px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: #2080f0;
  border: 3px solid #fff;
  box-shadow: 0 0 0 2px #2080f0;
  transform: translateX(-50%);
  z-index: 1;
}

/* 时间标签 */
.timeline-tag {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  margin-top: -4px;
  z-index: 2;
}

/* 事件卡片 - 左侧 */
.timeline-item.left .timeline-tag {
  left: calc(50% - 16px);
  transform: translateX(-100%);
  margin-right: 12px;
}
.timeline-item.left .timeline-card {
  margin-left: calc(50% + 20px);
  margin-right: 40px;
}

/* 事件卡片 - 右侧 */
.timeline-item.right .timeline-tag {
  left: calc(50% + 16px);
  margin-left: 12px;
}
.timeline-item.right .timeline-card {
  margin-right: calc(50% + 20px);
  margin-left: 40px;
}

/* 卡片：覆盖 NaiveUI small 模式的紧凑 padding */
.timeline-card {
  flex: 1;
  max-width: 45%;
}
.timeline-card :deep(.n-card-header) {
  /* 只加宽右侧 padding，保留 NaiveUI 默认的上下和左侧 */
  padding-right: 18px !important;
}
.timeline-card :deep(.n-card__content) {
  padding: 8px 18px 14px !important;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}
.card-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
  padding-left: 8px;
}

.timeline-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>

<script setup lang="ts">
import { ref } from 'vue'
import {
  NButton, NIcon, NModal, NForm, NFormItem,
  NInput, NSpace, NTimeline, NTimelineItem,
  NScrollbar, useMessage, NEmpty, NText,
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
const editDesc = ref('')

function openAdd() {
  editingIndex.value = -1
  editName.value = ''
  editDesc.value = ''
  showEdit.value = true
}

function openEdit(index: number) {
  const e = props.events[index]
  editingIndex.value = index
  editName.value = e.name
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
    <n-scrollbar v-if="events.length > 0" class="timeline-scroll">
      <div class="timeline-inner">
        <n-timeline item-placement="left" :icon-size="28">
          <n-timeline-item
            v-for="(evt, i) in events"
            :key="i"
          >
            <template #icon>
              <span class="timeline-num">{{ i + 1 }}</span>
            </template>
            <template #header>
              <div class="item-header">
                <n-text strong>{{ evt.name }}</n-text>
                <div class="item-actions">
                  <n-button text size="small" @click="openEdit(i)">
                    <template #icon><n-icon size="16"><EditIcon/></n-icon></template>
                  </n-button>
                  <n-button text size="small" style="color: #d03050;" @click="removeEvent(i)">
                    <template #icon><n-icon size="16"><DeleteIcon/></n-icon></template>
                  </n-button>
                </div>
              </div>
            </template>
            <n-text v-if="evt.description" depth="3" style="font-size: 13px; line-height: 1.6; white-space: pre-wrap;">
              {{ evt.description }}
            </n-text>
          </n-timeline-item>
        </n-timeline>
      </div>
    </n-scrollbar>

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

<style lang="scss" scoped>
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

.timeline-scroll {
  flex: 1;
  min-height: 0;
}
.timeline-inner {
  padding-top: 10px;
}

/* 事件标题行：名称后紧跟操作按钮 */
.item-header {
  display: flex;
  align-items: center;
  width: 100%;
}
.item-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
  margin-left: 8px;

  /* 编辑图标：hover 时笔杆轻旋 */
  :deep(.n-button:first-child:hover .n-icon) {
    animation: icon-wiggle 0.3s ease-in-out;
  }

  /* 删除图标：hover 时垃圾桶盖子开合 */
  :deep(.n-button:last-child:hover .n-icon) {
    animation: trash-lid 0.35s ease-in-out;
  }
}

@keyframes icon-wiggle {
  0%, 100% { transform: rotate(0deg); }
  25% { transform: rotate(-8deg); }
  75% { transform: rotate(8deg); }
}

@keyframes trash-lid {
  0% { transform: perspective(60px) rotateX(0deg); transform-origin: 50% 0%; }
  35% { transform: perspective(60px) rotateX(-40deg); transform-origin: 50% 0%; }
  60% { transform: perspective(60px) rotateX(-25deg); transform-origin: 50% 0%; }
  100% { transform: perspective(60px) rotateX(0deg); transform-origin: 50% 0%; }
}

.timeline-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 序号圆点，替换 NTimelineItem 默认的 dot */
/* 容器由 NaiveUI 控制宽高（--n-icon-size: 28px），flex 居中 */
.timeline-num {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background: #2080f0;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
}
</style>

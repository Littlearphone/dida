<script setup lang="ts">
import { ref } from 'vue'
import {
  NButton, NIcon, NModal, NForm, NFormItem,
  NInput, NSpace, NTimeline, NTimelineItem,
  NScrollbar, useMessage, useDialog, NEmpty, NText,
} from 'naive-ui'
import { AddOutline as AddIcon, CreateOutline as EditIcon, CloseOutline as CloseIcon, CaretUpOutline as UpIcon, CaretDownOutline as DownIcon } from '@vicons/ionicons5'
import type { Event } from '@/types'

const props = defineProps<{ events: Event[] }>()
const emit = defineEmits<{
  'update:events': [events: Event[]]
}>()

const message = useMessage()
const dialog = useDialog()

function moveUp(index: number) {
  if (index <= 0) return
  const list = [...props.events];
  [list[index - 1], list[index]] = [list[index], list[index - 1]]
  emit('update:events', list)
}

function moveDown(index: number) {
  if (index >= props.events.length - 1) return
  const list = [...props.events];
  [list[index], list[index + 1]] = [list[index + 1], list[index]]
  emit('update:events', list)
}

// ------ 编辑弹框 ------
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
  const name = props.events[index].name
  dialog.warning({
    title: '删除事件',
    content: `确定删除「${name}」吗？`,
    positiveText: '删除',
    negativeText: '取消',
    draggable: false,
    onPositiveClick: () => {
      const list = [...props.events]
      list.splice(index, 1)
      emit('update:events', list)
    },
  })
}
</script>

<template>
  <div class="timeline-wrapper">
    <div class="timeline-toolbar">
      <n-button size="small" type="primary" @click="openAdd">
        <template #icon><n-icon><AddIcon/></n-icon></template>添加事件
      </n-button>
      <n-text v-if="events.length > 0" depth="3" style="font-size: 13px;">共 {{ events.length }} 个事件</n-text>
    </div>

    <n-scrollbar v-if="events.length > 0" class="timeline-scroll">
      <div class="timeline-inner">
        <n-timeline item-placement="left" :icon-size="28">
          <n-timeline-item v-for="(evt, i) in events" :key="i">
            <template #icon>
              <span class="timeline-num">{{ i + 1 }}</span>
            </template>
            <template #header>
              <div class="item-header">
                <n-text strong>{{ evt.name }}</n-text>
                <div class="item-actions">
                  <!-- 上下移动 -->
                  <n-button text size="small" :disabled="i === 0" @click="moveUp(i)">
                    <template #icon><n-icon size="14"><UpIcon/></n-icon></template>
                  </n-button>
                  <n-button text size="small" :disabled="i === events.length - 1" @click="moveDown(i)">
                    <template #icon><n-icon size="14"><DownIcon/></n-icon></template>
                  </n-button>
                  <n-button text size="small" @click="openEdit(i)">
                    <template #icon><n-icon size="16"><EditIcon/></n-icon></template>
                  </n-button>
                  <n-button text size="small" style="color: #d03050;" @click="removeEvent(i)">
                    <template #icon>
                      <svg class="trash-icon" viewBox="0 0 512 512" width="16" height="16"
                        fill="none" stroke="currentColor" stroke-width="32"
                        stroke-linecap="round" stroke-linejoin="round">
                        <g class="trash-top">
                          <path d="M80 112h352" />
                          <path d="M192 112V72a23.93 23.93 0 0 1 24-24h80a23.93 23.93 0 0 1 24 24v40" />
                        </g>
                        <path d="M112 112l20 320c.95 18.49 14.4 32 32 32h184c17.67 0 30.87-13.51 32-32l20-320" />
                        <path d="M256 176v224" />
                        <path d="M184 176l8 224" />
                        <path d="M328 176l-8 224" />
                      </svg>
                    </template>
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

    <n-empty v-else description="还没有事件" class="timeline-empty">
      <template #extra>
        <n-button size="small" @click="openAdd">添加第一个事件</n-button>
      </template>
    </n-empty>

    <!-- 编辑弹框 -->
    <n-modal class="dialog-modal" :show="showEdit" :title="editingIndex >= 0 ? '编辑事件' : '添加事件'" preset="card"
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

/* 事件标题行 */
.item-header {
  display: flex;
  align-items: center;
  width: 100%;
}
.item-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
  margin-left: 8px;

  :deep(.n-button:first-child:hover .n-icon) {
    animation: icon-wiggle 0.3s ease-in-out;
  }

  :deep(.n-button:last-child:hover .trash-top) {
    animation: trash-lid 0.35s ease-in-out;
  }
  :deep(.n-button:last-child:hover .trash-icon) {
    animation: trash-drop 0.35s ease-in-out;
  }
}

@keyframes icon-wiggle {
  0%, 100% { transform: rotate(0deg); }
  25% { transform: rotate(-8deg); }
  75% { transform: rotate(8deg); }
}
@keyframes trash-lid {
  0%, 100% { transform: rotate(0deg); }
  35% { transform: rotate(35deg); }
  65% { transform: rotate(20deg); }
}
@keyframes trash-drop {
  0%, 100% { transform: translateY(0); }
  35% { transform: translateY(3px); }
  65% { transform: translateY(1.5px); }
}
.trash-icon .trash-top {
  transform-box: view-box;
  transform-origin: 432px 112px;
}

.timeline-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 序号圆点 */
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

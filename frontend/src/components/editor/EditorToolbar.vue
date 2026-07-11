<script setup lang="ts">
import { NButton, NDivider, NIcon, NInputNumber, NSelect, NText } from 'naive-ui'
import {
  ArrowUndoOutline as UndoIcon, ArrowRedoOutline as RedoIcon,
  SearchOutline as SearchIcon,
  HammerOutline as FormatIcon, SaveOutline as SaveIcon,
} from '@vicons/ionicons5'

defineProps<{
  undoable: boolean; redoable: boolean
  fontSize: number; lineHeight: number; paragraphSpacing: number
  isBold: boolean; isItalic: boolean; fontFamily: string
  showSearch: boolean
  canFormat: boolean
  fontOptions: { label: string; value: string }[]
  contentChanged: boolean; showSavedIndicator: boolean
  autoSaveEnabled: boolean
}>()

const emit = defineEmits<{
  undo: []; redo: []
  'update:fontSize': [v: number]; 'update:lineHeight': [v: number]
  'update:paragraphSpacing': [v: number]
  'update:isBold': [v: boolean]; 'update:isItalic': [v: boolean]
  'update:fontFamily': [v: string]
  'update:showSearch': [v: boolean]
  formatContent: []; save: []
}>()
</script>

<template>
  <div style="border-bottom: 1px solid #eee; padding: 4px 16px; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; flex-shrink: 0;">
    <n-button quaternary size="tiny" :disabled="!undoable" @click="emit('undo')" title="撤销 (Ctrl+Z)">
      <template #icon><n-icon size="16"><UndoIcon/></n-icon></template>
    </n-button>
    <n-button quaternary size="tiny" :disabled="!redoable" @click="emit('redo')" title="重做 (Ctrl+Shift+Z)">
      <template #icon><n-icon size="16"><RedoIcon/></n-icon></template>
    </n-button>
    <n-divider vertical />
    <n-button quaternary size="tiny" :type="isBold ? 'primary' : 'default'" @click="emit('update:isBold', !isBold)" title="全局粗体 (Ctrl+B)"><b>B</b></n-button>
    <n-button quaternary size="tiny" :type="isItalic ? 'primary' : 'default'" @click="emit('update:isItalic', !isItalic)" title="全局斜体 (Ctrl+I)"><i>I</i></n-button>
    <n-button quaternary size="tiny" @click="emit('update:showSearch', !showSearch)" title="搜索 (Ctrl+F)">
      <template #icon><n-icon size="16"><SearchIcon/></n-icon></template>
    </n-button>
    <n-button quaternary size="tiny" :disabled="!canFormat" @click="emit('formatContent')" title="删除多余空行 (Ctrl+Shift+F)">
      <template #icon><n-icon size="16"><FormatIcon/></n-icon></template>
    </n-button>
    <n-divider vertical />
    <n-select :value="fontFamily" :options="fontOptions" size="small" style="width: 120px;" placeholder="字体"
      @update:value="(v: string) => emit('update:fontFamily', v)" />
    <n-divider vertical />
    <n-text depth="3" style="font-size: 12px; white-space: nowrap;">字号</n-text>
    <n-input-number :value="fontSize" :min="12" :max="32" size="small" style="width: 82px;"
      @update:value="(v: number | null) => v && emit('update:fontSize', v)" />
    <n-divider vertical />
    <n-text depth="3" style="font-size: 12px; white-space: nowrap;">行距</n-text>
    <n-input-number :value="lineHeight" :min="1" :max="3" :step="0.1" size="small" style="width: 82px;"
      @update:value="(v: number | null) => v && emit('update:lineHeight', v)" />
    <n-divider vertical />
    <n-text depth="3" style="font-size: 12px; white-space: nowrap;">段距</n-text>
    <n-input-number :value="paragraphSpacing" :min="0" :max="48" :step="4" size="small" style="width: 82px;"
      @update:value="(v: number | null) => v && emit('update:paragraphSpacing', v)" />
    <template v-if="!autoSaveEnabled">
      <n-divider vertical />
      <n-button type="warning" size="tiny" :disabled="!contentChanged" @click="emit('save')">
        <template #icon><n-icon size="16"><SaveIcon/></n-icon></template>保存
      </n-button>
    </template>
    <div style="flex: 1" />
    <n-text v-if="contentChanged || showSavedIndicator"
      :style="{ fontSize: '12px', color: contentChanged ? '#e6a23c' : '#18a058' }">
      {{ contentChanged ? '未保存' : '已保存' }}
    </n-text>
  </div>
</template>

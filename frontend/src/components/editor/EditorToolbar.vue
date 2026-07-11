<script setup lang="ts">
import { NButton, NDivider, NIcon, NInput, NInputNumber, NSelect, NText } from 'naive-ui'
import {
  ArrowUndoOutline as UndoIcon, ArrowRedoOutline as RedoIcon,
  CopyOutline as CopyIcon, SearchOutline as SearchIcon,
  CutOutline as CutIcon, SaveOutline as SaveIcon,
} from '@vicons/ionicons5'

defineProps<{
  undoable: boolean; redoable: boolean
  fontSize: number; lineHeight: number; paragraphSpacing: number
  isBold: boolean; isItalic: boolean; fontFamily: string
  showSearch: boolean; searchQuery: string
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
  'update:showSearch': [v: boolean]; 'update:searchQuery': [v: string]
  copySelection: []; search: []; formatContent: []; save: []
}>()
</script>

<template>
  <div style="border-bottom: 1px solid #eee; padding: 4px 16px; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; flex-shrink: 0;">
    <n-button quaternary size="tiny" :disabled="!undoable" @click="emit('undo')">
      <template #icon><n-icon size="16"><UndoIcon/></n-icon></template>
    </n-button>
    <n-button quaternary size="tiny" :disabled="!redoable" @click="emit('redo')">
      <template #icon><n-icon size="16"><RedoIcon/></n-icon></template>
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
    <n-divider vertical />
    <n-button quaternary size="tiny" :type="isBold ? 'primary' : 'default'" @click="emit('update:isBold', !isBold)"><b>B</b></n-button>
    <n-button quaternary size="tiny" :type="isItalic ? 'primary' : 'default'" @click="emit('update:isItalic', !isItalic)"><i>I</i></n-button>
    <n-divider vertical />
    <n-button quaternary size="tiny" @click="emit('copySelection')">
      <template #icon><n-icon size="16"><CopyIcon/></n-icon></template>复制
    </n-button>
    <n-button quaternary size="tiny" @click="emit('update:showSearch', !showSearch)">
      <template #icon><n-icon size="16"><SearchIcon/></n-icon></template>搜索
    </n-button>
    <n-button quaternary size="tiny" @click="emit('formatContent')">
      <template #icon><n-icon size="16"><CutIcon/></n-icon></template>格式化
    </n-button>
    <template v-if="!autoSaveEnabled">
      <n-button type="warning" size="tiny" :disabled="!contentChanged" @click="emit('save')">
        <template #icon><n-icon size="16"><SaveIcon/></n-icon></template>保存
      </n-button>
      <n-divider vertical />
    </template>
    <template v-if="showSearch">
      <n-divider vertical />
      <n-input :value="searchQuery" placeholder="搜索正文..." size="small" style="width: 200px"
        @update:value="(v: string) => emit('update:searchQuery', v)"
        @keyup.enter="emit('search')" />
    </template>
    <div style="flex: 1" />
    <n-text v-if="contentChanged || showSavedIndicator"
      :style="{ fontSize: '12px', color: contentChanged ? '#e6a23c' : '#18a058' }">
      {{ contentChanged ? '未保存' : '已保存' }}
    </n-text>
  </div>
</template>

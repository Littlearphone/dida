<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NText, NSpace, NTooltip } from 'naive-ui'
import { SparklesOutline as SparklesIcon, ColorWandOutline as WandIcon, CreateOutline as ExpandIcon, SettingsOutline as AISetupIcon, InformationCircleOutline as InfoIcon } from '@vicons/ionicons5'

const props = defineProps<{ wordCount: number; aiConfigured: boolean; contentChanged: boolean }>()
const saveLabel = computed(() => props.contentChanged ? '未保存' : '已保存')
const saveColor = computed(() => props.contentChanged ? '#e6a23c' : '#18a058')

const emit = defineEmits<{
  continue: []
  polish: []
  expand: []
  setupAI: []
  showInfo: []
}>()
</script>

<template>
  <div style="border-top: 1px solid #eee; padding: 6px 16px; display: flex; align-items: center; gap: 8px; flex-shrink: 0;">
    <n-text depth="3" style="font-size: 12px;">共 {{ wordCount }} 字</n-text>
    <n-text :style="{ fontSize: '12px', color: saveColor, marginLeft: '12px' }">{{ saveLabel }}</n-text>
    <div style="flex: 1" />
    <n-button size="tiny" quaternary @click="emit('showInfo')" title="小说信息">
      <template #icon><n-icon size="14"><InfoIcon/></n-icon></template>
      <span style="font-size: 12px;">信息</span>
    </n-button>
    <template v-if="aiConfigured">
      <n-button size="small" type="primary" @click="emit('continue')">
        <template #icon><n-icon size="16"><SparklesIcon/></n-icon></template>AI 续写
      </n-button>
      <n-button size="small" type="info" @click="emit('polish')">
        <template #icon><n-icon size="16"><WandIcon/></n-icon></template>AI 润色
      </n-button>
      <n-button size="small" type="warning" @click="emit('expand')">
        <template #icon><n-icon size="16"><ExpandIcon/></n-icon></template>AI 扩写
      </n-button>
    </template>
    <template v-else>
      <n-button size="tiny" secondary @click="emit('setupAI')">
        <template #icon><n-icon size="14"><AISetupIcon/></n-icon></template>设置 AI
      </n-button>
    </template>
  </div>
</template>

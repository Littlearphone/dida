<script setup lang="ts">
import { NButton, NText, NSpace } from 'naive-ui'
import { SparklesOutline as SparklesIcon, ColorWandOutline as WandIcon, CreateOutline as ExpandIcon, SettingsOutline as AISetupIcon } from '@vicons/ionicons5'

defineProps<{ wordCount: number; aiConfigured: boolean }>()

const emit = defineEmits<{
  continue: []
  polish: []
  expand: []
  setupAI: []
}>()
</script>

<template>
  <div style="border-top: 1px solid #eee; padding: 6px 16px; display: flex; align-items: center; gap: 8px; flex-shrink: 0;">
    <n-text depth="3" style="font-size: 12px;">共 {{ wordCount }} 字</n-text>
    <div style="flex: 1" />
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

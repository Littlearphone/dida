<script setup lang="ts">
import { ref, watch } from 'vue'
import { NModal, NAlert, NForm, NFormItem, NInput, NButton, NSelect, NSpace, NText, useMessage } from 'naive-ui'
import { useSettingsStore } from '../../stores/settings'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ 'update:show': [value: boolean] }>()

const settingsStore = useSettingsStore()
const message = useMessage()

const setupEndpoint = ref('https://api.deepseek.com')
const setupApiKey = ref('')
const setupModel = ref('deepseek-chat')
const setupHasKey = ref(false)
const showCustomModel = ref(false)
const showChangeKey = ref(false)

const modelOptions = [
  { label: 'DeepSeek Chat', value: 'deepseek-chat' },
  { label: 'DeepSeek V3', value: 'deepseek-chat' },
  { label: 'DeepSeek R1', value: 'deepseek-reasoner' },
  { label: 'GPT-4o', value: 'gpt-4o' },
  { label: 'GPT-4o Mini', value: 'gpt-4o-mini' },
  { label: 'Claude Sonnet', value: 'claude-sonnet-4-20250514' },
  { label: '自定义', value: '__custom__' },
]

watch(setupModel, (val) => { showCustomModel.value = val === '__custom__' })

watch(() => props.show, async (open) => {
  if (!open) return
  if (!settingsStore.settings) await settingsStore.load()
  if (settingsStore.settings) {
    setupEndpoint.value = settingsStore.settings.endpoint || 'https://api.deepseek.com'
    setupModel.value = settingsStore.settings.aiModel || 'deepseek-chat'
    setupHasKey.value = settingsStore.settings.hasApiKey
  }
  setupApiKey.value = ''
  showCustomModel.value = !modelOptions.some(o => o.value === setupModel.value)
  showChangeKey.value = false
})

async function saveAISetup() {
  const form: Record<string, any> = { endpoint: setupEndpoint.value, aiModel: setupModel.value }
  if (setupApiKey.value) form.apiKey = setupApiKey.value
  const ok = await settingsStore.update(form as any)
  if (ok) {
    if (setupApiKey.value) setupHasKey.value = true
    message.success('AI 设置已保存')
    emit('update:show', false)
  }
}
</script>

<template>
  <n-modal class="dialog-modal" :show="show" title="配置 AI 提供商" preset="card" style="width: 80vw"
    :mask-closable="false" draggable @update:show="emit('update:show', $event)">
    <n-alert type="info" :bordered="false" style="margin-bottom: 12px">
      需要配置 AI 接口才能使用智能续写、润色等功能。目前支持 DeepSeek 兼容接口。
    </n-alert>
    <n-form label-placement="top">
      <n-form-item label="接口地址" required>
        <n-input v-model:value="setupEndpoint" placeholder="https://api.deepseek.com" />
      </n-form-item>
      <n-form-item label="模型">
        <n-select v-model:value="setupModel" :options="modelOptions" placeholder="选择模型" />
      </n-form-item>
      <n-form-item v-if="showCustomModel" label="自定义模型名">
        <n-input v-model:value="setupModel" placeholder="输入模型名称" />
      </n-form-item>
      <n-form-item v-if="setupHasKey && !showChangeKey" label="API Key">
        <div style="display: flex; align-items: center; gap: 8px; width: 100%;">
          <n-text style="color: #18a058;">✓ API Key 已配置</n-text>
          <n-button size="tiny" quaternary @click="showChangeKey = true">更换</n-button>
        </div>
      </n-form-item>
      <n-form-item v-else label="API Key">
        <n-input v-model:value="setupApiKey" type="password" show-password-on="click" placeholder="输入新的 API Key" />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-button quaternary @click="emit('update:show', false)">暂不配置</n-button>
      <n-button type="primary" @click="saveAISetup">保存</n-button>
    </template>
  </n-modal>
</template>

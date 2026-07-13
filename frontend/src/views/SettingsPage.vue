<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useSettingsStore } from '../stores/settings'
import {
  NInput, NForm, NFormItem, NButton, NTabPane, NTabs,
  NInputNumber, NSwitch, NText, NSpace, NAlert, useMessage,
} from 'naive-ui'

const settingsStore = useSettingsStore()
const message = useMessage()

// 表单数据
const novelPath = ref('')
const endpoint = ref('')
const apiKey = ref('')
const aiModel = ref('deepseek-chat')
const autoSave = ref(true)
const autoSaveMs = ref(3000)
const defaultFontSize = ref(16)
const defaultLineSpacing = ref(1.8)

const saving = ref(false)
const testingAI = ref(false)

// 跟踪是否已有密钥（不回填原始值到输入框，避免误覆盖）
const hasExistingKey = ref(false)

// 加载设置到表单
onMounted(() => {
  if (settingsStore.settings) {
    applySettings(settingsStore.settings)
  }
})

// 监听设置变更
watch(() => settingsStore.settings, (s) => {
  if (s) applySettings(s)
})

function applySettings(s: any) {
  novelPath.value = s.novelPath || ''
  endpoint.value = s.endpoint || 'https://api.deepseek.com'
  apiKey.value = ''
  hasExistingKey.value = !!s.hasApiKey
  aiModel.value = s.aiModel || 'deepseek-chat'
  autoSave.value = s.autoSave !== false
  autoSaveMs.value = s.autoSaveMs || 3000
  defaultFontSize.value = s.defaultFontSize || 16
  defaultLineSpacing.value = s.defaultLineSpacing || 1.8
}

/** 保存小说设置 */
async function saveNovelSettings() {
  saving.value = true
  const ok = await settingsStore.update({
    novelPath: novelPath.value,
    autoSave: autoSave.value,
    autoSaveMs: autoSaveMs.value,
    defaultFontSize: defaultFontSize.value,
    defaultLineSpacing: defaultLineSpacing.value,
  })
  saving.value = false
  if (ok) message.success('小说设置已保存')
}

/** 保存 AI 设置 */
async function saveAISettings() {
  saving.value = true
  const form: Record<string, any> = {
    endpoint: endpoint.value,
    aiModel: aiModel.value,
  }
  // 仅当用户输入了新密钥才提交，避免覆盖已有密钥
  if (apiKey.value) {
    form.apiKey = apiKey.value
  }
  const ok = await settingsStore.update(form as any)
  saving.value = false
  if (ok) {
    if (apiKey.value) {
      hasExistingKey.value = true
    }
    message.success('AI 设置已保存')
  }
}

/** 测试 AI 连接 */
async function testAIConnection() {
  // 用当前表单值或已保存的密钥测试，不要求表单已输入
  if (!endpoint.value && !settingsStore.settings?.endpoint) {
    message.warning('请先填写接口地址')
    return
  }
  testingAI.value = true
  try {
    const { checkAIStatus } = await import('../api/ai')
    const status = await checkAIStatus()
    if (status.connected) {
      message.success('AI 连接成功！')
    } else {
      message.warning(`连接失败: ${status.error || '未知错误'}`)
    }
  } catch (e: any) {
    message.error(`连接失败: ${e.message}`)
  } finally {
    testingAI.value = false
  }
}
</script>

<template>
  <div class="settings-container">
    <n-text style="font-size: 20px; font-weight: 600; display: block; padding: 16px 20px 0;">
      系统设置
    </n-text>

    <n-tabs type="line" animated style="padding: 16px 20px;" default-value="novel">
      <!-- 小说设置 -->
      <n-tab-pane name="novel" tab="小说设置">
        <div class="settings-form">
          <n-form label-placement="left" label-width="140" label-align="right">
            <n-form-item label="小说保存路径">
              <n-input v-model:value="novelPath" placeholder="选择小说文件保存位置" />
            </n-form-item>

            <n-divider />

            <n-form-item label="自动保存">
              <n-space align="center">
                <n-switch v-model:value="autoSave" />
                <n-text v-if="autoSave" depth="3">开启</n-text>
                <n-text v-else depth="3">关闭</n-text>
              </n-space>
            </n-form-item>

            <n-form-item v-if="autoSave" label="自动保存间隔">
              <n-space align="center">
                <n-input-number v-model:value="autoSaveMs" :min="1000" :max="60000" :step="1000" />
                <n-text depth="3">毫秒</n-text>
              </n-space>
            </n-form-item>

            <n-form-item label="默认字号">
              <n-space align="center">
                <n-input-number v-model:value="defaultFontSize" :min="12" :max="32" />
                <n-text depth="3">px</n-text>
              </n-space>
            </n-form-item>

            <n-form-item label="默认行距">
              <n-space align="center">
                <n-input-number v-model:value="defaultLineSpacing" :min="1" :max="3" :step="0.1" />
              </n-space>
            </n-form-item>

            <n-form-item label=" ">
              <n-button type="primary" :loading="saving" @click="saveNovelSettings">
                保存小说设置
              </n-button>
            </n-form-item>
          </n-form>
        </div>
      </n-tab-pane>

      <!-- AI 设置 -->
      <n-tab-pane name="ai" tab="AI 设置">
        <n-alert type="info" :bordered="false" style="margin-bottom: 16px;">
          目前支持 DeepSeek API 兼容接口。请填写您的接口地址和 API Key。
        </n-alert>

        <div class="settings-form">
          <n-form label-placement="left" label-width="140" label-align="right">
            <n-form-item label="接口地址">
              <n-input v-model:value="endpoint" placeholder="https://api.deepseek.com" />
            </n-form-item>

            <n-form-item label="模型名称">
              <n-input v-model:value="aiModel" placeholder="deepseek-chat" />
            </n-form-item>

            <n-form-item label="API Key">
              <n-input
                v-model:value="apiKey"
                type="password"
                show-password-on="click"
                :placeholder="hasExistingKey ? '已有密钥，输入新值替换' : '输入 API Key'"
              />
            </n-form-item>

            <n-form-item label=" ">
              <n-space>
                <n-button type="primary" :loading="saving" @click="saveAISettings">
                  保存 AI 设置
                </n-button>
                <n-button :loading="testingAI" @click="testAIConnection">
                  测试连接
                </n-button>
              </n-space>
            </n-form-item>
          </n-form>
        </div>
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<style lang="scss" scoped>
.settings-container {
  height: 100%;
  background: #fff;
  overflow-y: auto;

  .settings-form {
    max-width: 560px;
  }
}
</style>

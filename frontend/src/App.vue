<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useSettingsStore } from './stores/settings'
import { zhCN, dateZhCN } from 'naive-ui'

const settingsStore = useSettingsStore()

/** 阻止页面级缩放（Ctrl+滚轮 / Ctrl++/-） */
function preventPageZoom(e: WheelEvent) {
  if (e.ctrlKey || e.metaKey) e.preventDefault()
}
function preventKeyboardZoom(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && (e.key === '=' || e.key === '-' || e.key === '+' || e.key === '0')) {
    e.preventDefault()
  }
}

onMounted(() => {
  // 应用启动时加载设置
  settingsStore.load()
  document.addEventListener('wheel', preventPageZoom, { passive: false })
  document.addEventListener('keydown', preventKeyboardZoom)
})

onUnmounted(() => {
  document.removeEventListener('wheel', preventPageZoom)
  document.removeEventListener('keydown', preventKeyboardZoom)
})
</script>

<template>
  <!-- NaiveUI 全局配置：使用中文语言包 -->
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN">
    <!-- n-message-provider 是 useMessage() 的前置依赖，必须包裹整个应用 -->
    <n-dialog-provider>
      <n-message-provider>
        <router-view />
      </n-message-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>

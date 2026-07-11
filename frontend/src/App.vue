<script setup lang="ts">
import { onMounted } from 'vue'
import { useSettingsStore } from './stores/settings'
import { zhCN, dateZhCN } from 'naive-ui'

const settingsStore = useSettingsStore()

onMounted(() => {
  // 应用启动时加载设置
  settingsStore.load()
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

<style>
/* 全局样式重置 */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  width: 100%;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  /* WebView2 文字渲染优化 */
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-rendering: optimizeLegibility;
  font-weight: 400;
}
</style>

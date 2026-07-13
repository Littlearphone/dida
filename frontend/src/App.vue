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

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

/* 所有弹框（dialog-modal）统一样式 */
.dialog-modal {
  .n-card-header { border-bottom: 1px solid rgba(0,0,0,0.14); padding: 0 0 0 16px !important; height: 40px; }
  .n-card-header__close { height: 40px; width: 40px; border-radius: 0; color: #999; transition: all 0.15s;
    &:hover { background: #d03050;
      &::before { --n-close-color-hover: #d03050; --n-close-color-pressed: #d03050; }
    }
    &:hover, &:hover * { color: #fff; opacity: 1 !important; }
  }
  .n-card-content { padding: 20px 24px !important; flex: 1; min-height: 0; display: flex; flex-direction: column; }
  .n-card-content > div { flex: 1; min-height: 0; display: flex; flex-direction: column; }
  .n-form-item--top-labelled.n-form-item--top-labelled { grid-template-rows: auto 1fr auto; }
  .n-form-item-blank { align-items: stretch; min-height: 0; height: 100%; }
  .n-input--textarea { height: 100% !important; }
  .n-input--resizable textarea { resize: none !important; }
  .n-card__footer { padding: 0 !important; border-top: 1px solid #eee;
    .n-space { width: 100%; gap: 1px !important; flex-flow: nowrap !important; & > div { width: 100%; } }
    .n-space-item { flex: 1; display: flex; }
    .n-button { width: 100%; border-radius: 0; height: 36px;
      &:not(:last-child) { border-right: 1px solid #eee; }
      &:focus, &:focus-visible,
      &:focus-within { box-shadow: none !important; outline: none !important; }
      &.n-button--quaternary-type {
        &:hover { background: #f5f5f5; }
        &:active { background: #ebebeb; }
      }
    }
  }
}
</style>

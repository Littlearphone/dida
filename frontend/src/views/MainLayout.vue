<script setup lang="ts">
import {ref} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {NIcon} from 'naive-ui'
import {BookOutline as BookIcon, SettingsOutline as SettingsIcon} from '@vicons/ionicons5'

const router = useRouter()
const route = useRoute()

/** 左侧功能栏菜单项 */
const menuItems = [
  { key: 'novels', label: '小说列表', icon: BookIcon },
  { key: 'settings', label: '系统设置', icon: SettingsIcon },
]

// 根据当前路由确定选中项
const activeKey = ref(route.name === 'Settings' ? 'settings' : 'novels')

/** 侧边栏折叠状态（默认折叠，仅显示图标） */
const collapsed = ref(true)

/** 切换菜单时跳转路由 */
function handleMenuClick(key: string) {
  activeKey.value = key
  router.push({ name: key === 'novels' ? 'NovelList' : 'Settings' })
}
</script>

<template>
  <n-layout has-sider style="height: 100vh">
    <!-- 左侧图标侧边栏：折叠时仅图标，展开时显示文字 -->
    <n-layout-sider
      bordered
      :width="170"
      :collapsed-width="48"
      show-trigger="arrow-circle"
      collapse-mode="width"
      v-model:collapsed="collapsed"
      style="background: #f5f5f5; padding-top: 12px;"
    >
      <div class="sidebar-menu" :class="{ collapsed }">
        <div
          v-for="item in menuItems"
          :key="item.key"
          class="sidebar-item"
          :class="{ active: activeKey === item.key }"
          :title="item.label"
          @click="handleMenuClick(item.key)"
        >
          <n-icon :size="22">
            <component :is="item.icon" />
          </n-icon>
          <span v-if="!collapsed" class="sidebar-label">{{ item.label }}</span>
        </div>
      </div>
    </n-layout-sider>

    <!-- 右侧内容区 -->
    <n-layout-content>
      <router-view />
    </n-layout-content>
  </n-layout>
</template>

<style scoped>
.sidebar-item {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 40px;
  padding: 0 12px;
  margin: 0 4px 4px;
  border-radius: 8px;
  cursor: pointer;
  color: #666;
  transition: all 0.2s;
  white-space: nowrap;

  &:hover {
    background: #e8e8e8;
    color: #333;
  }

  &.active {
    background: #d4e8ff;
    color: #2080f0;
  }

  .sidebar-label {
    font-size: 13px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

/* 折叠态：图标居中，去掉文字留白和间距 */
.sidebar-menu {
  &.collapsed .sidebar-item {
    justify-content: center;
    padding: 0;
    margin: 0 2px 4px;
    gap: 0;
  }
}
</style>

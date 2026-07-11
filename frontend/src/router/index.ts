import {createRouter, createWebHashHistory} from 'vue-router'
import MainLayout from '../views/MainLayout.vue'
import NovelEditor from '../views/NovelEditor.vue'

/**
 * 使用 Hash 模式路由（WebView2 需要 hash 路由避免刷新问题）
 * 主界面：小说列表（默认）和设置
 * 编辑器：独立的小说编辑页面（在新窗口中使用时提供独立路由）
 */
const routes = [
  {
    path: '/',
    component: MainLayout,
    children: [
      {
        path: '',
        redirect: '/novels',
      },
      {
        path: 'novels',
        name: 'NovelList',
        component: () => import('../views/NovelList.vue'),
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('../views/SettingsPage.vue'),
      },
    ],
  },
  {
    path: '/editor/:novelId',
    name: 'NovelEditor',
    component: NovelEditor,
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import naive from 'naive-ui'
import router from './router'
import App from './App.vue'
import { registerGlobalComponents } from './utils/registerComponents'
import './assets/global.css'

const app = createApp(App)

// 安装全局状态管理
app.use(createPinia())

// 安装 Vue Router（Hash 模式，兼容 WebView2）
app.use(router)

// 安装 NaiveUI（全局注册所有组件，确保模板中可直接使用 n-* 组件）
app.use(naive)

// 全局注册异步弹框组件，按需加载免 import
registerGlobalComponents(app)

app.mount('#app')

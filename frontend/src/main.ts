import { createApp } from 'vue'
import { createPinia } from 'pinia'
import naive from 'naive-ui'
import router from './router'
import App from './App.vue'

const app = createApp(App)

// 安装全局状态管理
app.use(createPinia())

// 安装 Vue Router（Hash 模式，兼容 WebView2）
app.use(router)

// 安装 NaiveUI（全局注册所有组件，确保模板中可直接使用 n-* 组件）
app.use(naive)

app.mount('#app')

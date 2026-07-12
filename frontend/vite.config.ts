import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  // 强制预构建 vis 依赖，确保 ESM 模块在开发模式下正确加载
  optimizeDeps: {
    include: ['vis-network', 'vis-data'],
  },
  // 开发模式下代理 API 请求到 Go 后端（固定端口 18520）
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:18520',
        changeOrigin: true,
      },
    },
  },
  // 构建输出到 dist 目录，供 Go embed 使用
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})

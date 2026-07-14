import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  // @ 路径别名（与 tsconfig.json paths 保持一致）
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
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

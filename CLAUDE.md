## 项目概览
AI 小说编辑器 — Go + WebView2 + Vue3/NaiveUI 桌面应用。双击 `dida.exe` 即可运行。

## 目录结构
- `backend/` — Go 后端源码（HTTP API + WebView2）
- `frontend/` — Vue3 + NaiveUI + Vite 前端
- `build/` — 构建脚本
- `dida.exe` — 构建产物（已忽略）

## 构建方式
```bash
# 生产构建（需要 pnpm + Go）
build\build.bat           # Windows
bash build.sh             # Git Bash

# 开发模式
cd frontend && pnpm dev   # 终端1：Vite 开发服务器
cd backend && go run .    # 终端2：Go 后端（需要先构建前端）
```

## 技术栈
- **桌面层**: Go + WebView2 (`github.com/jchv/go-webview2`)
- **前端**: Vue 3 + NaiveUI + Vite + TypeScript + Pinia
- **后端**: Go HTTP 服务（内置文件存储）
- **AI**: DeepSeek API 兼容接口
- **包管理**: pnpm
- **端口**: 18520（Go 后端 API + 静态文件）/ 5173（Vite 开发服务器）

## 核心命令
- `go run .` — 启动开发版后端
- `go build -tags production -o dida.exe .` — 构建生产版
- `pnpm dev` — 启动 Vite 开发服务器
- `pnpm build` — 构建前端

## 开发须知
- 前端 Vite 开发模式通过 `vite.config.ts` 的 proxy 将 `/api` 代理到 Go 后端 18520 端口
- 生产模式下 Go 内嵌前端静态文件，单端口提供全部服务
- 不要运行 `pnpm run dev` 或 `vite` 作为后台进程
- 构建前端前需要 `pnpm install`
- 生产构建 `dida.exe` 在项目根目录，已加入 `.gitignore`

## Development Rules

- NEVER run development servers (e.g., `pnpm run dev`, `vite`) automatically.
- Do not run any long-running or non-terminating background processes.
- Only run build (`pnpm run build`) or linting/testing commands if needed to verify code.

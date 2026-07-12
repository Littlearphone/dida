## 项目概览
AI 小说编辑器 — Go + WebView2 + Vue3/NaiveUI 桌面应用。双击 `dida.exe` 即可运行。

## 目录结构
- `backend/` — Go 后端源码（HTTP API + WebView2）
  - `icon.ico` — 程序图标（通过 `//go:embed` 嵌入二进制）
- `frontend/` — Vue3 + NaiveUI + Vite 前端
- `build/` — 构建脚本 + 图标源文件
- `dida.exe` — 构建产物（已忽略）

## 构建方式
```bash
# 生产构建（需要 pnpm + Go）
build\build.bat           # Windows
bash build/build.sh       # Git Bash

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

## 编辑器操作注意事项

### Undo / Redo
- 使用 `editor.commands.undo()` + `editor.view.focus()` 模式，**不**使用 `chain().focus().undo()` 链式调用
- `chain().focus()` 会产生多余的 focus 事务干扰 undo 历史追踪，快速连续操作时可能导致选区位置异常（光标跳转到文档末尾）
- undoable 状态依赖 `hasUserEdited && editor.can().undo()`，前者防止初始加载后 undo 按钮误亮

### 替换操作（replace / replaceAll）
- 所有替换操作必须使用 `view.dispatch(tr)` 直接派发单次事务，配合 `view.focus()` 恢复焦点
- `replaceCurrent()` 和 `replaceAll()` 已遵循此模式：创建单次 `state.tr` 后 `view.dispatch(tr)`，避免多余历史记录
- **禁止**使用 `editor.chain().focus().command(...)` 模式进行替换

### 章节切换前的保存
- ChapterSidebar 切换章节时通过 `@saveBeforeSwitch` 事件通知父组件执行 `doSave()`
- 确保切换前有未保存内容时自动触发保存，避免数据丢失

## AI 编辑对话框关键逻辑
- `AIEditDialog.vue` — 润色/扩写共用组件，通过 `mode` prop 区分
- **选中检测**: `checkSelection()` 基于 ProseMirror 选区状态（不受 DOM 焦点影响）
- **状态重置**: `watch(show)` 的 else 分支 + `closeDialog()` 双重兜底，确保 Escape 等非按钮关闭方式也重置状态
- **结果编辑**: 结果展示区使用 `<n-input type="textarea" v-model:value="editResult">`，可直接修改 AI 输出后再替换
- **选中清除**: 选中预览区提供 X 关闭按钮（`clearSelection()`），可取消选中改用整章内容

## 组件样式规范

### n-button 自定义样式
- 当需要高度自定义的 `<n-button>` 样式（如等宽填充按钮），使用 NaiveUI 的 CSS 变量覆盖，而非原生 `<button>`：
  ```css
  .custom-btn {
    --n-text-color: #555 !important;
    --n-text-color-hover: #333 !important;
  }
  ```
- 对于文本渐显动画（hover 时 max-width 过渡），使用 `:deep(.n-button__content > span:last-child)` 选择器定位按钮文本
- **CSS 变量优先**: 自定义 NaiveUI 组件样式时优先使用 `--n-*` CSS 变量覆盖，其次使用 `:deep()` 穿透，避免直接覆盖全局类名

## 开发规则

- 永远不要自动启动开发服务器（如 `pnpm run dev`、`vite`）
- 不要运行任何长驻或不会终止的后台进程
- 只运行构建（`pnpm run build`）或 lint/test 命令来验证代码
- **UI 组件优先级**: 用户没有明确要求实现方式时，总是优先使用 NaiveUI 内置组件 → 没有合适组件时寻找合适的开源库（如 vicons 图标库） → 都没有再用原生 HTML 实现
- **组件拆分**: 每次增加新功能后，对过大的组件自动进行拆分，保持单个组件职责单一、可维护

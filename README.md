# AI 小说编辑器 · Dida

[![Build & Release](https://github.com/Lenovo/dida/actions/workflows/release.yml/badge.svg)](https://github.com/Lenovo/dida/actions/workflows/release.yml)

一个基于 **Go + WebView2 + Vue3/NaiveUI** 的桌面端 AI 辅助小说编辑应用。Windows 平台双击 `dida.exe` 即可运行。

---

## 功能概览

### 📚 小说管理
- 封面网格展示，支持创建和导入小说（TXT/MD 格式）
- 导入时自动 AI 拆分章节、提取大纲/角色/事件/人物关系
- 批量导入支持，自动识别章节结构

### ✍️ 富文本编辑器（Tiptap）
- 基于 ProseMirror 的富文本编辑，支持字体、字号、行距、加粗、斜体
- 选中文本复制、正文搜索替换、空行格式化
- 自动保存（可配置间隔，支持关闭），实时字数统计
- 撤销/重做（`editor.commands.undo()` 模式，避免链式调用干扰历史追踪）

### 🤖 AI 智能写作辅助
搭载 **DeepSeek API 兼容接口**，提供以下 AI 功能：

| 功能 | 说明 | 流式输出 |
|------|------|----------|
| **续写** | 基于大纲和前文续接内容，支持插入末尾或新建章节 | ✅ SSE |
| **润色** | 选中内容或整章润色，左右对比查看，结果可编辑后替换原文 | ✅ SSE |
| **扩写** | 选中内容或整章扩写，丰富细节描写，结果可自由编辑后使用 | ✅ SSE |
| **章节拆分** | 导入时自动识别章节边界并拆分 | ❌ |
| **内容提取** | 提取大纲、主要角色、人物关系、事件时间线（支持增量更新） | ✅ SSE |
- 选中内容预览弹框，可关闭选中切换为整章操作
- 所有 AI 操作结果均支持手动编辑后再替换

### ⚙️ 系统设置
- 小说保存路径配置（默认 `Documents/DidaNovels/`）
- AI 接口地址、模型名称、API Key 配置
- 自动保存开关和间隔（毫秒）
- 默认字体大小和行距

---

## AI 开发声明

> **更新日期**: 2026-07-22

### 使用目的
本应用集成 AI 能力纯粹用于**辅助文学创作**，包括内容续写、润色、扩写、章节拆分和信息提取。AI 的输出始终由用户审查、编辑并决定是否采用。

### 使用的 AI 模型
- **默认模型**: DeepSeek Chat（`deepseek-chat`，通过 DeepSeek API）
- **兼容性**: 支持任何兼容 OpenAI Chat Completions API 格式的接口（可自行配置 endpoint 和模型名称）
- **不支持**: 本地或离线模型（当前必须通过网络请求调用远程 API）

### 数据透明

| 数据类别 | 是否发送到 AI 服务 | 说明 |
|----------|-------------------|------|
| 章节正文内容 | **是**（AI 操作时） | 续写/润色/扩写时发送当前章节内容；拆分时发送完整正文 |
| 小说大纲 | **是**（AI 操作时） | 作为上下文提供给 AI 以保证创作一致性 |
| 角色/关系/事件 | **是**（AI 操作时） | 提供给 AI 以维持作品设定连贯性 |
| 用户设置的提示要求 | **是**（AI 操作时） | 用户在对话框中填写的自定义要求 |
| 小说元数据（标题/作者） | **否** | 不发送给 AI 服务 |
| 应用设置/密钥 | **否** | API Key 仅用于鉴权，不出现在提示词中 |
| 用户文件系统路径 | **否** | 本地存储路径不发送给 AI 服务 |

### 隐私说明
- 发送到 AI 服务的数据**受 DeepSeek 或您配置的第三方 API 服务商的隐私政策约束**
- 建议**不要**在使用 AI 功能的小说正文中包含个人敏感信息
- 应用默认使用 DeepSeek 官方 API（`https://api.deepseek.com`），您可以自行更换为私有部署或其他兼容服务
- 所有 AI 请求均通过本地 Go 后端发起，前端不直接调用第三方 API

### 可选性
- **AI 功能完全可选**。不配置 API Key 时，应用可正常使用除 AI 功能外的所有编辑和管理功能
- 每个 AI 操作均需要用户**主动触发**（点击按钮），不会在后台静默调用 AI
- 用户可以在设置页面随时更改或清除 API Key

### 依赖性
- AI 功能依赖网络连接和第三方 API 服务的可用性
- 应用本身的编辑、存储、管理功能完全离线可用，仅 AI 功能需要联网

---

## 技术架构

```
┌──────────────────────────────────────┐
│          WebView2 窗口 (Edge)        │
│  ┌──────────────────────────────┐   │
│  │     Vue3 + NaiveUI SPA       │   │
│  │    (Hash Router + Pinia)     │   │
│  │  ┌────┬────┬────┬────────┐  │   │
│  │  │小说│章节│ AI │ 设置   │  │   │
│  │  └────┴────┴────┴────────┘  │   │
│  └──────────┬───────────────────┘   │
└─────────────┼───────────────────────┘
              │ HTTP (127.0.0.1:18520)
┌─────────────┴───────────────────────┐
│      Go 后端 (net/http + CORS)      │
│  ┌──────────────────────────────┐   │
│  │  handlers/                   │   │
│  │  ┌──────┬───────┬────┬────┐ │   │
│  │  │novel │chapter│ ai │setting│ │   │
│  │  └──────┴───────┴────┴────┘ │   │
│  ├──────────────────────────────┤   │
│  │  ai/ (DeepSeek API 客户端)   │   │
│  │  ┌──────┬─────┬──────┬────┐ │   │
│  │  │client│edits│extract│...│ │   │
│  │  └──────┴─────┴──────┴────┘ │   │
│  ├──────────────────────────────┤   │
│  │  store/ (JSON 文件存储)      │   │
│  │  ┌────────────┬───────────┐  │   │
│  │  │novel_store │settings   │  │   │
│  │  └────────────┴───────────┘  │   │
│  └──────────────────────────────┘   │
└─────────────────────────────────────┘
```

---

## 快速开始

### 下载运行
从 Release 页面下载 `dida.exe`，双击即可运行。

> **系统要求**: Windows 10 (v1803+) 或 Windows 11（已预装 WebView2 Runtime）

### 从源码构建

#### 前置要求
| 工具 | 版本 | 用途 |
|------|------|------|
| Go | 1.22+ | 编译后端 + 嵌入前端资源 |
| Node.js | 22+ | 构建前端 |
| pnpm | ≥8 | 前端包管理（`npm install -g pnpm`） |

#### 生产构建

**Windows (CMD)**
```bash
build\build.bat
# 输出: dida.exe（项目根目录）
```

**Linux/macOS (Git Bash)**
```bash
bash build/build.sh
# 输出: dida.exe
```

> 生产构建将前端产物嵌入 Go 二进制，单文件分发，无需依赖 Web 服务器。

#### 开发模式

```bash
# 终端 1: 启动前端开发服务器
cd frontend
pnpm install
pnpm dev            # Vite 开发服务器 → http://localhost:5173

# 终端 2: 启动 Go 后端
cd backend
go run -ldflags="-X main.devMode=true" .
# 后端 → http://localhost:18520
```

> 开发模式下，Vite 通过 `vite.config.ts` 的 proxy 将 `/api` 请求代理到 Go 后端 18520 端口。

---

## 配置 AI 接口

在设置页面或编辑器底部配置：

1. **接口地址**: `https://api.deepseek.com`（默认，可替换为私有部署）
2. **模型名称**: `deepseek-chat`（默认）
3. **API Key**: 你的 API 密钥

未配置 AI 时，应用仍可正常使用除 AI 功能外的所有编辑功能。

---

## 数据存储

| 内容 | 路径 | 格式 |
|------|------|------|
| 应用配置 | `%APPDATA%/.dida/settings.json` | JSON |
| 小说数据 | `Documents/DidaNovels/`（可配置） | 每本小说一个文件夹 |
| 小说元信息 | `<novel>/meta.json` | JSON |
| 章节内容 | `<novel>/chapters/<id>.json` | JSON |
| 封面图片 | `<novel>/cover.*` | 原始图片 |

---

## 项目结构

```
dida/
├── backend/                         # Go 后端
│   ├── main.go                      # 入口 + HTTP 服务器 + 信号处理
│   ├── static_prod.go               # 生产模式 embeds 前端静态文件
│   ├── static_dev.go                # 开发模式 stub（指向 Vite 地址）
│   ├── webview_windows.go           # WebView2 窗口创建与管理
│   ├── webview_stub.go              # 非 Windows 平台 stub
│   ├── handlers/                    # HTTP API 处理器
│   │   ├── novel.go                 # 小说 CRUD + 封面上传
│   │   ├── chapter.go               # 章节 CRUD + 自动保存
│   │   ├── ai.go                    # AI 功能统一入口（SSE 流式/非流式）
│   │   └── settings.go              # 系统设置读写
│   ├── models/
│   │   └── models.go                # 共享数据模型（Novel/Chapter/Character...）
│   ├── store/
│   │   ├── novel_store.go           # 小说文件存储引擎
│   │   └── settings_store.go        # 设置文件存储引擎
│   ├── ai/                          # DeepSeek API 客户端
│   │   ├── client.go                # HTTP 客户端 + 请求构建
│   │   ├── edits.go                 # 续写/润色/扩写
│   │   ├── extract.go               # 非流式信息提取
│   │   └── extract_stream.go        # 流式（SSE）信息提取
│   ├── icon.ico                     # 程序图标（go:embed）
│   ├── go.mod / go.sum
│   └── icon.ico
├── frontend/                        # Vue3 + TypeScript 前端
│   ├── index.html                   # 入口 HTML
│   ├── vite.config.ts               # Vite 配置（+ 开发代理）
│   ├── tsconfig.json
│   └── src/
│       ├── main.ts                  # 应用挂载 + 全局组件注册
│       ├── App.vue                  # 根组件（路由视图 + 标题栏）
│       ├── router/index.ts          # Hash 路由配置
│       ├── stores/                  # Pinia 状态管理
│       │   ├── novelStore.ts        # 小说/章节状态
│       │   └── settingStore.ts      # 应用设置状态
│       ├── api/                     # HTTP API 调用封装
│       │   ├── ai.ts                # AI 续写/润色/扩写/拆分/提取
│       │   ├── chapter.ts           # 章节 API
│       │   ├── novel.ts             # 小说 API
│       │   └── settings.ts          # 设置 API
│       ├── views/                   # 路由视图
│       │   ├── NovelList.vue        # 小说封面网格页
│       │   ├── NovelEditor.vue      # 编辑器主页面
│       │   ├── MainLayout.vue       # 布局壳（侧边栏 + 内容区）
│       │   └── SettingsPage.vue     # 系统设置页
│       ├── components/              # 可复用组件
│       │   ├── NovelCard.vue        # 小说封面卡片
│       │   ├── editor/              # 编辑器相关子组件
│       │   └── novel/               # 小说管理相关子组件
│       ├── composables/             # 组合式逻辑（Vue composables）
│       │   ├── useAIStream.ts       # SSE 流式消费逻辑
│       │   ├── useAIDialogs.ts      # AI 弹框状态管理
│       │   ├── useAutoSave.ts       # 自动保存逻辑
│       │   ├── useChapterDrag.ts    # 章节拖拽排序
│       │   ├── useChapterSplit.ts   # 导入章节拆分
│       │   ├── useCharacterEdit.ts  # 角色编辑
│       │   ├── useEditorAppearance.ts
│       │   ├── useExport.ts         # 导出功能
│       │   ├── useGraphNetwork.ts   # 角色关系图
│       │   └── useSearch.ts         # 正文搜索
│       ├── types/index.ts           # TypeScript 类型定义
│       ├── utils/
│       │   └── registerComponents.ts  # 全局异步组件注册
│       └── assets/                  # 静态资源（CSS/图片）
├── build/
│   ├── build.bat                    # Windows 生产构建脚本
│   └── build.sh                     # Git Bash 生产构建脚本
├── .github/
│   └── workflows/
│       └── release.yml              # GitHub Actions 自动构建与发布
├── README.md
├── package.json                     # 项目级（仅 Git Hooks 等）
├── CLAUDE.md                        # AI 编程助手项目说明
└── .gitignore
```

---

## 端口说明

| 端口 | 用途 | 备注 |
|------|------|------|
| `18520` | Go 后端 API + 生产静态文件 | 固定端口，被占用则随机回退 |
| `5173` | Vite 开发服务器 | 仅开发模式使用 |

---

## 开发约定

### 编辑器状态管理
- **撤销/重做**: 使用 `editor.commands.undo()` + `editor.view.focus()` 模式，禁用 `chain().focus().undo()` 链式调用避免干扰历史追踪
- **替换操作**: 使用 `view.dispatch(tr)` 直接派发单次事务，禁用 `editor.chain().focus().command(...)` 链条

### 组件约束
- 脚本超过 200 行 → 提取 composable
- `.vue` 文件超过 300 行 → 拆分子组件
- 所有弹框组件必须注册为全局异步组件（`registerComponents.ts` 中注册）

### 样式
- `<style lang="scss" scoped>` 优先
- NaiveUI 组件使用 `--n-*` CSS 变量自定义样式
- `&` 嵌套语法用于有父子/状态关系的选择器

---

## 自动发布

项目使用 **GitHub Actions** 自动构建和发布。详见 [.github/workflows/release.yml](.github/workflows/release.yml)。

### 触发方式

| 事件 | 行为 |
|------|------|
| 推送 `v*` 标签 | 构建 `dida.exe` + 创建 GitHub Release |
| 推送 `main`/`master` | 仅构建验证，不发布 |
| Pull Request | 构建验证 |

### 发布新版本

```bash
# 1. 打标签并推送
git tag v1.0.0
git push origin v1.0.0

# 2. Actions 自动构建并发布到
#    https://github.com/Lenovo/dida/releases/tag/v1.0.0
```

### 手动下载产物

非标签推送的构建产物可在 Actions 运行页面的 **Artifacts** 中下载。

---

## 许可

MIT

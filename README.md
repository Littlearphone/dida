# AI 小说编辑器

一个基于 Go + WebView2 + Vue3/NaiveUI 的桌面端 AI 小说编辑应用，支持 Windows 平台双击运行。

## 功能概览

### 📚 小说管理
- 封面网格展示，支持创建和导入小说
- 导入时自动 AI 拆分章节、提取大纲/角色/事件
- 支持批量导入（TXT/MD 格式）

### ✍️ 富文本编辑
- 可编辑正文，支持字体、字号、行距、加粗、斜体设置
- 选中文本复制、正文搜索、空行格式化
- 自动保存（可配置间隔，支持关闭）
- 实时字数统计

### 🤖 AI 智能写作（DeepSeek）
- **AI 续写** — 设置要求，AI 续接内容，支持插入末尾或新建章节
- **AI 润色** — 选中内容或整章润色，左右对比，结果可编辑后替换原文
- **AI 扩写** — 选中内容或整章扩写，丰富细节描写，结果可自由编辑后使用
- 选中内容预览 — 弹框内显示选中文本，可关闭选中切换为整章操作
- 智能导入拆分 — 导入时自动识别章节结构
- 内容提取 — 提取大纲、主要角色、人物关系、事件时间线

### ⚙️ 系统设置
- 小说保存路径配置
- AI 接口地址和密钥配置（DeepSeek 兼容）
- 自动保存开关和间隔
- 默认字体和行距

## 技术架构

```
┌─────────────────────────────┐
│   WebView2 窗口 (Edge)      │
│  ┌───────────────────────┐  │
│  │   Vue3 + NaiveUI SPA  │  │
│  │   (Hash Router)       │  │
│  └──────────┬────────────┘  │
└─────────────┼───────────────┘
              │ HTTP (127.0.0.1:18520)
┌─────────────┴───────────────┐
│   Go 后端 (net/http)        │
│  ┌───────────────────────┐  │
│  │  API 路由              │  │
│  │  ┌─────┬──────┬────┐  │  │
│  │  │小说 │章节  │ AI │  │  │
│  │  └─────┴──────┴────┘  │  │
│  └───────────────────────┘  │
│  ┌───────────────────────┐  │
│  │ 文件存储 (JSON)        │  │
│  └───────────────────────┘  │
└─────────────────────────────┘
```

## 快速开始

### 下载运行
从 Release 页面下载 `dida.exe`，双击即可运行。

> **系统要求**: Windows 10 v1803+ 或 Windows 11（已预装 WebView2 Runtime）

### 从源码构建

#### 前置要求
- Go 1.22+
- Node.js 22+
- pnpm (`npm install -g pnpm`)

#### 构建步骤

**Windows (CMD)**
```bash
build\build.bat
# 输出: dida.exe
```

**Linux/macOS (Git Bash)**
```bash
bash build/build.sh
# 输出: dida.exe
```

#### 开发模式

```bash
# 终端 1: 启动前端开发服务器
cd frontend
pnpm install
pnpm dev            # 启动于 http://localhost:5173

# 终端 2: 启动 Go 后端
cd backend
go run -ldflags="-X main.devMode=true" .
# 后端启动于 http://localhost:18520
```

> 开发模式下，Vite 会将 `/api` 请求代理到 Go 后端。

## 配置 AI 接口

应用支持 DeepSeek API 兼容接口。在设置页面或编辑器底部可配置：

1. **接口地址**: `https://api.deepseek.com`（默认）
2. **模型名称**: `deepseek-chat`（默认）
3. **API Key**: 你的 DeepSeek API 密钥

未配置 AI 时，应用仍可正常使用除 AI 功能外的所有编辑功能。

## 数据存储

- **设置文件**: `%APPDATA%/.dida/settings.json`
- **小说文件**: 默认 `Documents/DidaNovels/`（可在设置中修改）
- 每本小说一个文件夹，包含 `meta.json` 和 `chapters/` 目录

## 项目结构

```
dida/
├── backend/                     # Go 后端
│   ├── main.go                  # 入口 + HTTP 服务
│   ├── static_prod.go           # 生产静态文件嵌入
│   ├── static_dev.go            # 开发模式桩
│   ├── webview_windows.go       # WebView2 窗口
│   ├── webview_stub.go          # 非 Windows 桩
│   ├── handlers/                # HTTP 处理器
│   │   ├── novel.go             # 小说 CRUD
│   │   ├── chapter.go           # 章节 + 自动保存
│   │   ├── ai.go                # AI 功能接口
│   │   └── settings.go          # 系统设置
│   ├── models/models.go         # 数据模型
│   ├── store/                   # 文件存储
│   │   ├── novel_store.go
│   │   └── settings_store.go
│   └── ai/deepseek.go           # DeepSeek 客户端
├── frontend/                    # Vue3 前端
│   ├── src/
│   │   ├── api/                 # API 调用
│   │   ├── stores/              # Pinia 状态
│   │   ├── views/               # 页面组件
│   │   ├── components/          # 通用组件
│   │   ├── types/               # TS 类型
│   │   └── router/              # 路由
│   ├── index.html
│   └── vite.config.ts
├── build/
│   ├── build.bat                # Windows 构建脚本
│   └── build.sh                 # Git Bash 构建脚本
└── README.md
```

## 端口说明

| 端口 | 用途 | 备注 |
|------|------|------|
| 18520 | Go 后端 API + 生产静态文件 | 固定端口，占用则随机回退 |
| 5173 | Vite 开发服务器 | 仅开发模式使用 |

## 许可

MIT

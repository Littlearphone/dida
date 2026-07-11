#!/usr/bin/env bash
# AI 小说编辑器 - 构建脚本
# 用法: bash build.sh
set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"

echo "=== AI 小说编辑器 - 构建脚本 ==="

# 1. 构建前端
echo "[1/3] 构建前端..."
cd "$FRONTEND_DIR"
pnpm install
pnpm build
echo "前端构建完成"

# 2. 复制前端产物到后端嵌入目录
echo "[2/3] 复制前端静态文件..."
mkdir -p "$BACKEND_DIR/frontend/dist"
cp -r "$FRONTEND_DIR/dist/"* "$BACKEND_DIR/frontend/dist/"
echo "静态文件复制完成"

# 3. 构建 Go 生产版本
echo "[3/3] 构建可执行文件..."
cd "$BACKEND_DIR"
go build -tags production -o "$ROOT_DIR/dida.exe" -ldflags="-H=windowsgui" .
echo "Go 构建完成"

# 清理临时文件
rm -rf "$BACKEND_DIR/frontend"

echo ""
echo "=== ✅ 构建成功！输出: $ROOT_DIR/dida.exe ==="
echo "双击 dida.exe 即可运行"

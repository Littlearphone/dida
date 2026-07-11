@echo off
chcp 65001 >nul
title AI 小说编辑器 - 构建
echo ========================================
echo   AI 小说编辑器 - 构建脚本
echo ========================================
echo.

set ROOT_DIR=%~dp0..
set BACKEND_DIR=%ROOT_DIR%\backend
set FRONTEND_DIR=%ROOT_DIR%\frontend

REM 1. 构建前端
echo [1/3] 构建前端...
cd /d "%FRONTEND_DIR%"
call pnpm install
if %errorlevel% neq 0 (
    echo pnpm install 失败!
    pause
    exit /b 1
)

call pnpm build
if %errorlevel% neq 0 (
    echo pnpm build 失败!
    pause
    exit /b 1
)
echo 前端构建完成

REM 2. 复制前端产物到后端嵌入目录
echo [2/3] 复制前端静态文件...
if exist "%BACKEND_DIR%\frontend" rmdir /s /q "%BACKEND_DIR%\frontend"
mkdir "%BACKEND_DIR%\frontend\dist"
xcopy /e /i /q "%FRONTEND_DIR%\dist\*" "%BACKEND_DIR%\frontend\dist\"
echo 静态文件复制完成

REM 3. 构建 Go 生产版本
echo [3/3] 构建可执行文件...
cd /d "%BACKEND_DIR%"
go build -tags production -o "%ROOT_DIR%\dida.exe" -ldflags="-H=windowsgui" .
if %errorlevel% neq 0 (
    echo Go 构建失败!
    pause
    exit /b 1
)
echo Go 构建完成

REM 清理临时文件
if exist "%BACKEND_DIR%\frontend" rmdir /s /q "%BACKEND_DIR%\frontend"

echo.
echo ========================================
echo   ✅ 构建成功！输出: %ROOT_DIR%\dida.exe
echo   双击 dida.exe 即可运行
echo ========================================
echo.
echo 提示: 运行时防火墙可能会询问网络访问，请允许。
pause

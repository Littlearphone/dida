//go:build windows

package main

import (
	"log"

	"github.com/jchv/go-webview2"
)

// createWebViewWindow 在 Windows 上创建 WebView2 窗口
// 使用 Edge WebView2 控件渲染前端界面，提供原生桌面体验
// jchv/go-webview2 无需 CGo，直接通过 COM 调用 WebView2 API
func createWebViewWindow(frontendURL string) {
	// 创建 WebView2 窗口，debug=true 启用 DevTools 方便调试
	w := webview2.New(true)
	defer w.Destroy()

	// 设置窗口标题和默认尺寸
	w.SetTitle("AI 小说编辑器")
	w.SetSize(1280, 800, webview2.HintNone)

	// 导航到前端地址
	w.Navigate(frontendURL)

	log.Println("WebView2 窗口已创建，导航到:", frontendURL)
	w.Run() // 阻塞运行，直到用户关闭窗口
}

//go:build windows

package main

import (
	"log"
	"syscall"

	"github.com/jchv/go-webview2"
)

// user32 懒加载，供窗口最大化使用
var user32 = syscall.NewLazyDLL("user32.dll")

// init 在进程启动时设置 DPI 感知，确保高 DPI 屏幕下 WebView2 内容清晰渲染
func init() {
	// Shcore.dll 的 SetProcessDpiAwareness 支持三种级别：
	//   PROCESS_DPI_UNAWARE          = 0 — 系统默认，高 DPI 下位图拉伸，模糊
	//   PROCESS_SYSTEM_DPI_AWARE     = 1 — 感知系统 DPI，窗口跨不同 DPI 显示器时仍会模糊
	//   PROCESS_PER_MONITOR_DPI_AWARE = 2 — 逐显示器感知，推荐值
	const perMonitorDPI = uintptr(2)

	shcore := syscall.NewLazyDLL("Shcore.dll")
	setAwareness := shcore.NewProc("SetProcessDpiAwareness")
	if err := setAwareness.Find(); err != nil {
		// 回退：SetProcessDPIAware (Vista+ 最低级别 DPI 感知)
		setDPIAware := user32.NewProc("SetProcessDPIAware")
		setDPIAware.Call()
		return
	}
	setAwareness.Call(perMonitorDPI)
}

// createWebViewWindow 在 Windows 上创建 WebView2 窗口
// 使用 Edge WebView2 控件渲染前端界面，提供原生桌面体验
// jchv/go-webview2 无需 CGo，直接通过 COM 调用 WebView2 API
func createWebViewWindow(frontendURL string) {
	// 创建 WebView2 窗口，debug=true 启用 DevTools 方便调试
	w := webview2.New(true)
	defer w.Destroy()

	// 设置窗口标题和默认最小尺寸（防止最大化后缩得太小）
	w.SetTitle("AI 小说编辑器")
	w.SetSize(960, 640, webview2.HintMin)

	// 导航到前端地址
	w.Navigate(frontendURL)

	// 获取原生窗口句柄并最大化显示
	showWindow := user32.NewProc("ShowWindow")
	showWindow.Call(uintptr(w.Window()), 3) // SW_MAXIMIZE = 3

	log.Println("WebView2 窗口已创建（最大化），导航到:", frontendURL)
	w.Run() // 阻塞运行，直到用户关闭窗口
}

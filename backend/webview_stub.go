//go:build !windows

package main

import (
	"fmt"
	"log"
)

// createWebViewWindow 非 Windows 平台的桩实现
// 回退到控制台提示，用户可手动打开浏览器
func createWebViewWindow(url string) {
	log.Printf("WebView2 仅在 Windows 上可用，当前操作系统不支持。")
	fmt.Printf("\n========================================\n")
	fmt.Printf("  请手动打开浏览器访问: %s\n", url)
	fmt.Printf("========================================\n\n")
	fmt.Println("按 Enter 键退出...")
	fmt.Scanln()
}

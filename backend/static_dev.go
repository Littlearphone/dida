//go:build !production

package main

import (
	"net/http"
)

// registerStaticRoutes 开发模式下不注册静态文件路由
// 前端由 Vite 开发服务器 (localhost:5173) 提供
func registerStaticRoutes(mux *http.ServeMux) {
	// 开发模式下不提供静态文件服务
	// 前端由独立的 Vite dev server 托管
}

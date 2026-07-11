//go:build production

package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

// 嵌入前端构建产物
//
//go:embed frontend/dist
var embeddedStatic embed.FS

// init 在生产模式下注册静态文件路由
func registerStaticRoutes(mux *http.ServeMux) {
	// 从嵌入的 FS 中提取前端 dist 子目录
	staticFS, err := fs.Sub(embeddedStatic, "frontend/dist")
	if err != nil {
		log.Fatalf("读取内嵌静态文件失败: %v", err)
	}

	// 使用 http.FileServer 提供静态文件服务
	fileServer := http.FileServer(http.FS(staticFS))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// 只处理非 API 的请求
		fileServer.ServeHTTP(w, r)
	})

	log.Println("生产模式：静态文件已内嵌")
}

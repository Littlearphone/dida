package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"dida/handlers"
	"dida/store"
)

// devMode 编译时通过 -ldflags 注入，开发模式为 "true"
// 开发模式下：前端指向 Vite 开发服务器 (localhost:5173)
// 生产模式下：内嵌静态文件并由 Go HTTP 服务器提供
var devMode = "false"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("AI 小说编辑器启动中...")

	// === 初始化存储层 ===
	settingsStore, err := store.NewSettingsStore()
	if err != nil {
		log.Fatalf("初始化设置存储失败: %v", err)
	}
	log.Println("设置存储初始化完成")

	settings := settingsStore.Get()

	// 确保小说存储目录存在
	if err := os.MkdirAll(settings.NovelPath, 0755); err != nil {
		log.Fatalf("创建小说目录失败: %v", err)
	}

	novelStore := store.NewNovelStore(settings.NovelPath)
	if err := novelStore.LoadAll(); err != nil {
		log.Printf("加载已有小说时出现警告: %v", err)
	}
	log.Printf("已加载小说数据")

	// === 初始化 HTTP 路由 ===
	mux := http.NewServeMux()
	registerRoutes(mux, settingsStore, novelStore)

	// 启动 HTTP 服务器 — 监听端口 18520
	// 开发模式下要求必须使用 18520（Vite proxy 硬编码指向此端口）
	// 生产模式下若被占用则回退到随机端口
	listener, err := net.Listen("tcp", "localhost:18520")
	if err != nil {
		if devMode == "true" {
			log.Fatalf("端口 18520 被占用！请关闭占用该端口的程序后重试。\n" +
				"  Vite proxy 固定指向 18520，开发模式不可使用随机端口。\n" +
				"  执行: netstat -ano | findstr 18520  查看占用进程")
		}
		// 生产模式：回退到随机端口
		listener, err = net.Listen("tcp", "localhost:0")
		if err != nil {
			log.Fatalf("监听端口失败: %v", err)
		}
	}
	port := listener.Addr().(*net.TCPAddr).Port

	corsHandler := corsMiddleware(mux)

	server := &http.Server{Handler: corsHandler}
	go func() {
		log.Printf("HTTP 服务启动于 http://localhost:%d", port)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP 服务错误: %v", err)
		}
	}()

	// === 确定前端地址 ===
	// 开发模式：前端由 Vite 提供，需要手动启动 pnpm dev
	// 生产模式：内置静态文件由 Go 提供，直接使用 API 端口
	frontendURL := fmt.Sprintf("http://localhost:%d", port)
	if devMode == "true" {
		frontendURL = "http://localhost:5173"
		log.Printf("[开发模式] 前端地址: %s", frontendURL)
		log.Printf("[开发模式] 请确保已在 frontend/ 目录执行: pnpm dev")
	} else {
		log.Printf("[生产模式] 内嵌静态文件，端口: %d", port)
	}

	// === 打开 WebView2 窗口 ===
	openWebView(frontendURL)

	if runtime.GOOS == "windows" {
		// WebView2 窗口已关闭，直接退出
		log.Println("窗口已关闭，程序退出")
		return
	}

	// === 等待退出信号（非 Windows 下收到 Ctrl+C 时优雅退出） ===
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("正在关闭服务...")
}

// registerRoutes 注册所有 API 路由
func registerRoutes(mux *http.ServeMux, ss *store.SettingsStore, ns *store.NovelStore) {
	novelHandler := handlers.NewNovelHandler(ns)
	chapterHandler := handlers.NewChapterHandler(ns)
	aiHandler := handlers.NewAIHandler(ss)
	settingsHandler := handlers.NewSettingsHandler(ss)

	// 小说 API
	mux.HandleFunc("GET /api/novels", novelHandler.HandleList)
	mux.HandleFunc("POST /api/novels", novelHandler.HandleCreate)
	mux.HandleFunc("GET /api/novels/{id}", novelHandler.HandleGet)
	mux.HandleFunc("DELETE /api/novels/{id}", novelHandler.HandleDelete)
	mux.HandleFunc("PUT /api/novels/{id}", novelHandler.HandleUpdate)
	mux.HandleFunc("POST /api/novels/import", novelHandler.HandleImport)
	mux.HandleFunc("GET /api/novels/{id}/chapters", novelHandler.HandleGetChapters)
	mux.HandleFunc("PUT /api/novels/{id}/chapters/reorder", novelHandler.HandleReorderChapters)

	// 章节 API
	mux.HandleFunc("POST /api/chapters", chapterHandler.HandleCreate)
	mux.HandleFunc("GET /api/chapters/{id}", chapterHandler.HandleGet)
	mux.HandleFunc("PUT /api/chapters/{id}", chapterHandler.HandleUpdate)
	mux.HandleFunc("DELETE /api/chapters/{id}", chapterHandler.HandleDelete)
	mux.HandleFunc("PUT /api/chapters/{id}/autosave", chapterHandler.HandleAutoSave)

	// AI API
	mux.HandleFunc("GET /api/ai/status", aiHandler.HandleCheck)
	mux.HandleFunc("POST /api/ai/split-chapters", aiHandler.HandleSplitChapters)
	mux.HandleFunc("POST /api/ai/extract-info", aiHandler.HandleExtractInfo)
	mux.HandleFunc("POST /api/ai/continue-write", aiHandler.HandleContinueWrite)
	mux.HandleFunc("POST /api/ai/polish", aiHandler.HandlePolish)
	mux.HandleFunc("POST /api/ai/expand", aiHandler.HandleExpand)

	// 设置 API
	mux.HandleFunc("GET /api/settings", settingsHandler.HandleGet)
	mux.HandleFunc("PUT /api/settings", settingsHandler.HandleUpdate)
	mux.HandleFunc("GET /api/settings/apikey", settingsHandler.HandleGetAPIKey)

	// 生产模式下：注册静态文件路由（在 static_prod.go 中定义）
	registerStaticRoutes(mux)
}

// corsMiddleware 添加 CORS 头，允许开发模式下前端跨域访问 API
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// openWebView 打开 WebView2 窗口（Windows）或提示手动打开浏览器
func openWebView(url string) {
	if runtime.GOOS != "windows" {
		log.Printf("WebView2 仅在 Windows 上可用，当前系统: %s", runtime.GOOS)
		log.Printf("请手动打开浏览器访问: %s", url)
		return
	}
	createWebViewWindow(url)
}

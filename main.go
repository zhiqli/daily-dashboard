package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"daily-dashboard/handler"
	"daily-dashboard/scheduler"
)

//go:embed templates/* static/*
var embeddedFS embed.FS

func main() {
	// --- 数据目录 ---
	dataDir := "data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if wd, _ := os.Getwd(); filepath.Base(wd) == "daily-dashboard" {
			dataDir = filepath.Join(wd, "data")
		}
	}

	// --- 加载本地名言（fallback） ---
	quotesPath := filepath.Join(dataDir, "quotes.json")
	if err := handler.LoadLocalQuotes(quotesPath); err != nil {
		log.Fatalf("[init] 加载名言失败: %v", err)
	}
	log.Printf("[init] 已加载 %d 条本地名言", handler.ListQuotesForLog())

	// --- 初始化 Todo Store（带文件持久化） ---
	store := handler.NewTodoStore(dataDir)

	// --- SSE Broker ---
	broker := handler.NewSSEBroker()
	go broker.StartBroadcastLoop(store.BroadcastChan())

	// --- 每日定时刷新 ---
	scheduler.StartDailyRefresh(store)

	// --- 路由 ---
	mux := http.NewServeMux()

	staticFS, _ := fs.Sub(embeddedFS, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		tmpl, _ := embeddedFS.ReadFile("templates/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(tmpl)
	})

	mux.HandleFunc("/api/weather", handler.WeatherHandler)
	mux.HandleFunc("/api/quote", handler.QuoteHandler)
	mux.HandleFunc("/api/todos", handler.MakeTodoHandler(store))
	mux.HandleFunc("/api/todos/", handler.MakeTodoHandler(store))
	mux.HandleFunc("/api/homework", handler.MakeHomeworkHandler(store))
	mux.HandleFunc("/api/homework/", handler.MakeHomeworkHandler(store))
	mux.HandleFunc("/api/menu", handler.MakeMenuHandler(store))
	mux.HandleFunc("/api/menu/", handler.MakeMenuHandler(store))
	mux.HandleFunc("/api/events", handler.SSEHandler(broker))
	mux.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) {
		store.DailyRefresh()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/agent/summary", handler.MakeAgentSummaryHandler(store))

	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("[server] 🚀 每日面板启动: http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("[server] 启动失败: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

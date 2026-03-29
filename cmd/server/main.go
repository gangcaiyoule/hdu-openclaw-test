package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hduhelp/hdu-openclaw/internal/bot"
	"github.com/hduhelp/hdu-openclaw/internal/chat"
	"github.com/hduhelp/hdu-openclaw/internal/config"
	"github.com/hduhelp/hdu-openclaw/internal/feishu"
	"github.com/hduhelp/hdu-openclaw/internal/llm"
	"github.com/hduhelp/hdu-openclaw/internal/reminder"
	"github.com/hduhelp/hdu-openclaw/internal/store"
)

// main 负责组装 HTTP 服务、PostgreSQL 存储、提醒流程和调度器。
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(rootCtx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer closeDB(db)

	repo := reminder.NewRepository(db)
	if err := repo.EnsureSchema(rootCtx); err != nil {
		log.Fatalf("ensure reminder schema: %v", err)
	}

	llmClient := llm.NewClient(cfg)
	memory := chat.NewMemoryStore(cfg.ChatHistoryLimit)
	chatService := chat.NewService(cfg, llmClient, memory)
	reminderParser := reminder.NewParser(cfg, llmClient)
	reminderService := reminder.NewService(cfg, repo, reminderParser)
	botService := bot.NewService(chatService, reminderService)
	feishuClient := feishu.NewClient(cfg)
	webhookHandler := feishu.NewWebhookHandler(cfg, botService, feishuClient)

	scheduler := reminder.NewScheduler(cfg, repo, feishuClient)
	go scheduler.Start(rootCtx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /webhook/feishu/event", webhookHandler.HandleProbe)
	mux.HandleFunc("POST /webhook/feishu/event", webhookHandler.HandleEvent)

	server := &http.Server{
		Addr:    cfg.AppAddr,
		Handler: mux,
	}

	go func() {
		<-rootCtx.Done()
		_ = server.Shutdown(context.Background())
	}()

	log.Printf("hdu-openclaw listening on %s", cfg.AppAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

// handleHealthz 返回最小健康检查响应，方便本地和内网穿透探活。
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// closeDB 在服务退出时关闭共享数据库连接。
func closeDB(db *sql.DB) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		log.Printf("close database: %v", err)
	}
}

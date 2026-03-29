package main

import (
	"log"
	"net/http"

	"github.com/hduhelp/hdu-openclaw/internal/chat"
	"github.com/hduhelp/hdu-openclaw/internal/config"
	"github.com/hduhelp/hdu-openclaw/internal/feishu"
	"github.com/hduhelp/hdu-openclaw/internal/llm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	llmClient := llm.NewClient(cfg)
	memory := chat.NewMemoryStore(cfg.ChatHistoryLimit)
	chatService := chat.NewService(cfg, llmClient, memory)
	feishuClient := feishu.NewClient(cfg)
	webhookHandler := feishu.NewWebhookHandler(cfg, chatService, feishuClient)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /webhook/feishu/event", webhookHandler.HandleProbe)
	mux.HandleFunc("POST /webhook/feishu/event", webhookHandler.HandleEvent)

	server := &http.Server{
		Addr:    cfg.AppAddr,
		Handler: mux,
	}

	log.Printf("hdu-openclaw listening on %s", cfg.AppAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

package chat

import (
	"context"
	"strings"

	"github.com/hduhelp/hdu-openclaw/internal/config"
	"github.com/hduhelp/hdu-openclaw/internal/llm"
)

// Service 在提醒解析未命中时负责普通聊天回复。
type Service struct {
	cfg    config.Config
	llm    *llm.Client
	memory *MemoryStore
}

// NewService 创建普通聊天服务实例。
func NewService(cfg config.Config, llmClient *llm.Client, memory *MemoryStore) *Service {
	return &Service{
		cfg:    cfg,
		llm:    llmClient,
		memory: memory,
	}
}

// Reply 生成普通助手回复，并把本轮对话写入会话内存。
func (s *Service) Reply(ctx context.Context, sessionID, userText string) (string, error) {
	userText = strings.TrimSpace(userText)
	if userText == "" {
		return "目前先支持文本消息，你可以直接和我聊天。", nil
	}

	messages := []llm.ChatMessage{{
		Role:    "system",
		Content: s.cfg.LLMSystemPrompt,
	}}

	for _, item := range s.memory.Get(sessionID) {
		messages = append(messages, llm.ChatMessage{
			Role:    item.Role,
			Content: item.Content,
		})
	}

	messages = append(messages, llm.ChatMessage{
		Role:    "user",
		Content: userText,
	})

	answer, err := s.llm.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	s.memory.Append(sessionID,
		Message{Role: "user", Content: userText},
		Message{Role: "assistant", Content: answer},
	)

	return answer, nil
}

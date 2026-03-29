package bot

import (
	"context"

	"github.com/hduhelp/hdu-openclaw/internal/reminder"
)

// MessageInput 描述处理一条入站消息所需的上下文信息。
type MessageInput struct {
	SessionID      string
	Platform       string
	PlatformUserID string
	ChatID         string
	Text           string
}

// ChatService 定义普通聊天链路需要提供的能力。
type ChatService interface {
	Reply(ctx context.Context, sessionID, userText string) (string, error)
}

// ReminderService 定义提醒链路需要提供的能力。
type ReminderService interface {
	TryCreate(ctx context.Context, req reminder.CreateRequest) (bool, string, error)
}

// Service 负责统一编排提醒创建和普通聊天两条处理链路。
type Service struct {
	chat      ChatService
	reminders ReminderService
}

// NewService 创建一个可在提醒与普通聊天之间路由的机器人服务。
func NewService(chatService ChatService, reminderService ReminderService) *Service {
	return &Service{
		chat:      chatService,
		reminders: reminderService,
	}
}

// HandleText 处理文本消息，优先尝试创建提醒，否则回落到普通聊天回复。
func (s *Service) HandleText(ctx context.Context, input MessageInput) (string, error) {
	handled, reply, err := s.reminders.TryCreate(ctx, reminder.CreateRequest{
		Platform:       input.Platform,
		PlatformUserID: input.PlatformUserID,
		ChatID:         input.ChatID,
		RawText:        input.Text,
	})
	if err != nil {
		return "", err
	}
	if handled {
		return reply, nil
	}

	return s.chat.Reply(ctx, input.SessionID, input.Text)
}

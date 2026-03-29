package feishu

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hduhelp/hdu-openclaw/internal/bot"
	"github.com/hduhelp/hdu-openclaw/internal/config"
)

// WebhookHandler 负责校验并处理飞书事件回调。
type WebhookHandler struct {
	cfg        config.Config
	botService *bot.Service
	feishu     *Client
	dedup      *eventDeduper
}

// NewWebhookHandler 创建一个用于处理飞书事件的 webhook 处理器。
func NewWebhookHandler(cfg config.Config, botService *bot.Service, feishuClient *Client) *WebhookHandler {
	return &WebhookHandler{
		cfg:        cfg,
		botService: botService,
		feishu:     feishuClient,
		dedup:      newEventDeduper(10 * time.Minute),
	}
}

type urlVerificationRequest struct {
	Type      string `json:"type"`
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
	Encrypt   string `json:"encrypt"`
}

type urlVerificationResponse struct {
	Challenge string `json:"challenge"`
}

type eventEnvelope struct {
	Schema string `json:"schema"`
	Header struct {
		EventID   string `json:"event_id"`
		EventType string `json:"event_type"`
		Token     string `json:"token"`
	} `json:"header"`
	Event eventPayload `json:"event"`
}

type eventPayload struct {
	Sender struct {
		SenderID struct {
			OpenID string `json:"open_id"`
		} `json:"sender_id"`
	} `json:"sender"`
	Message struct {
		MessageType string `json:"message_type"`
		ChatID      string `json:"chat_id"`
		Content     string `json:"content"`
	} `json:"message"`
}

type textMessageContent struct {
	Text string `json:"text"`
}

// HandleProbe 返回一个简短的 JSON 响应，用于飞书的地址探测。
func (h *WebhookHandler) HandleProbe(w http.ResponseWriter, r *http.Request) {
	log.Printf("feishu probe request: method=%s path=%s remote=%s", r.Method, r.URL.Path, r.RemoteAddr)
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "ok",
	})
}

// HandleEvent 校验飞书事件并分发支持的文本消息。
func (h *WebhookHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var verifyReq urlVerificationRequest
	if err := json.Unmarshal(rawBody, &verifyReq); err != nil {
		log.Printf("feishu invalid request body: method=%s body=%s err=%v", r.Method, string(rawBody), err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("feishu event received: type=%s token_present=%t encrypt_present=%t", verifyReq.Type, verifyReq.Token != "", strings.TrimSpace(verifyReq.Encrypt) != "")

	if verifyReq.Type == "url_verification" {
		if verifyReq.Token != h.cfg.FeishuVerificationToken {
			log.Printf("feishu verification token mismatch: received=%q", verifyReq.Token)
			http.Error(w, "invalid verification token", http.StatusUnauthorized)
			return
		}

		writeJSON(w, http.StatusOK, urlVerificationResponse{Challenge: verifyReq.Challenge})
		return
	}

	if strings.TrimSpace(verifyReq.Encrypt) != "" {
		log.Printf("feishu encrypted event rejected in v0.1")
		http.Error(w, errEncryptedEvent.Error(), http.StatusBadRequest)
		return
	}

	var envelope eventEnvelope
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		http.Error(w, "invalid event payload", http.StatusBadRequest)
		return
	}

	if envelope.Header.Token != h.cfg.FeishuVerificationToken {
		log.Printf("feishu event token mismatch: received=%q event_type=%s", envelope.Header.Token, envelope.Header.EventType)
		http.Error(w, "invalid event token", http.StatusUnauthorized)
		return
	}

	if envelope.Header.EventType != "im.message.receive_v1" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if envelope.Header.EventID == "" || h.dedup.Seen(envelope.Header.EventID) {
		w.WriteHeader(http.StatusOK)
		return
	}

	if envelope.Event.Message.MessageType != "text" {
		w.WriteHeader(http.StatusOK)
		_ = h.feishu.SendTextToChat(context.Background(), envelope.Event.Message.ChatID, "目前先只支持文本消息。")
		return
	}

	var content textMessageContent
	if err := json.Unmarshal([]byte(envelope.Event.Message.Content), &content); err != nil {
		http.Error(w, "invalid message content", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)

	go h.handleChatEvent(envelope, strings.TrimSpace(content.Text))
}

// handleChatEvent 执行下游机器人处理流程，并把最终回复发回飞书。
func (h *WebhookHandler) handleChatEvent(envelope eventEnvelope, text string) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	sessionID := envelope.Event.Message.ChatID
	if sessionID == "" {
		sessionID = envelope.Event.Sender.SenderID.OpenID
	}

	answer, err := h.botService.HandleText(ctx, bot.MessageInput{
		SessionID:      sessionID,
		Platform:       "feishu",
		PlatformUserID: envelope.Event.Sender.SenderID.OpenID,
		ChatID:         envelope.Event.Message.ChatID,
		Text:           text,
	})
	if err != nil {
		log.Printf("chat service failed: event_id=%s err=%v", envelope.Header.EventID, err)
		answer = "我刚刚有点忙，请稍后再试。"
	}

	if err := h.feishu.SendTextToChat(ctx, envelope.Event.Message.ChatID, answer); err != nil {
		log.Printf("send feishu message failed: event_id=%s err=%v", envelope.Header.EventID, err)
	}
}

// writeJSON 以指定状态码写出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// eventDeduper 会短时间记住事件 ID，避免飞书重试造成重复处理。
type eventDeduper struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]time.Time
}

// newEventDeduper 创建一个新的内存事件去重器。
func newEventDeduper(ttl time.Duration) *eventDeduper {
	return &eventDeduper{
		ttl:   ttl,
		items: make(map[string]time.Time),
	}
}

// Seen 在事件已于 TTL 窗口内处理过时返回 true。
func (d *eventDeduper) Seen(eventID string) bool {
	now := time.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	for id, expireAt := range d.items {
		if now.After(expireAt) {
			delete(d.items, id)
		}
	}

	if expireAt, ok := d.items[eventID]; ok && now.Before(expireAt) {
		return true
	}

	d.items[eventID] = now.Add(d.ttl)
	return false
}

var errEncryptedEvent = errors.New("encrypted feishu events are not supported in v0.1")

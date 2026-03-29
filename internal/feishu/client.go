package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hduhelp/hdu-openclaw/internal/config"
)

// Client 封装当前机器人回复和提醒推送所需的飞书 API。
type Client struct {
	appID      string
	appSecret  string
	httpClient *http.Client

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// NewClient 使用应用凭证创建飞书 API 客户端。
func NewClient(cfg config.Config) *Client {
	return &Client{
		appID:     cfg.FeishuAppID,
		appSecret: cfg.FeishuAppSecret,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type tenantTokenRequest struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type tenantTokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	Expire            int    `json:"expire"`
}

// getTenantAccessToken 获取并缓存调用飞书开放接口所需的 tenant access token。
func (c *Client) getTenantAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		return c.accessToken, nil
	}

	payload := tenantTokenRequest{
		AppID:     c.appID,
		AppSecret: c.appSecret,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal tenant token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create tenant token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request tenant token: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read tenant token response: %w", err)
	}

	var parsed tenantTokenResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode tenant token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("tenant token status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if parsed.Code != 0 {
		return "", fmt.Errorf("tenant token error %d: %s", parsed.Code, parsed.Msg)
	}

	c.accessToken = parsed.TenantAccessToken
	c.expiresAt = time.Now().Add(time.Duration(parsed.Expire-60) * time.Second)
	return c.accessToken, nil
}

type createMessageRequest struct {
	ReceiveID string `json:"receive_id"`
	MsgType   string `json:"msg_type"`
	Content   string `json:"content"`
}

type createMessageResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// SendTextToChat 向指定飞书会话发送一条纯文本消息。
func (c *Client) SendTextToChat(ctx context.Context, chatID, text string) error {
	token, err := c.getTenantAccessToken(ctx)
	if err != nil {
		return err
	}

	contentBody, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("marshal feishu text content: %w", err)
	}

	payload := createMessageRequest{
		ReceiveID: chatID,
		MsgType:   "text",
		Content:   string(contentBody),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal create message request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create send message request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send feishu message: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read send message response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("send message status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed createMessageResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("decode send message response: %w", err)
	}
	if parsed.Code != 0 {
		return fmt.Errorf("send message error %d: %s", parsed.Code, parsed.Msg)
	}
	return nil
}

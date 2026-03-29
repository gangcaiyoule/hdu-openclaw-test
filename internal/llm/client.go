package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hduhelp/hdu-openclaw/internal/config"
)

// ChatMessage 表示一条 chat-completions 风格的消息。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Client 封装了一个兼容 OpenAI Chat Completions 的接口客户端。
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient 根据统一配置创建 LLM 客户端。
func NewClient(cfg config.Config) *Client {
	return &Client{
		baseURL: cfg.LLMBaseURL,
		apiKey:  cfg.LLMAPIKey,
		model:   cfg.LLMModel,
		httpClient: &http.Client{
			Timeout: cfg.LLMTimeout,
		},
	}
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Chat 发送一次聊天补全请求，并返回第一条文本结果。
func (c *Client) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	payload := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.7,
		Stream:      false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送聊天请求: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取聊天响应: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("聊天 API 状态 %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("解码聊天响应: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("聊天 API 错误: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("聊天 API 返回了空结果")
	}

	answer := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if answer == "" {
		return "", fmt.Errorf("聊天 API 返回了空内容")
	}
	return answer, nil
}

package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppAddr string

	FeishuAppID             string
	FeishuAppSecret         string
	FeishuVerificationToken string

	LLMBaseURL      string
	LLMAPIKey       string
	LLMModel        string
	LLMSystemPrompt string
	LLMTimeout      time.Duration

	ChatHistoryLimit int
}

func Load() (Config, error) {
	loadDotEnv(".env")

	cfg := Config{
		AppAddr:                 getEnv("APP_ADDR", ":8080"),
		FeishuAppID:             strings.TrimSpace(os.Getenv("FEISHU_APP_ID")),
		FeishuAppSecret:         strings.TrimSpace(os.Getenv("FEISHU_APP_SECRET")),
		FeishuVerificationToken: strings.TrimSpace(os.Getenv("FEISHU_VERIFICATION_TOKEN")),
		LLMBaseURL:              strings.TrimRight(getEnv("LLM_BASE_URL", "https://api.openai.com/v1"), "/"),
		LLMAPIKey:               strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		LLMModel:                getEnv("LLM_MODEL", "gpt-4o-mini"),
		LLMSystemPrompt:         getEnv("LLM_SYSTEM_PROMPT", "你是杭电龙虾的飞书助手，回答简洁、自然、友好，优先使用中文。"),
		LLMTimeout:              time.Duration(getEnvInt("LLM_TIMEOUT_SECONDS", 30)) * time.Second,
		ChatHistoryLimit:        getEnvInt("CHAT_HISTORY_LIMIT", 8),
	}

	if cfg.FeishuAppID == "" {
		return Config{}, errors.New("FEISHU_APP_ID is required")
	}
	if cfg.FeishuAppSecret == "" {
		return Config{}, errors.New("FEISHU_APP_SECRET is required")
	}
	if cfg.FeishuVerificationToken == "" {
		return Config{}, errors.New("FEISHU_VERIFICATION_TOKEN is required")
	}
	if cfg.LLMBaseURL == "" {
		return Config{}, errors.New("LLM_BASE_URL is required")
	}
	if cfg.LLMAPIKey == "" {
		return Config{}, errors.New("LLM_API_KEY is required")
	}
	if cfg.LLMModel == "" {
		return Config{}, errors.New("LLM_MODEL is required")
	}
	if cfg.ChatHistoryLimit < 0 {
		return Config{}, fmt.Errorf("CHAT_HISTORY_LIMIT must be >= 0")
	}

	return cfg, nil
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		_ = os.Setenv(key, value)
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

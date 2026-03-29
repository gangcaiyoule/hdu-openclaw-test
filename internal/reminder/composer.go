package reminder

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hduhelp/hdu-openclaw/internal/config"
	"github.com/hduhelp/hdu-openclaw/internal/llm"
)

const reminderComposePrompt = `你是杭电龙虾的提醒秘书。
你的任务是把一条提醒任务改写成更自然、更有人设的飞书提醒文案。

要求：
1. 保持简短，自然，像助手在轻声提醒用户。
2. 保留提醒核心内容，不能改动事实。
3. 不要编造时间、地点、任务细节。
4. 不要输出 markdown，不要解释。
5. 尽量使用中文，可以稍微有一点“杭电龙虾”助手的人设，但不要浮夸。
6. 输出只是一段可直接发送给用户的话。`

// Composer 定义提醒文案生成层需要提供的能力。
type Composer interface {
	Compose(ctx context.Context, task Task) string
}

// LLMComposer 使用大模型生成更自然的提醒文案，并在失败时回退到朴素文本。
type LLMComposer struct {
	cfg config.Config
	llm *llm.Client
}

// NewLLMComposer 创建一个基于大模型的提醒文案生成器。
func NewLLMComposer(cfg config.Config, llmClient *llm.Client) *LLMComposer {
	return &LLMComposer{
		cfg: cfg,
		llm: llmClient,
	}
}

// Compose 根据提醒任务生成最终发送给用户的提醒文案。
func (c *LLMComposer) Compose(ctx context.Context, task Task) string {
	fallback := c.fallback(task)

	raw, err := c.llm.Chat(ctx, []llm.ChatMessage{
		{Role: "system", Content: c.cfg.LLMSystemPrompt + "\n\n" + reminderComposePrompt},
		{Role: "user", Content: fmt.Sprintf("提醒内容：%s\n提醒类型：%s", task.RemindText, task.ScheduleType)},
	})
	if err != nil {
		log.Printf("reminder compose failed, fallback used: task_id=%d err=%v", task.ID, err)
		return fallback
	}

	message := strings.TrimSpace(raw)
	if message == "" {
		return fallback
	}
	return message
}

// fallback 返回一条稳定可用的默认提醒文案。
func (c *LLMComposer) fallback(task Task) string {
	return "提醒你：" + task.RemindText
}

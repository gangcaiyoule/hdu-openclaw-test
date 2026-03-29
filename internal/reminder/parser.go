package reminder

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hduhelp/hdu-openclaw/internal/config"
	"github.com/hduhelp/hdu-openclaw/internal/llm"
)

const reminderParserPrompt = `你是提醒任务解析器。
你的任务是判断用户输入是否是在创建提醒任务。
如果不是提醒请求，请输出 {"intent":"chat","parse_success":true}。
如果是提醒请求，请输出结构化 JSON。

输出规则：
1. 只能输出 JSON，不能输出解释。
2. intent 只能是 reminder_create 或 chat。
3. schedule_type 只能是 once、daily 或 null。
4. run_at 必须使用 RFC3339 格式，例如 2026-03-30T08:00:00+08:00。
5. repeat_time 如果存在，使用 HH:mm 格式。
6. timezone 固定使用 Asia/Shanghai。
7. parse_success 为 false 时，error_reason 需要解释原因。

示例1：
用户输入：一分钟后提醒我吃饭
输出：
{"intent":"reminder_create","schedule_type":"once","remind_text":"吃饭","run_at":"2026-03-29T19:22:00+08:00","repeat_time":null,"repeat_rule":null,"timezone":"Asia/Shanghai","parse_success":true,"error_reason":null}

示例2：
用户输入：3.29 19:21提醒我吃饭
输出：
{"intent":"reminder_create","schedule_type":"once","remind_text":"吃饭","run_at":"2026-03-29T19:21:00+08:00","repeat_time":null,"repeat_rule":null,"timezone":"Asia/Shanghai","parse_success":true,"error_reason":null}

示例3：
用户输入：每天晚上10点提醒我背单词
输出：
{"intent":"reminder_create","schedule_type":"daily","remind_text":"背单词","run_at":null,"repeat_time":"22:00","repeat_rule":"daily","timezone":"Asia/Shanghai","parse_success":true,"error_reason":null}`

// Parser 使用配置好的大模型判断一条消息是否应创建提醒。
type Parser struct {
	cfg config.Config
	llm *llm.Client
}

// NewParser 创建一个与主聊天链路共用 LLM 客户端的提醒解析器。
func NewParser(cfg config.Config, llmClient *llm.Client) *Parser {
	return &Parser{
		cfg: cfg,
		llm: llmClient,
	}
}

// Parse 将自然语言消息解析为结构化的提醒意图。
func (p *Parser) Parse(ctx context.Context, userText string, now time.Time) (ParseResult, error) {
	userPrompt := fmt.Sprintf(`当前时间：%s
当前时区：Asia/Shanghai
用户输入：%s

请判断这是不是提醒请求，并输出 JSON。`,
		now.In(time.FixedZone("CST", 8*3600)).Format(time.RFC3339),
		userText,
	)

	raw, err := p.llm.Chat(ctx, []llm.ChatMessage{
		{Role: "system", Content: reminderParserPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return ParseResult{}, fmt.Errorf("call reminder parser llm: %w", err)
	}

	log.Printf("reminder parser raw output: input=%q raw=%s", userText, raw)

	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var parsed ParseResult
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		return ParseResult{}, fmt.Errorf("decode reminder parser result: %w; raw=%s", err, clean)
	}
	log.Printf("reminder parser parsed result: input=%q intent=%s schedule_type=%s parse_success=%t run_at=%v repeat_time=%v remind_text=%q", userText, parsed.Intent, parsed.ScheduleType, parsed.ParseSuccess, parsed.RunAt, parsed.RepeatTime, parsed.RemindText)
	return parsed, nil
}

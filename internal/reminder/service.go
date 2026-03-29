package reminder

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// Parser 定义提醒意图解析层需要提供的能力。
type Parser interface {
	Parse(ctx context.Context, userText string, now time.Time) (ParseResult, error)
}

// Service 负责提醒解析、落库以及用户确认文案生成。
type Service struct {
	repository Repository
	parser     Parser
}

// NewService 创建一个基于仓储和解析器的提醒应用服务。
func NewService(repository Repository, parser Parser) *Service {
	return &Service{repository: repository, parser: parser}
}

// TryCreate 尝试把用户消息识别并创建为提醒任务。
func (s *Service) TryCreate(ctx context.Context, req CreateRequest) (bool, string, error) {
	now := time.Now().In(location())
	parsed, err := s.parser.Parse(ctx, req.RawText, now)
	if err != nil {
		log.Printf("reminder try create parse failed: raw_text=%q err=%v", req.RawText, err)
		return false, "", err
	}

	if parsed.Intent != "reminder_create" {
		return false, "", nil
	}

	if !parsed.ParseSuccess {
		log.Printf("reminder try create parse unsuccessful: raw_text=%q error_reason=%v", req.RawText, parsed.ErrorReason)
		return true, "我识别到你想创建提醒，但还没看懂时间。你可以换一种说法，比如“5分钟后提醒我喝药”或“明天早上8点提醒我吃早餐”。", nil
	}

	task, err := s.buildTask(req, parsed, now)
	if err != nil {
		log.Printf("reminder try create build task failed: raw_text=%q parsed=%+v err=%v", req.RawText, parsed, err)
		return true, "我识别到你想创建提醒，但解析具体时间时失败了。你可以换一种说法再试一次。", nil
	}

	if err := s.repository.Create(ctx, &task); err != nil {
		log.Printf("reminder try create save task failed: raw_text=%q err=%v", req.RawText, err)
		return true, "", err
	}

	return true, s.formatConfirmation(task), nil
}

// NextDailyRun 计算每日提醒下一次的绝对执行时间。
func (s *Service) NextDailyRun(now time.Time, repeatTime string) (time.Time, error) {
	return nextDailyRun(now, repeatTime)
}

// buildTask 将解析结果转换成可落库的提醒任务记录。
func (s *Service) buildTask(req CreateRequest, parsed ParseResult, now time.Time) (Task, error) {
	task := Task{
		Platform:       req.Platform,
		PlatformUserID: req.PlatformUserID,
		ChatID:         req.ChatID,
		RawText:        req.RawText,
		RemindText:     strings.TrimSpace(parsed.RemindText),
		ScheduleType:   parsed.ScheduleType,
		Timezone:       "Asia/Shanghai",
		Status:         "active",
	}

	switch parsed.ScheduleType {
	case "once":
		if parsed.RunAt == nil || strings.TrimSpace(*parsed.RunAt) == "" {
			return Task{}, fmt.Errorf("missing run_at for once reminder")
		}
		runAt, err := parseRunAt(strings.TrimSpace(*parsed.RunAt))
		if err != nil {
			return Task{}, fmt.Errorf("parse run_at: %w", err)
		}
		runAt = runAt.In(location())
		if !runAt.After(now) {
			return Task{}, fmt.Errorf("run_at must be in the future")
		}
		task.RunAt = &runAt
		task.NextRunTime = runAt
	case "daily":
		if parsed.RepeatTime == nil || strings.TrimSpace(*parsed.RepeatTime) == "" {
			return Task{}, fmt.Errorf("missing repeat_time for daily reminder")
		}
		repeatTime := strings.TrimSpace(*parsed.RepeatTime)
		task.RepeatTime = &repeatTime
		if parsed.RepeatRule != nil {
			repeatRule := strings.TrimSpace(*parsed.RepeatRule)
			task.RepeatRule = &repeatRule
		}
		nextRun, err := nextDailyRun(now, repeatTime)
		if err != nil {
			return Task{}, err
		}
		task.NextRunTime = nextRun
	default:
		return Task{}, fmt.Errorf("unsupported schedule_type: %s", parsed.ScheduleType)
	}

	if task.RemindText == "" {
		return Task{}, fmt.Errorf("remind_text is required")
	}

	return task, nil
}

// formatConfirmation 生成提醒创建成功后的用户提示文案。
func (s *Service) formatConfirmation(task Task) string {
	switch task.ScheduleType {
	case "once":
		return fmt.Sprintf("已为你创建提醒：%s 提醒你%s。", task.NextRunTime.In(location()).Format("2006-01-02 15:04"), task.RemindText)
	case "daily":
		if task.RepeatTime != nil {
			return fmt.Sprintf("已为你创建每日提醒：每天 %s 提醒你%s。", *task.RepeatTime, task.RemindText)
		}
	}
	return "已为你创建提醒。"
}

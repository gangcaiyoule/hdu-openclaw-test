package reminder

import (
	"context"
	"log"
	"time"

	"github.com/hduhelp/hdu-openclaw/internal/config"
)

// MessageSender 定义调度器发送提醒消息所需的最小能力接口。
type MessageSender interface {
	SendTextToChat(ctx context.Context, chatID, text string) error
}

// Scheduler 负责周期性扫描到期提醒任务并发送飞书通知。
type Scheduler struct {
	cfg        config.Config
	repository Repository
	sender     MessageSender
	composer   Composer
}

// NewScheduler 创建一个用于提醒投递的调度器。
func NewScheduler(cfg config.Config, repository Repository, sender MessageSender, composer Composer) *Scheduler {
	return &Scheduler{
		cfg:        cfg,
		repository: repository,
		sender:     sender,
		composer:   composer,
	}
}

// Start 启动提醒调度循环，直到上下文被取消。
func (s *Scheduler) Start(ctx context.Context) {
	log.Printf("reminder scheduler started, interval=%s", s.cfg.SchedulerInterval)
	s.runOnce(ctx)

	ticker := time.NewTicker(s.cfg.SchedulerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("reminder scheduler stopped")
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

// runOnce 执行一次“查询到期任务 -> 发送提醒 -> 更新状态”的完整流程。
func (s *Scheduler) runOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	tasks, err := s.repository.ListDueTasks(runCtx, time.Now().In(s.location()), s.cfg.SchedulerBatchSize)
	if err != nil {
		log.Printf("scheduler list due tasks failed: %v", err)
		return
	}

	for _, task := range tasks {
		if err := s.handleTask(runCtx, task); err != nil {
			log.Printf("scheduler handle task failed: task_id=%d err=%v", task.ID, err)
		}
	}
}

// handleTask 处理单条提醒任务，并在发送后更新其调度状态。
func (s *Scheduler) handleTask(ctx context.Context, task Task) error {
	message := s.composer.Compose(ctx, task)
	if err := s.sender.SendTextToChat(ctx, task.ChatID, message); err != nil {
		return err
	}

	now := time.Now().In(s.location())
	switch task.ScheduleType {
	case "once":
		return s.repository.MarkDone(ctx, task.ID, now)
	case "daily":
		if task.RepeatTime == nil {
			return nil
		}
		nextRun, err := nextDailyRun(now, *task.RepeatTime)
		if err != nil {
			return err
		}
		return s.repository.RescheduleDaily(ctx, task.ID, now, nextRun)
	default:
		return nil
	}
}

// location 返回调度器使用的时区。
func (s *Scheduler) location() *time.Location {
	return location()
}

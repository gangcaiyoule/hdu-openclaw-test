package reminder

import (
	"context"
	"time"
)

// Repository 定义提醒任务持久化层需要提供的能力。
type Repository interface {
	EnsureSchema(ctx context.Context) error
	Create(ctx context.Context, task *Task) error
	ListDueTasks(ctx context.Context, now time.Time, limit int) ([]Task, error)
	MarkDone(ctx context.Context, id int64, lastRun time.Time) error
	RescheduleDaily(ctx context.Context, id int64, lastRun, nextRun time.Time) error
}

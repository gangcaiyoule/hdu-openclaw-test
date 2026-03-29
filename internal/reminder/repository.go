package reminder

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository 负责将提醒任务持久化到 PostgreSQL。
type Repository struct {
	db *sql.DB
}

// NewRepository 创建一个基于共享数据库连接池的提醒仓储。
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// EnsureSchema 在数据库中创建 reminder_tasks 表及其索引。
func (r *Repository) EnsureSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS reminder_tasks (
    id BIGSERIAL PRIMARY KEY,
    platform TEXT NOT NULL,
    platform_user_id TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    raw_text TEXT NOT NULL,
    remind_text TEXT NOT NULL,
    schedule_type TEXT NOT NULL,
    run_at TIMESTAMPTZ NULL,
    repeat_time TEXT NULL,
    repeat_rule TEXT NULL,
    timezone TEXT NOT NULL,
    next_run_time TIMESTAMPTZ NOT NULL,
    last_run_time TIMESTAMPTZ NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reminder_tasks_due
    ON reminder_tasks (status, next_run_time);
`
	_, err := r.db.ExecContext(ctx, ddl)
	if err != nil {
		return fmt.Errorf("create reminder schema: %w", err)
	}
	return nil
}

// Create 向数据库插入一条新的提醒任务。
func (r *Repository) Create(ctx context.Context, task *Task) error {
	const query = `
INSERT INTO reminder_tasks (
    platform, platform_user_id, chat_id, raw_text, remind_text, schedule_type,
    run_at, repeat_time, repeat_rule, timezone, next_run_time, status
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12
)
RETURNING id, created_at, updated_at;
`

	err := r.db.QueryRowContext(
		ctx,
		query,
		task.Platform,
		task.PlatformUserID,
		task.ChatID,
		task.RawText,
		task.RemindText,
		task.ScheduleType,
		task.RunAt,
		task.RepeatTime,
		task.RepeatRule,
		task.Timezone,
		task.NextRunTime,
		task.Status,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert reminder task: %w", err)
	}
	return nil
}

// ListDueTasks 查询已经到期、等待执行的活跃提醒任务。
func (r *Repository) ListDueTasks(ctx context.Context, now time.Time, limit int) ([]Task, error) {
	const query = `
SELECT
    id, platform, platform_user_id, chat_id, raw_text, remind_text, schedule_type,
    run_at, repeat_time, repeat_rule, timezone, next_run_time, last_run_time,
    status, created_at, updated_at
FROM reminder_tasks
WHERE status = 'active' AND next_run_time <= $1
ORDER BY next_run_time ASC
LIMIT $2;
`

	rows, err := r.db.QueryContext(ctx, query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("list due reminder tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due reminder tasks: %w", err)
	}
	return tasks, nil
}

// MarkDone 在一次性提醒发送完成后将任务标记为已完成。
func (r *Repository) MarkDone(ctx context.Context, id int64, lastRun time.Time) error {
	const query = `
UPDATE reminder_tasks
SET status = 'done', last_run_time = $2, updated_at = NOW()
WHERE id = $1;
`
	if _, err := r.db.ExecContext(ctx, query, id, lastRun); err != nil {
		return fmt.Errorf("mark reminder task done: %w", err)
	}
	return nil
}

// RescheduleDaily 在每日提醒执行后推进到下一次触发时间。
func (r *Repository) RescheduleDaily(ctx context.Context, id int64, lastRun, nextRun time.Time) error {
	const query = `
UPDATE reminder_tasks
SET last_run_time = $2, next_run_time = $3, updated_at = NOW()
WHERE id = $1;
`
	if _, err := r.db.ExecContext(ctx, query, id, lastRun, nextRun); err != nil {
		return fmt.Errorf("reschedule daily reminder task: %w", err)
	}
	return nil
}

// scanTask 将 SQL 查询结果转换成服务层使用的 Task 结构。
func scanTask(scanner interface {
	Scan(dest ...any) error
}) (Task, error) {
	var task Task
	if err := scanner.Scan(
		&task.ID,
		&task.Platform,
		&task.PlatformUserID,
		&task.ChatID,
		&task.RawText,
		&task.RemindText,
		&task.ScheduleType,
		&task.RunAt,
		&task.RepeatTime,
		&task.RepeatRule,
		&task.Timezone,
		&task.NextRunTime,
		&task.LastRunTime,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return Task{}, fmt.Errorf("scan reminder task: %w", err)
	}
	return task, nil
}

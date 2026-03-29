package reminder

import "time"

// Task 表示一条已落库、可由调度器执行的提醒任务。
type Task struct {
	ID             int64
	Platform       string
	PlatformUserID string
	ChatID         string
	RawText        string
	RemindText     string
	ScheduleType   string
	RunAt          *time.Time
	RepeatTime     *string
	RepeatRule     *string
	Timezone       string
	NextRunTime    time.Time
	LastRunTime    *time.Time
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

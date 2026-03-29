package reminder

// CreateRequest 表示创建提醒任务所需的入参。
type CreateRequest struct {
	Platform       string
	PlatformUserID string
	ChatID         string
	RawText        string
}

// ParseResult 表示提醒解析模型返回的结构化结果。
type ParseResult struct {
	Intent       string  `json:"intent"`
	ScheduleType string  `json:"schedule_type"`
	RemindText   string  `json:"remind_text"`
	RunAt        *string `json:"run_at"`
	RepeatTime   *string `json:"repeat_time"`
	RepeatRule   *string `json:"repeat_rule"`
	Timezone     string  `json:"timezone"`
	ParseSuccess bool    `json:"parse_success"`
	ErrorReason  *string `json:"error_reason"`
}

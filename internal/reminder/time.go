package reminder

import (
	"fmt"
	"strings"
	"time"
)

// location 返回提醒功能默认使用的时区。
func location() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// nextDailyRun 计算每日提醒下一次的绝对执行时间。
func nextDailyRun(now time.Time, repeatTime string) (time.Time, error) {
	hour, err := time.Parse("15:04", repeatTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse repeat_time: %w", err)
	}

	current := now.In(location())
	next := time.Date(current.Year(), current.Month(), current.Day(), hour.Hour(), hour.Minute(), 0, 0, location())
	if !next.After(current) {
		next = next.Add(24 * time.Hour)
	}
	return next, nil
}

// parseRunAt 兼容多种常见时间格式，降低模型输出轻微偏差导致的失败率。
func parseRunAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	layoutsWithZone := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05Z07:00",
	}
	for _, layout := range layoutsWithZone {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.In(location()), nil
		}
	}

	layoutsWithoutZone := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-1-2 15:04:05",
		"2006-1-2 15:04",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/1/2 15:04:05",
		"2006/1/2 15:04",
	}
	for _, layout := range layoutsWithoutZone {
		if parsed, err := time.ParseInLocation(layout, raw, location()); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", raw)
}

package utils

import (
	"database/sql"
	"time"
)

// FormatNullTime 格式化 sql.NullTime 为 RFC3339 字符串
func FormatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

// FormatNullTimeCustom 使用自定义格式格式化 sql.NullTime
func FormatNullTimeCustom(t sql.NullTime, layout string) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(layout)
}

// FormatTimePtr 格式化时间指针
func FormatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// FormatTimePtrCustom 使用自定义格式格式化时间指针
func FormatTimePtrCustom(t *time.Time, layout string) string {
	if t == nil {
		return ""
	}
	return t.Format(layout)
}

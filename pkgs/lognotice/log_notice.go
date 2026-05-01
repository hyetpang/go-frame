package lognotice

import (
	"go.uber.org/zap"
)

// Notifier 错误日志通知接口,实现方负责把错误信息异步推送到 webhook 等告警通道。
type Notifier interface {
	Notice(msg string, fields ...zap.Field)
}

// 默认的一个实现,避免 panic
var logNotice Notifier = defaultLogNotice{}

type defaultLogNotice struct{}

// 不实现
func (defaultLogNotice) Notice(msg string, fields ...zap.Field) {}

// 日志通知
func Notice(msg string, fields ...zap.Field) {
	logNotice.Notice(msg, fields...)
}

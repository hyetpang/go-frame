package interfaces

import "go.uber.org/zap"

// 错误日志通知接口
type LogNoticeInterface interface {
	Notice(msg string, fields ...zap.Field)
}

package lognotice

import (
	"github.com/HyetPang/go-frame/pkgs/interfaces"
	"go.uber.org/zap"
)

// 默认的一个实现,避免panic
var logNotice interfaces.LogNoticeInterface = defaultLogNotice{}

type defaultLogNotice struct{}

// 不实现
func (defaultLogNotice) Notice(msg string, fields ...zap.Field) {}

// 日志通知
func Notice(msg string, fields ...zap.Field) {
	logNotice.Notice(msg, fields...)
}

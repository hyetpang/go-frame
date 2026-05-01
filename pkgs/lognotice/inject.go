package lognotice

import (
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/zap"
)

// Inject 注入一个 Notifier 实现,并向 pkgs/logs 注册同名 hook,
// 这样 logs.Error 触发时会自动转发到通知器,避免 logs 包反向依赖 lognotice。
func Inject(notifier Notifier) {
	logNotice = notifier
	logs.RegisterNoticeHook(func(msg string, filename string, line int, fields ...zap.Field) {
		notifier.Notice(msg, filename, line, fields...)
	})
}

// zap日志的封装
package logs

import (
	"runtime"
	"sync/atomic"

	"go.uber.org/zap"
)

// NoticeHook 错误日志通知钩子函数签名,filename 与 line 表示触发 logs.Error 的源代码位置。
type NoticeHook func(msg string, filename string, line int, fields ...zap.Field)

// noticeHook 当前注册的通知钩子,使用 atomic.Pointer 保证读写无锁安全。
var noticeHook atomic.Pointer[NoticeHook]

// RegisterNoticeHook 注册错误日志通知钩子,后续 logs.Error 会异步触发该钩子。
// 该机制允许 pkgs/lognotice 在初始化时注入实现,避免 pkgs/logs 反向依赖。
func RegisterNoticeHook(hook NoticeHook) {
	noticeHook.Store(&hook)
}

// unregisterNoticeHook 取消注册,仅供测试使用。
func unregisterNoticeHook() {
	noticeHook.Store(nil)
}

// callNoticeHook 在 logs.Error 触发时异步调用钩子,callerSkip 表示从调用 callNoticeHook 处再向上跳过几层。
func callNoticeHook(callerSkip int, msg string, fields ...zap.Field) {
	hookPtr := noticeHook.Load()
	if hookPtr == nil {
		return
	}
	_, filename, line, _ := runtime.Caller(callerSkip + 1)
	hook := *hookPtr
	go hook(msg, filename, line, fields...)
}

func Error(msg string, fields ...zap.Field) {
	callNoticeHook(1, msg, fields...)
	zap.L().Error(msg, fields...)
}

func ErrorWithoutNotice(msg string, fields ...zap.Field) {
	zap.L().Error(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	zap.L().Debug(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	zap.L().Fatal(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	zap.L().Warn(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	zap.L().Info(msg, fields...)
}

func With(fields ...zap.Field) *Log {
	return &Log{Logger: zap.L().With(fields...)}
}

// Log 在 *zap.Logger 之上仅覆盖 Error（附带错误通知），
// Debug/Info/Warn/Fatal 通过结构体嵌入自动提升。
type Log struct {
	*zap.Logger
}

func (l *Log) Error(msg string, fields ...zap.Field) {
	callNoticeHook(1, msg, fields...)
	l.Logger.Error(msg, fields...)
}

func (l *Log) ErrorWithoutNotice(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

func (l *Log) With(fields ...zap.Field) *Log {
	return &Log{Logger: l.Logger.With(fields...)}
}

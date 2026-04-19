// zap日志的封装
package logs

import (
	"github.com/hyetpang/go-frame/pkgs/lognotice"
	"go.uber.org/zap"
)

func Error(msg string, fields ...zap.Field) {
	lognotice.Notice(msg, fields...)
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
	lognotice.Notice(msg, fields...)
	l.Logger.Error(msg, fields...)
}

func (l *Log) ErrorWithoutNotice(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

func (l *Log) With(fields ...zap.Field) *Log {
	return &Log{Logger: l.Logger.With(fields...)}
}

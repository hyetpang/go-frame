// zap日志的封装
package logs

import (
	"github.com/HyetPang/go-frame/pkgs/lognotice"
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

func With(fields ...zap.Field) *log {
	// return zap.L().With(fields...).WithOptions(zap.AddCallerSkip())
	return &log{
		Logger: zap.L().With(fields...),
	}
}

type log struct {
	*zap.Logger
}

func (l *log) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

func (l *log) ErrorWithoutNotice(msg string, fields ...zap.Field) {
	lognotice.Notice(msg, fields...)
	l.Logger.Error(msg, fields...)
}

func (l *log) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

func (l *log) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

func (l *log) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

func (l *log) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

func (l *log) With(fields ...zap.Field) *log {
	l.Logger = l.Logger.With(fields...)
	return l
}

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

func With(fields ...zap.Field) *Log {
	return &Log{
		Logger: zap.L().With(fields...),
	}
}

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

func (l *Log) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

func (l *Log) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

func (l *Log) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

func (l *Log) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

func (l *Log) With(fields ...zap.Field) *Log {
	return &Log{
		Logger: l.Logger.With(fields...),
	}
}

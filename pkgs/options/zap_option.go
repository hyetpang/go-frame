package options

import "go.uber.org/zap/zapcore"

// 使用zap日志，可以不指定
func WithZap(logFile string, logLevel zapcore.Level) Option {
	return func(o *Options) {
		o.LogFile = logFile
		o.LogLevel = logLevel
	}
}

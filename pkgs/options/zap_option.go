package options

import "go.uber.org/zap/zapcore"

func WithZap(logFile string, logLevel zapcore.Level) Option {
	return func(o *Options) {
		o.LogFile = logFile
		o.LogLevel = logLevel
	}
}

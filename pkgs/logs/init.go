package logs

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func Init(logFile string, minLevel zapcore.Level) {
	hook := &lumberjack.Logger{
		Filename:   logFile, // 日志文件路径
		MaxSize:    128,     // 最大日志大小（Mb级别）
		MaxBackups: 30,      // 最多保留30个备份
		MaxAge:     7,       // days
		Compress:   true,    // 是否压缩 disabled by default
		LocalTime:  true,
	}

	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		result := lvl >= minLevel && lvl >= zapcore.ErrorLevel
		return result
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		result := lvl >= minLevel && lvl < zapcore.ErrorLevel
		return result
	})

	fileDebugging := zapcore.AddSync(hook)
	fileErrors := zapcore.AddSync(hook)

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeTime = customTimeEncoder

	fileEncoder := zapcore.NewConsoleEncoder(cfg)
	consoleEncoder := zapcore.NewConsoleEncoder(cfg)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(fileEncoder, fileErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
		zapcore.NewCore(fileEncoder, fileDebugging, lowPriority),
	)

	logger := zap.New(core, zap.AddStacktrace(zap.WarnLevel), zap.AddCallerSkip(1))

	zap.ReplaceGlobals(logger)
}

func customTimeEncoder(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(time.Format("2006-01-02 15:04:05.000000"))
}

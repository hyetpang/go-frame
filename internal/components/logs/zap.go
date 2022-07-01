/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-04-30 16:03:18
 * @FilePath: \go-frame\internal\components\logs\init.go
 */
package logs

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HyetPang/go-frame/pkgs/common"
	"github.com/HyetPang/go-frame/pkgs/validate"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New() *zap.Logger {
	conf := new(config)
	err := viper.UnmarshalKey("zap_log", &conf)
	if err != nil {
		log.Fatal("zap_log配置Unmarshal到对象出错", zap.Error(err))
	}
	validate.MustValidate(conf)
	if len(conf.File) < 1 {
		conf.File = getDefaultLogFile()
	}
	if conf.Level == nil {
		level := int(zap.DebugLevel)
		conf.Level = &level
	}
	hook := &lumberjack.Logger{
		Filename:   conf.File, // 日志文件路径
		MaxSize:    128,       // 最大日志大小（Mb级别）
		MaxBackups: 30,        // 最多保留30个备份
		MaxAge:     7,         // days
		Compress:   true,      // 是否压缩 disabled by default
		LocalTime:  true,
	}
	minLevel := zapcore.Level(*conf.Level)
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

	logger := zap.New(core, zap.AddStacktrace(zap.WarnLevel))

	// logger和下面return的zap.Logger依赖唯一不同是zap.AddCallerSkip(1)，下面return是作为依赖给各种第三方库使用的
	zap.ReplaceGlobals(logger.WithOptions(zap.AddCallerSkip(1)))
	return logger
}

func customTimeEncoder(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(time.Format("2006-01-02 15:04:05.000000"))
}

// 获取默认的日志文件位置
func getDefaultLogFile() string {
	currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	common.Panic(err)
	return filepath.Join(currentPath, "log", strings.Replace(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]), "", 1)+".log")
}

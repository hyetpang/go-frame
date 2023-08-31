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

	"github.com/hyetpang/go-frame/pkgs/common"
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
	common.MustValidate(conf)
	if len(conf.Path) < 1 {
		currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatalf("获取当前文件路径出错:%s", err.Error())
		}
		conf.Path = filepath.Join(currentPath, "logs")
	}
	debugFile, errFile := getLogFilePath(conf.Path)
	hook := &lumberjack.Logger{
		Filename:   debugFile,     // 日志文件路径
		MaxSize:    logMaxSize,    // 最大日志大小（Mb级别）
		MaxBackups: logMaxBackups, // 最多保留30个备份
		MaxAge:     logMaxAge,     // days
		Compress:   true,          // 是否压缩 disabled by default
		LocalTime:  true,
	}
	minLevel := zapcore.Level(conf.Level)
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		result := lvl >= minLevel && lvl >= zapcore.ErrorLevel
		return result
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		result := lvl >= minLevel && lvl < zapcore.ErrorLevel
		return result
	})

	// 错误日志单独写一份文件
	hook_err := &lumberjack.Logger{
		Filename:   errFile,       // 日志文件路径
		MaxSize:    logMaxSize,    // 最大日志大小（Mb级别）
		MaxBackups: logMaxBackups, // 最多保留30个备份
		MaxAge:     logMaxAge,     // days
		Compress:   true,          // 是否压缩 disabled by default
		LocalTime:  true,
	}
	fileError := zapcore.AddSync(hook_err)

	fileDebugging := zapcore.AddSync(hook)
	fileErrors := zapcore.AddSync(hook)

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)
	fileCfg := zap.NewProductionEncoderConfig()
	consoleCfg := zap.NewProductionEncoderConfig()
	if common.Dev {
		fileCfg = zap.NewDevelopmentEncoderConfig()
		consoleCfg = zap.NewDevelopmentEncoderConfig()
	}
	fileCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000000")
	fileEncoder := zapcore.NewJSONEncoder(fileCfg)

	// consoleCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000000")
	consoleEncoder := zapcore.NewConsoleEncoder(consoleCfg)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(fileEncoder, fileErrors, highPriority),
		zapcore.NewCore(fileEncoder, fileError, highPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
		zapcore.NewCore(fileEncoder, fileDebugging, lowPriority),
	)

	logger := zap.New(core, zap.AddStacktrace(zap.WarnLevel))
	// logger和下面return的zap.Logger依赖唯一不同是zap.AddCallerSkip(1)，下面return是作为依赖给各种第三方库使用的
	zap.ReplaceGlobals(logger.WithOptions(zap.AddCallerSkip(1)))
	return logger
}

// func customTimeEncoder(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {
// 	encoder.AppendString(time.Format("2006-01-02 15:04:05.000000"))
// }

// 获取默认的日志文件位置
func getLogFilePath(currentPath string) (string, string) {
	err := makeDir(currentPath)
	common.Panic(err)

	debugDir := filepath.Join(currentPath, "debug")
	err = makeDir(debugDir)
	common.Panic(err)
	debugFile := filepath.Join(debugDir, filepath.Base(os.Args[0])+".log")

	errDir := filepath.Join(currentPath, "error")
	err = makeDir(errDir)
	common.Panic(err)
	errFile := filepath.Join(errDir, filepath.Base(os.Args[0])+".log")
	return debugFile, errFile
}

// 创建不存在的目录
func makeDir(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Mkdir(path, 666)
		}
	}
	return nil
}

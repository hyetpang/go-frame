package logs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyetpang/go-frame/pkgs/common"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New(lc fx.Lifecycle, conf *config) (*zap.Logger, error) {
	applyDefaults(conf)
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("zap_log配置验证不通过: %w", err)
	}
	if len(conf.Path) < 1 {
		currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return nil, fmt.Errorf("获取当前文件路径出错: %w", err)
		}
		conf.Path = filepath.Join(currentPath, "logs")
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
	cores := make([]zapcore.Core, 0, 10)
	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)
	fileCfg := zap.NewProductionEncoderConfig()
	consoleCfg := zap.NewProductionEncoderConfig()

	if common.Dev {
		fileCfg = zap.NewDevelopmentEncoderConfig()
		consoleCfg = zap.NewDevelopmentEncoderConfig()
		consoleCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	fileCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000000")
	fileEncoder := zapcore.NewJSONEncoder(fileCfg)
	consoleCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000000")
	consoleEncoder := zapcore.NewConsoleEncoder(consoleCfg)
	cores = append(cores,
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
	)
	if conf.IsLogFile {
		debugFile, errFile, err := getLogFilePath(conf.Path)
		if err != nil {
			return nil, fmt.Errorf("获取日志文件路径出错: %w", err)
		}
		debugHook := &lumberjack.Logger{
			Filename:   debugFile,
			MaxSize:    conf.LogMaxSize,
			MaxBackups: conf.LogMaxBackups,
			MaxAge:     conf.LogMaxAge,
			Compress:   true,
			LocalTime:  true,
		}
		errHook := &lumberjack.Logger{
			Filename:   errFile,
			MaxSize:    conf.LogMaxSize,
			MaxBackups: conf.LogMaxBackups,
			MaxAge:     conf.LogMaxAge,
			Compress:   true,
			LocalTime:  true,
		}
		fileAllLevels := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= minLevel
		})
		debugSync := zapcore.AddSync(debugHook)
		errSync := zapcore.AddSync(errHook)
		cores = append(cores,
			zapcore.NewCore(fileEncoder, debugSync, fileAllLevels),
			zapcore.NewCore(fileEncoder, errSync, highPriority),
		)
	}

	core := zapcore.NewTee(cores...)

	logger := zap.New(core, zap.AddStacktrace(stacktraceLevel(conf)))
	if len(conf.ServiceName) > 0 {
		logger = logger.WithOptions(zap.Fields(zap.String("service", conf.ServiceName)))
	}
	// logger和下面return的zap.Logger依赖唯一不同是zap.AddCallerSkip(1)，下面return是作为依赖给各种第三方库使用的
	zap.ReplaceGlobals(logger.WithOptions(zap.AddCallerSkip(1)))
	lc.Append(fx.StopHook(func() {
		_ = zap.L().Sync()
		_ = logger.Sync() // 日志同步
	}))
	return logger, nil
}

func applyDefaults(conf *config) {
	if conf.LogMaxSize == 0 {
		conf.LogMaxSize = logMaxSize
	}
	if conf.LogMaxBackups == 0 {
		conf.LogMaxBackups = logMaxBackups
	}
	if conf.LogMaxAge == 0 {
		conf.LogMaxAge = logMaxAge
	}
	if conf.StacktraceLevel == 0 {
		conf.StacktraceLevel = 1
	}
}

func stacktraceLevel(conf *config) zapcore.Level {
	return zapcore.Level(conf.StacktraceLevel)
}

// func customTimeEncoder(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {
// 	encoder.AppendString(time.Format("2006-01-02 15:04:05.000000"))
// }

// 获取默认的日志文件位置
func getLogFilePath(currentPath string) (string, string, error) {
	err := makeDir(currentPath)
	if err != nil {
		return "", "", err
	}

	debugDir := filepath.Join(currentPath, "debug")
	err = makeDir(debugDir)
	if err != nil {
		return "", "", err
	}
	debugFile := filepath.Join(debugDir, filepath.Base(os.Args[0])+".log")

	errDir := filepath.Join(currentPath, "error")
	err = makeDir(errDir)
	if err != nil {
		return "", "", err
	}
	errFile := filepath.Join(errDir, filepath.Base(os.Args[0])+".log")
	return debugFile, errFile, nil
}

// 创建不存在的目录
func makeDir(path string) error {
	return os.MkdirAll(path, 0755)
}

package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/HyetPang/go-frame/pkgs/common"
	"github.com/HyetPang/go-frame/pkgs/logadapter"
	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/HyetPang/go-frame/pkgs/options"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap/zapcore"
)

type App struct {
	app *fx.App
}

func Run(opt ...options.Option) {
	new(opt...).app.Run()
}

func new(opt ...options.Option) *App {
	ops := &options.Options{
		FxOptions:  make([]fx.Option, 0, 10),
		ConfigFile: "./conf/app.toml",
		LogFile:    getDefaultLogFile(),
		LogLevel:   zapcore.DebugLevel,
	}
	for _, op := range opt {
		op(ops)
	}
	// 设置配置文件
	viper.SetConfigFile(ops.ConfigFile)
	common.Panic(viper.ReadInConfig())
	// 日志
	logs.Init(ops.LogFile, ops.LogLevel)
	// 使用zap日志
	ops.FxOptions = append(ops.FxOptions, fx.WithLogger(logadapter.NewFxZap))
	return &App{app: fx.New(ops.FxOptions...)}
}

func getDefaultLogFile() string {
	currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	common.Panic(err)
	return filepath.Join(currentPath, "log", strings.Replace(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]), "", 1)+".log")
}

/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-05-07 17:07:30
 * @FilePath: /go-frame/pkgs/app/app.go
 */
package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/HyetPang/go-frame/internal/adapter/log"
	"github.com/HyetPang/go-frame/internal/components/logs"
	"github.com/HyetPang/go-frame/pkgs/common"
	"github.com/HyetPang/go-frame/pkgs/dev"
	"github.com/HyetPang/go-frame/pkgs/options"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
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
	// 使用zap日志
	ops.FxOptions = append(ops.FxOptions, fx.Provide(func() *zap.Logger {
		return logs.New(ops.LogFile, ops.LogLevel)
	}))
	// 设置配置文件
	viper.SetConfigFile(ops.ConfigFile)
	common.Panic(viper.ReadInConfig())
	ops.FxOptions = append(ops.FxOptions, fx.WithLogger(log.NewFxZap))
	if viper.GetString("server.run_mode") == common.DevMode {
		dev.IsDebug = true
	}
	return &App{app: fx.New(ops.FxOptions...)}
}

// 获取默认的日志文件位置
func getDefaultLogFile() string {
	currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	common.Panic(err)
	return filepath.Join(currentPath, "log", strings.Replace(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]), "", 1)+".log")
}

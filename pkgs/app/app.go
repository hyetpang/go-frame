/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-05-09 10:08:41
 * @FilePath: /go-frame/pkgs/app/app.go
 */
package app

import (
	"context"
	"log"

	adapterLog "github.com/HyetPang/go-frame/internal/adapter/log"
	"github.com/HyetPang/go-frame/internal/components/logs"
	"github.com/HyetPang/go-frame/pkgs/common"
	"github.com/HyetPang/go-frame/pkgs/dev"
	"github.com/HyetPang/go-frame/pkgs/options"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type App struct {
	app     *fx.App
	isStart bool
}

func Run(opt ...options.Option) {
	app := new(opt...)
	if app.isStart {
		err := app.app.Start(context.Background())
		if err != nil {
			log.Fatalf("start 出错:%s", err.Error())
		}
	} else {
		app.app.Run()
	}
}

func new(opt ...options.Option) *App {
	ops := &options.Options{
		FxOptions:  make([]fx.Option, 0, 10),
		ConfigFile: "./conf/app.toml",
		IsStart:    false,
	}
	for _, op := range opt {
		op(ops)
	}
	// 使用zap日志
	ops.FxOptions = append(ops.FxOptions, fx.Provide(logs.New))
	// 设置配置文件
	viper.SetConfigFile(ops.ConfigFile)
	viper.SetConfigType("toml")
	common.Panic(viper.ReadInConfig())
	ops.FxOptions = append(ops.FxOptions, fx.WithLogger(adapterLog.NewFxZap))
	if viper.GetString("server.run_mode") == common.DevMode {
		dev.IsDebug = true
	}
	return &App{
		app:     fx.New(ops.FxOptions...),
		isStart: ops.IsStart,
	}
}

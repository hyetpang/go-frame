package app

import (
	"fmt"
	"runtime/debug"

	adapterLog "github.com/hyetpang/go-frame/internal/adapter/log"
	"github.com/hyetpang/go-frame/internal/components/gin"
	"github.com/hyetpang/go-frame/internal/components/logs"
	frameconfig "github.com/hyetpang/go-frame/internal/config"
	"github.com/hyetpang/go-frame/pkgs/common"
	log "github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/hyetpang/go-frame/pkgs/options"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Run(opt ...options.Option) {
	run(opt...)
}

func run(opt ...options.Option) {
	ops := &options.Options{
		FxOptions:  make([]fx.Option, 0, 10),
		ConfigFile: "./conf/app.toml",
		IsStart:    false,
	}
	for _, op := range opt {
		op(ops)
	}
	conf, err := frameconfig.Load(ops.ConfigFile)
	if err != nil {
		panic(fmt.Errorf("加载配置文件失败: %w", err))
	}
	ops.FxOptions = append(ops.FxOptions,
		fx.Provide(func() *frameconfig.Config { return conf }),
		fx.Provide(frameconfig.SectionProviders()...),
	)
	// 使用zap日志
	ops.FxOptions = append(ops.FxOptions, fx.Provide(logs.New))
	// 打印版本
	ops.FxOptions = append(ops.FxOptions, fx.Invoke(printVersion))
	runMode := conf.Server.RunMode
	isDev := runMode == common.DevMode
	common.Dev = isDev
	if ops.UseHttp {
		ops.FxOptions = append(ops.FxOptions, fx.Provide(gin.New))
	}
	ops.FxOptions = append(ops.FxOptions, fx.WithLogger(adapterLog.NewFxZap))
	app := &App{
		options: ops.FxOptions,
		isStart: ops.IsStart,
	}
	app.run()
}

func printVersion(_ *zap.Logger) {
	info, ok := debug.ReadBuildInfo()
	if ok {
		var vcsTime, vcsRevision string
		for _, b := range info.Settings {
			if b.Key == "vcs.time" {
				vcsTime = b.Value
			}
			if b.Key == "vcs.revision" {
				vcsRevision = b.Value
			}
		}
		log.Info("版本信息", zap.String("git提交时间", vcsTime), zap.String("git提交hash", vcsRevision), zap.String("go_version", info.GoVersion))
	}
}

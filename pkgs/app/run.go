package app

import (
	"fmt"
	"os"
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

// exitFunc/fatalLog 默认指向 os.Exit/logs.Error,测试中可替换以避免真实退出。
var (
	exitFunc = os.Exit
	fatalLog = func(msg string, fields ...zap.Field) { log.Error(msg, fields...) }
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
	conf, err := frameconfig.LoadWithEnv(ops.ConfigFile)
	if err != nil {
		fatalExit(fmt.Errorf("加载配置文件失败: %w", err))
		return
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
	common.SetDev(isDev)
	if ops.UseHttp {
		ops.FxOptions = append(ops.FxOptions, fx.Provide(gin.New))
	}
	ops.FxOptions = append(ops.FxOptions, fx.WithLogger(adapterLog.NewFxZap))
	app := &App{
		options: ops.FxOptions,
		isStart: ops.IsStart,
	}
	if err := app.run(); err != nil {
		fatalExit(err)
		return
	}
}

// fatalExit 在启动/构建失败时记录日志、刷新 zap buffer,并以非 0 退出码退出,
// 让 k8s/systemd 等编排层能感知失败重启。
func fatalExit(err error) {
	fatalLog("应用启动失败", zap.Error(err))
	if l := zap.L(); l != nil {
		_ = l.Sync()
	}
	exitFunc(1)
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

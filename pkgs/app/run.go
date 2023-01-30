package app

import (
	"os"
	"syscall"
	"time"

	adapterLog "github.com/HyetPang/go-frame/internal/adapter/log"
	"github.com/HyetPang/go-frame/internal/components/gin"
	"github.com/HyetPang/go-frame/internal/components/logs"
	"github.com/HyetPang/go-frame/pkgs/common"
	"github.com/HyetPang/go-frame/pkgs/dev"
	log "github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/HyetPang/go-frame/pkgs/options"
	"github.com/jpillora/overseer"
	"github.com/jpillora/overseer/fetcher"
	"github.com/spf13/viper"
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
	// 使用zap日志
	ops.FxOptions = append(ops.FxOptions, fx.Provide(logs.New))
	// 设置配置文件
	// viper.SetDe
	viper.SetConfigFile(ops.ConfigFile)
	viper.SetConfigType("toml")
	common.Panic(viper.ReadInConfig())
	var isDev bool // 这个参数用来控制在本地开发的时候不用平滑重启，直接启动，避免打断点不生效，无法调试的问题
	if viper.GetString("server.run_mode") == common.DevMode || viper.GetString("server.run_mode") == common.TestMode {
		dev.IsDebug = true
		if viper.GetString("server.run_mode") == common.DevMode {
			isDev = true
		}
	}

	var overseerConfig *overseer.Config
	var httpProvider fx.Option
	if ops.UseGraceRestart && !isDev {
		graceRestartConfig := newGraceRestartConfig()
		overseerConfig = &overseer.Config{
			// ExecFile:      graceRestartConfig.ExecFile,
			RestartSignal: syscall.SIGTERM, // 这个重启信号是为了兼容supervisor进程管理器，它默认的终止信号就是TERM
			Address:       graceRestartConfig.HttpAddr,
			Fetcher:       &fetcher.File{Path: graceRestartConfig.ExecLatestFile, Interval: 5 * time.Second},
			Debug:         true, // display log of overseer actions
			PreUpgrade: func(tempBinaryPath string) error {
				log.Info("要更新的文件路径-", zap.String("temp_binary_path", tempBinaryPath))
				_, err := os.Stat(tempBinaryPath)
				if err != nil {
					log.Error("stat temp_binary_path by path err", zap.Error(err))
				}
				return err
			},
		}
		httpProvider = fx.Provide(gin.NewWithGraceRestart)
	} else {
		httpProvider = fx.Provide(gin.New)
	}
	if ops.UseHttp {
		ops.FxOptions = append(ops.FxOptions, httpProvider)
	}
	ops.FxOptions = append(ops.FxOptions, fx.WithLogger(adapterLog.NewFxZap))
	app := &App{
		options: ops.FxOptions,
		isStart: ops.IsStart,
	}
	if ops.UseGraceRestart && !isDev {
		(*overseerConfig).Program = app.runWith
		overseer.Run(*overseerConfig)
	} else {
		app.run()
	}
}

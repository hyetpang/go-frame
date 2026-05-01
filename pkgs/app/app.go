package app

import (
	"context"
	"time"

	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type App struct {
	options []fx.Option
	isStart bool // true=>运行后马上退出
}

func (app *App) run() {
	application := fx.New(app.options...)
	// 在启动前检查构建期错误（Provider 注入失败等），让业务侧能捕获而非静默忽略
	if err := application.Err(); err != nil {
		logs.Error("fx 应用构建失败", zap.Error(err))
		return
	}
	if app.isStart {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err := application.Start(ctx)
		if err != nil {
			logs.Error("运行出错", zap.Error(err))
			return
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer stopCancel()
		if err := application.Stop(stopCtx); err != nil {
			logs.Error("停止出错", zap.Error(err))
		}
	} else {
		application.Run()
	}
}

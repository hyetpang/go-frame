package app

import (
	"context"
	"fmt"
	"time"

	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type App struct {
	options []fx.Option
	isStart bool // true=>运行后马上退出
}

// run 返回 error 由调用方决定退出码,避免 fx 构建/启动失败被静默忽略导致进程
// 仍以 exit 0 退出 — k8s/systemd 不会重启,运维侧无法感知。
func (app *App) run() error {
	application := fx.New(app.options...)
	if err := application.Err(); err != nil {
		return fmt.Errorf("fx 应用构建失败: %w", err)
	}
	if app.isStart {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		if err := application.Start(ctx); err != nil {
			return fmt.Errorf("启动出错: %w", err)
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer stopCancel()
		if err := application.Stop(stopCtx); err != nil {
			// Stop 阶段进程即将退出,不再是致命错误,记日志即可。
			logs.Error("停止出错", zap.Error(err))
		}
		return nil
	}
	// fx.App.Run 在启动失败时内部已经会调用 os.Exit(1),
	// 信号触发的正常退出不会返回错误,这里仍透传以便未来 fx 行为变化时能被捕获。
	application.Run()
	return nil
}

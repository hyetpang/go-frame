package tasks

import (
	"context"
	"errors"

	"github.com/go-logr/zapr"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/robfig/cron/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(zapLog *zap.Logger, lc fx.Lifecycle) *cron.Cron {
	logger := zapr.NewLogger(zapLog)
	cron := cron.New(cron.WithSeconds(), cron.WithChain(cron.Recover(logger), cron.SkipIfStillRunning(logger)), cron.WithLogger(logger))
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			cron.Start()
			logs.Info("定时任务启动成功")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			cronCtx := cron.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-cronCtx.Done():
				err := cronCtx.Err()
				if err != nil && !errors.Is(err, context.Canceled) {
					return err
				}
				logs.Info("定时任务已关闭...")
				return nil
			}
		},
	})
	return cron
}

/*
 * @Date: 2022-04-30 15:53:54
 * @LastEditTime: 2022-05-01 22:54:24
 * @FilePath: \go-frame\internal\components\tasks\tasks.go
 */
package tasks

import (
	"context"

	"github.com/go-logr/zapr"
	"github.com/hyetpang/go-frame/internal/adapter/log"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/robfig/cron/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(zapLog *zap.Logger, lc fx.Lifecycle) *cron.Cron {
	cron := cron.New(cron.WithSeconds(), cron.WithChain(cron.Recover(zapr.NewLogger(zapLog))), cron.WithLogger(cron.VerbosePrintfLogger(log.CronLog{Logger: zapLog})))
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
				if err != context.Canceled {
					return err
				}
				logs.Info("定时任务已关闭...")
				return nil
			}
		},
	})
	return cron
}

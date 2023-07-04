/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-05-09 10:08:41
 * @FilePath: /go-frame/pkgs/app/app.go
 */
package app

import (
	"context"
	"time"

	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/jpillora/overseer"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type App struct {
	options []fx.Option
	isStart bool // true=>运行后马上退出
}

func (app *App) runWith(state overseer.State) {
	app.options = append(app.options, fx.Provide(func() overseer.State {
		return state
	}))
	app.run()
}

func (app *App) run() {
	viper.Debug() // 打印配置项
	application := fx.New(app.options...)
	if app.isStart {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err := application.Start(ctx)
		if err != nil {
			logs.Error("运行出错", zap.Error(err))
		}
	} else {
		application.Run()
	}
}

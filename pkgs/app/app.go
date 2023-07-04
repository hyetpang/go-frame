/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-05-09 10:08:41
 * @FilePath: /go-frame/pkgs/app/app.go
 */
package app

import (
	"context"
	"time"

	"github.com/jpillora/overseer"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type App struct {
	options []fx.Option
	isStart bool
}

func (app *App) runWith(state overseer.State) {
	app.options = append(app.options, fx.Provide(func() overseer.State {
		return state
	}))
	app.run(false)
}

func (app *App) run(isStart bool) {
	viper.Debug() // 打印配置项
	application := fx.New(app.options...)
	if isStart {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		application.Start(ctx)
	} else {
		application.Run()
	}
}

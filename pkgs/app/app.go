/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-05-09 10:08:41
 * @FilePath: /go-frame/pkgs/app/app.go
 */
package app

import (
	"github.com/HyetPang/go-frame/pkgs/dev"
	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/HyetPang/overseer"
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
	app.run()
}

func (app *App) run() {
	if dev.IsDebug {
		viper.Debug()
	}
	logs.Info("start_running 开始启动程序...")
	fx.New(app.options...).Run()
}

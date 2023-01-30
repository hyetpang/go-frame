/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-05-09 10:08:41
 * @FilePath: /go-frame/pkgs/app/app.go
 */
package app

import (
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
	app.run()
}

func (app *App) run() {
	viper.Debug() // 打印配置项
	fx.New(app.options...).Run()
}

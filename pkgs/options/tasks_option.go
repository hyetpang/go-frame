package options

import (
	"github.com/hyetpang/go-frame/internal/components/tasks"
	"go.uber.org/fx"
)

// 定时任务
func WithTasks() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(tasks.New))
	}
}

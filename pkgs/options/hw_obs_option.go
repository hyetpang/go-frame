package options

import (
	"github.com/hyetpang/go-frame/internal/components/obs"
	"go.uber.org/fx"
)

// 华为obs,这里返回的是一个interfaces.OBSInterface的接口类型,HwObs实现了这个接口
func WithHwOBS() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(obs.NewHw))
	}
}

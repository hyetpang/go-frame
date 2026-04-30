package options

import (
	"time"

	"github.com/guonaihong/gout"
	frameconfig "github.com/hyetpang/go-frame/internal/config"
	"go.uber.org/fx"
)

// 指定github.com/guonaihong/gout这个库的全局配置: 是否全局开启调试,设置全局的超时
func WithGoutConfig() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Invoke(func(lc fx.Lifecycle, conf *frameconfig.Gout) {
			lc.Append(fx.StartHook(func() {
				if conf.Debug {
					gout.SetDebug(conf.Debug)
				}
				if conf.Timeout > 0 {
					gout.SetTimeout(time.Second * time.Duration(conf.Timeout))
				}
			}))
		}))
	}
}

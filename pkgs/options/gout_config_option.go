package options

import (
	"log"
	"time"

	"github.com/guonaihong/gout"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type goutConfig struct {
	Debug   bool `mapstructure:"debug"`
	Timeout int  `mapstructure:"timeout"`
}

// 指定github.com/guonaihong/gout这个库的全局配置: 是否全局开启调试,设置全局的超时
func WithGoutConfig() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Invoke(func(lc fx.Lifecycle) {
			lc.Append(fx.StartHook(func() {
				conf := new(goutConfig)
				err := viper.UnmarshalKey("gout", conf)
				if err != nil {
					log.Fatalf("Unmarshal gout配置出错:%s", err.Error())
				}
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

package options

import (
	"log"
	"time"

	"github.com/guonaihong/gout"
	"github.com/spf13/viper"
)

type goutConfig struct {
	GoutDebug   bool `mapstructure:"gout_debug"`
	GoutTimeout int  `mapstructure:"gout_timeout"`
}

// 指定github.com/guonaihong/gout这个库的全局配置: 是否全局开启调试,设置全局的超时
func WithGoutConfig() Option {
	return func(o *Options) {
		conf := new(goutConfig)
		err := viper.UnmarshalKey("gout", conf)
		if err != nil {
			log.Fatalf("Unmarshal gout配置出错:%s", err.Error())
		}
		if conf.GoutDebug {
			gout.SetDebug(conf.GoutDebug)
		}
		if conf.GoutTimeout > 0 {
			gout.SetTimeout(time.Second * time.Duration(conf.GoutTimeout))
		}
	}
}

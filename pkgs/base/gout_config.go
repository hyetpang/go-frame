package base

import (
	"time"

	"github.com/guonaihong/gout"
)

// 使用方法:
// 1. 直接嵌入，没有给提供名字，那么需要如下tags `mapstructure:",squash"`
// 例如:
// type ServerConfig struct {
// 	RunMode               string `mapstructure:"run_mode" validate:"required"`
// 	base.GoutConfig       `mapstructure:",squash"`
// }
// 这时候可以这样配置
// [server]
// run_mode = "dev"                                                 # 运行模式,可选的值是dev,online
// gout_debug = true                                                # 是否开启github.com/guonaihong/gout库的全局debug
// gout_timeout = 5
// 2. 提供了字段名，那么在配置时候需要指定名字
// type ServerConfig struct {
// 	RunMode               string `mapstructure:"run_mode" validate:"required"`
// 	GoutConf base.GoutConfig       `mapstructure:"gout_conf"`
// }
// 这时候可以这样配置
// [server]
// run_mode = "dev"                                                 # 运行模式,可选的值是dev,online
// gout_conf.gout_debug = true                                                # 是否开启github.com/guonaihong/gout库的全局debug
// gout_conf.gout_timeout = 5
type GoutConfig struct {
	GoutDebug   bool `mapstructure:"gout_debug"`
	GoutTimeout int  `mapstructure:"gout_timeout"`
}

// 指定了响应配置需要生效的方法
func (goutConfig *GoutConfig) Config() {
	if goutConfig.GoutDebug {
		gout.SetDebug(true)
	}
	if goutConfig.GoutTimeout > 0 {
		gout.SetTimeout(time.Second * time.Duration(goutConfig.GoutTimeout))
	}
}

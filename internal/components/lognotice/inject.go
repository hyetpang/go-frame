package lognotice

import (
	"log"

	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/interfaces"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// 错误日志通知返回具体实例
func New(lc fx.Lifecycle) interfaces.LogNoticeInterface {
	conf := new(config)
	err := viper.UnmarshalKey("log_notice", &conf)
	if err != nil {
		log.Fatalf("log_notice配置Unmarshal到对象出错: %s", err.Error())
	}
	common.MustValidate(conf)
	n := newNotice(conf, lc)
	return n
}

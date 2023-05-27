package lognotice

import (
	"log"

	"github.com/HyetPang/go-frame/pkgs/common"
	"github.com/HyetPang/go-frame/pkgs/interfaces"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// 错误日志通知返回具体实例
func New(lc fx.Lifecycle) interfaces.LogNoticeInterface {
	conf := new(config)
	err := viper.UnmarshalKey("log_notice", &conf)
	if err != nil {
		log.Fatal("log_notice配置Unmarshal到对象出错", zap.Error(err))
	}
	common.MustValidate(conf)
	n := newNotice(conf, lc)
	return n
}

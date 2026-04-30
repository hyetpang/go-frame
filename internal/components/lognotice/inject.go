package lognotice

import (
	"fmt"

	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/interfaces"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// 错误日志通知返回具体实例
func New(lc fx.Lifecycle) (interfaces.LogNoticeInterface, error) {
	conf := new(config)
	err := viper.UnmarshalKey("log_notice", &conf)
	if err != nil {
		return nil, fmt.Errorf("log_notice配置Unmarshal到对象出错: %w", err)
	}
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("log_notice配置验证不通过: %w", err)
	}
	n, err := newNotice(conf, lc)
	if err != nil {
		return nil, err
	}
	return n, nil
}

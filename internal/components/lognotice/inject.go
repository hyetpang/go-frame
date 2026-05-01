package lognotice

import (
	"fmt"

	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/lognotice"
	"go.uber.org/fx"
)

// New 创建错误日志通知器实例,接口类型来自 pkgs/lognotice.Notifier。
func New(lc fx.Lifecycle, conf *config) (lognotice.Notifier, error) {
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("log_notice配置验证不通过: %w", err)
	}
	n, err := newNotice(conf, lc)
	if err != nil {
		return nil, err
	}
	return n, nil
}

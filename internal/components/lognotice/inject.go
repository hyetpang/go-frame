package lognotice

import (
	"fmt"

	"github.com/hyetpang/go-frame/pkgs/common"
	lognoticepkg "github.com/hyetpang/go-frame/pkgs/lognotice"
	"go.uber.org/fx"
)

// New 错误日志通知返回具体实例。
func New(lc fx.Lifecycle, conf *config) (lognoticepkg.Notifier, error) {
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("log_notice配置验证不通过: %w", err)
	}
	n, err := newNotice(conf, lc)
	if err != nil {
		return nil, err
	}
	return n, nil
}

package grpc

import (
	"fmt"

	frameconfig "github.com/hyetpang/go-frame/internal/config"
	"github.com/hyetpang/go-frame/pkgs/common"
)

type config = frameconfig.GRPC

// 初始化config
func newConfig(conf *config) (*config, error) {
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("grpc配置验证不通过: %w", err)
	}
	if len(conf.ServicePrefix) <= 0 {
		conf.ServicePrefix = "grpc_services"
	}
	return conf, nil
}

package grpc

import (
	"fmt"

	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/spf13/viper"
)

// grpc 配置
type config struct {
	Address       string   `mapstructure:"address"`        // 监听地址
	ServicePrefix string   `mapstructure:"service_prefix"` // 服务前缀
	ServiceNames  []string `mapstructure:"service_names"`  // 服务名字
}

// 初始化config
func newConfig() (*config, error) {
	conf := new(config)
	err := viper.UnmarshalKey("grpc", &conf)
	if err != nil {
		return nil, fmt.Errorf("grpc配置Unmarshal到对象出错: %w", err)
	}
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("grpc配置验证不通过: %w", err)
	}
	if len(conf.ServicePrefix) <= 0 {
		conf.ServicePrefix = "grpc_services"
	}
	return conf, nil
}

package grpc

import (
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// grpc 配置
type config struct {
	Address       string   `mapstructure:"address"`        // 监听地址
	ServicePrefix string   `mapstructure:"service_prefix"` // 服务前缀
	ServiceNames  []string `mapstructure:"service_names"`  // 服务名字
}

const defaultServicePrefix = "grpc_services"

// 初始化config
func newConfig() *config {
	conf := new(config)
	err := viper.UnmarshalKey("grpc", &conf)
	if err != nil {
		logs.Fatal("kafka配置Unmarshal到对象出错", zap.Error(err), zap.Any("conf", conf))
	}
	common.MustValidate(conf)
	return conf
}

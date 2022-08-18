package app

import (
	"log"
	"os"

	"github.com/HyetPang/go-frame/pkgs/validate"
	"github.com/spf13/viper"
)

// 优雅重启配置
type graceRestartConfig struct {
	HttpAddr       string `mapstructure:"http_addr" validate:"required"`        // http监听地址
	ExecFile       string `mapstructure:"exec_file" validate:"required"`        // 可执行文件路径
	ExecLatestFile string `mapstructure:"exec_latest_file" validate:"required"` // 最新可执行文件路径
}

func newGraceRestartConfig() *graceRestartConfig {
	conf := new(graceRestartConfig)
	err := viper.UnmarshalKey("graceful_restart", conf)
	if err != nil {
		log.Fatalf("graceful_restart配置Unmarshal到对象出错:%s", err.Error())
	}
	validate.Must(conf)
	// 验证路径是否有效
	_, err = os.Stat(conf.ExecFile)
	if err != nil {
		log.Fatalf("graceful_restart配置的exec_file出错:%s", err.Error())
	}
	return conf
}

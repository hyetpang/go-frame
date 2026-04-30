package app

import (
	"os"

	frameconfig "github.com/hyetpang/go-frame/internal/config"
)

// 优雅重启配置
type graceRestartConfig = frameconfig.GraceRestart

func newGraceRestartConfig(conf *graceRestartConfig) error {
	// 验证路径是否有效
	_, err := os.Stat(conf.ExecFile)
	if err != nil {
		return err
	}
	return nil
}

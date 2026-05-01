package etcd

import (
	"fmt"
	"strings"
	"time"

	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(zapLog *zap.Logger, lc fx.Lifecycle, conf *config) (*clientv3.Client, error) {
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("etcd配置验证不通过: %w", err)
	}
	tlsCfg, err := conf.TLS.BuildClientTLS()
	if err != nil {
		return nil, fmt.Errorf("构建 etcd TLS 配置出错: %w", err)
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            strings.Split(conf.Addresses, ","),
		AutoSyncInterval:     0,
		DialTimeout:          time.Second * 5,
		DialKeepAliveTime:    time.Second * 3,
		DialKeepAliveTimeout: time.Second * 5,
		MaxCallSendMsgSize:   0,
		MaxCallRecvMsgSize:   0,
		RejectOldCluster:     false,
		Logger:               zapLog,
		PermitWithoutStream:  false,
		Username:             conf.Username,
		Password:             conf.Password,
		TLS:                  tlsCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("创建etcd客户端出错 addresses=%s: %w", conf.Addresses, err)
	}
	lc.Append(fx.StopHook(func() {
		if e := cli.Close(); e != nil {
			logs.Error("关闭etcd客户端出错", zap.Error(e))
		}
	}))
	return cli, nil
}

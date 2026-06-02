package etcd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hyetpang/go-frame/internal/lifecycle"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// etcd 客户端拨号相关超时常量。
// 说明:Etcd 配置结构位于 internal/config 包,本次不在可改动范围内,
// 故暂以命名常量集中管理这些原本散落写死的超时值;后续若需开放为配置项,
// 可在 config.Etcd 上增补字段并在此读取覆盖。
const (
	defaultDialTimeout          = time.Second * 5 // 建立连接的拨号超时
	defaultDialKeepAliveTime    = time.Second * 3 // 发送 keepalive 探测的间隔
	defaultDialKeepAliveTimeout = time.Second * 5 // 等待 keepalive 响应的超时
)

// parseEndpoints 解析逗号分隔的地址串,对每个 endpoint 做 TrimSpace 并过滤空串,
// 避免 "a, b" 这类带空格的配置产生非法 endpoint。
func parseEndpoints(addresses string) []string {
	parts := strings.Split(addresses, ",")
	endpoints := make([]string, 0, len(parts))
	for _, p := range parts {
		if ep := strings.TrimSpace(p); ep != "" {
			endpoints = append(endpoints, ep)
		}
	}
	return endpoints
}

func New(zapLog *zap.Logger, lc fx.Lifecycle, conf *config) (*clientv3.Client, error) {
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("etcd配置验证不通过: %w", err)
	}
	tlsCfg, err := conf.TLS.BuildClientTLS()
	if err != nil {
		return nil, fmt.Errorf("构建 etcd TLS 配置出错: %w", err)
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            parseEndpoints(conf.Addresses),
		AutoSyncInterval:     0,
		DialTimeout:          defaultDialTimeout,
		DialKeepAliveTime:    defaultDialKeepAliveTime,
		DialKeepAliveTimeout: defaultDialKeepAliveTimeout,
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
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if e := lifecycle.CloseWithContext(ctx, "etcd", cli.Close); e != nil {
				logs.Error("关闭etcd客户端出错", zap.Error(e))
				return e
			}
			return nil
		},
	})
	return cli, nil
}

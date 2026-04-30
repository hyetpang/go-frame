package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hyetpang/go-frame/pkgs/logs"
	clientv3 "go.etcd.io/etcd/client/v3"
	resolver "go.etcd.io/etcd/client/v3/naming/resolver"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// 使用etcd作为服务发现
func NewServerEtcd(lc fx.Lifecycle, zapLog *zap.Logger, etcdClient *clientv3.Client, conf *config) (*grpc.Server, error) {
	s, lis, conf, err := newServer(zapLog, conf)
	if err != nil {
		return nil, err
	}
	serviceNamePrefix := conf.ServicePrefix
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			errCh := make(chan error, 1)
			go func() {
				if e := s.Serve(lis); e != nil {
					errCh <- e
				}
			}()
			select {
			case e := <-errCh:
				cancel()
				return fmt.Errorf("启动grpc serve出错: %w", e)
			case <-startCtx.Done():
				cancel()
				return startCtx.Err()
			case <-time.After(time.Second):
				serviceNames := conf.ServiceNames
				if len(serviceNames) == 0 {
					serviceNames = make([]string, 0, len(s.GetServiceInfo()))
					for name := range s.GetServiceInfo() {
						serviceNames = append(serviceNames, name)
					}
				}
				for _, serviceName := range serviceNames {
					if err := etcdRegisterService(ctx, serviceNamePrefix, serviceName, conf.Address, etcdClient); err != nil {
						cancel()
						return fmt.Errorf("注册服务出错 %s: %w", serviceName, err)
					}
					logs.Info("注册GRPC服务", zap.String("服务名", serviceName))
				}
				logs.Debug("grpc start success", zap.String("address", conf.Address))
				return nil
			}
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			gracefulStopServer(stopCtx, s)
			return nil
		},
	})
	return s, nil
}

// 使用etcd作为服务发现
func NewClientEtcd(lc fx.Lifecycle, zapLog *zap.Logger, etcdClient *clientv3.Client, conf *config) (map[string]*grpc.ClientConn, error) {
	conf, err := newConfig(conf)
	if err != nil {
		return nil, err
	}
	if len(conf.ServiceNames) < 1 {
		return nil, errors.New("grpc client 必须配置一个服务名字")
	}
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		return nil, fmt.Errorf("创建etcd服务解析器对象出错: %w", err)
	}
	clients := make(map[string]*grpc.ClientConn, len(conf.ServiceNames))
	for _, serviceName := range conf.ServiceNames {
		conn, err := newClient(etcdTarget(etcdResolver.Scheme(), conf.ServicePrefix, serviceName), lc, zapLog, etcdResolver)
		if err != nil {
			return nil, fmt.Errorf("创建grpc客户端 %s 失败: %w", serviceName, err)
		}
		clients[serviceName] = conn
	}
	return clients, nil
}

func etcdTarget(scheme, servicePrefix, serviceName string) string {
	return fmt.Sprintf("%s:///%s/%s", scheme, servicePrefix, serviceName)
}

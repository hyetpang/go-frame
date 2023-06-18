package grpc

import (
	"context"
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
func NewServerEtcd(lc fx.Lifecycle, zapLog *zap.Logger, etcdClient *clientv3.Client) *grpc.Server {
	s, lis, conf := newServer(zapLog)
	serviceNamePrefix := defaultServicePrefix
	if len(conf.ServicePrefix) > 0 {
		serviceNamePrefix = conf.ServicePrefix
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 开始处理
			err := make(chan error, 1)
			go func() {
				if e := s.Serve(lis); e != nil {
					logs.Error("grpc监听出错", zap.Error(e))
					err <- e
				}
			}()
			select {
			case e := <-err:
				logs.Fatal("启动grpc serve出错", zap.Error(e))
				return e
			case <-ctx.Done():
				logs.Fatal("启动grpc serve超时", zap.Error(ctx.Err()))
				return ctx.Err()
			case <-time.After(time.Second):
				if len(conf.ServiceNames) > 0 {
					for _, serviceName := range conf.ServiceNames {
						// 服务注册
						err := etcdRegisterService(context.TODO(), serviceNamePrefix, serviceName, conf.Address, etcdClient)
						if err != nil {
							logs.Fatal("注册服务出错", zap.Error(err))
						}
						logs.Info("注册GRPC服务", zap.String("服务名", serviceName))
					}
				} else {
					for serviceName := range s.GetServiceInfo() {
						// 服务注册
						err := etcdRegisterService(context.TODO(), serviceNamePrefix, serviceName, conf.Address, etcdClient)
						if err != nil {
							logs.Fatal("注册服务出错", zap.Error(err))
						}
						logs.Info("注册GRPC服务", zap.String("服务名", serviceName))
					}
				}
				logs.Debug("grpc start success", zap.String("address", conf.Address))
				return nil
			}
		},
		OnStop: func(ctx context.Context) error {
			// 关闭服务
			s.GracefulStop()
			return nil
		},
	})
	return s
}

// 使用etcd作为服务发现
func NewClientEtcd(lc fx.Lifecycle, zapLog *zap.Logger, etcdClient *clientv3.Client) map[string]*grpc.ClientConn {
	conf := newConfig()
	if len(conf.ServiceNames) < 1 {
		logs.Fatal("grpc client 必须配置一个服务名字!")
	}
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		logs.Fatal("创建etcd服务解析器对象出错", zap.Error(err))
	}
	serviceNamePrefix := defaultServicePrefix
	if len(conf.ServicePrefix) > 0 {
		serviceNamePrefix = conf.ServicePrefix
	}
	clients := make(map[string]*grpc.ClientConn, len(conf.ServiceNames))
	for _, serviceName := range conf.ServiceNames {
		conn := newClient(fmt.Sprintf("%s:///%s/%s", etcdResolver.Scheme(), serviceNamePrefix, serviceName), lc, zapLog, etcdResolver)
		clients[serviceName] = conn
	}
	return clients
}

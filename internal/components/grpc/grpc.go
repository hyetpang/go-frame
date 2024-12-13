package grpc

import (
	"context"
	"net"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	gresolver "google.golang.org/grpc/resolver"
)

// 不使用服务发现
func NewServer(lc fx.Lifecycle, zapLog *zap.Logger) *grpc.Server {
	s, lis, conf := newServer(zapLog)
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
				for serviceName, serviceInfo := range s.GetServiceInfo() {
					zap.L().Info("注册GRPC服务", zap.String("服务名", serviceName), zap.Any("Metadata", serviceInfo.Metadata))
				}
				logs.Debug("grpc start success", zap.String("address", conf.Address))
				return nil
			}
		},
		OnStop: func(ctx context.Context) error {
			s.GracefulStop()
			return nil
		},
	})
	return s
}

// 不使用服务发现
func NewClient(lc fx.Lifecycle, zapLog *zap.Logger) *grpc.ClientConn {
	conf := newConfig()
	if len(conf.Address) < 1 {
		// 监听地址没填
		logs.Error("grpc配置监听地址必填")
	}
	return newClient(conf.Address, lc, zapLog, nil)
}

func newServer(zapLog *zap.Logger) (*grpc.Server, net.Listener, *config) {
	conf := newConfig()
	if len(conf.Address) < 1 {
		// 监听地址没填
		logs.Error("grpc监听地址必填")
	}
	// 创建grpc server
	s := grpc.NewServer(grpc.UnaryInterceptor(
		grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(zapLog, grpc_zap.WithLevels(grpc_zap.DefaultCodeToLevel)),
			grpc_recovery.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_opentracing.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(zapLog, grpc_zap.WithLevels(grpc_zap.DefaultCodeToLevel)),
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     time.Second * 180,
			MaxConnectionAge:      time.Hour * 2,
			MaxConnectionAgeGrace: time.Second * 20,
			Time:                  time.Second * 60,
			Timeout:               time.Second * 20,
		}))
	// 监听端口
	lis, err := net.Listen("tcp", conf.Address)
	if err != nil {
		logs.Fatal("failed to listen: %v", zap.Error(err))
	}
	// 注册服务
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	return s, lis, conf
}

func newClient(addr string, lc fx.Lifecycle, zapLog *zap.Logger, grpcResolver gresolver.Builder) *grpc.ClientConn {
	options := make([]grpc.DialOption, 0, 10)
	options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // 轮询负载均衡,https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/client/main.go
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Second * 60,
			Timeout:             time.Second * 20,
			PermitWithoutStream: false,
		}),
		grpc.WithChainStreamInterceptor(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
			grpc_zap.StreamClientInterceptor(zapLog, grpc_zap.WithLevels(grpc_zap.DefaultClientCodeToLevel)),
		),
		grpc.WithChainUnaryInterceptor(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
			grpc_zap.UnaryClientInterceptor(zapLog, grpc_zap.WithLevels(grpc_zap.DefaultClientCodeToLevel)),
		))
	if grpcResolver != nil {
		options = append(options, grpc.WithResolvers(grpcResolver))
	}
	conn, err := grpc.Dial(addr, options...)
	if err != nil {
		logs.Fatal("创建连接出错", zap.Error(err))
	}
	lc.Append(fx.StopHook(func() error {
		return conn.Close()
	}))
	return conn
}

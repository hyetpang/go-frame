package grpc

import (
	"context"
	"errors"
	"fmt"
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
func NewServer(lc fx.Lifecycle, zapLog *zap.Logger) (*grpc.Server, error) {
	s, lis, conf, err := newServer(zapLog)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			errCh := make(chan error, 1)
			go func() {
				if e := s.Serve(lis); e != nil {
					errCh <- e
				}
			}()
			select {
			case e := <-errCh:
				return fmt.Errorf("启动grpc serve出错: %w", e)
			case <-ctx.Done():
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
	return s, nil
}

// 不使用服务发现
func NewClient(lc fx.Lifecycle, zapLog *zap.Logger) (*grpc.ClientConn, error) {
	conf := newConfig()
	if len(conf.Address) < 1 {
		return nil, errors.New("grpc客户端必须配置监听地址")
	}
	return newClient(conf.Address, lc, zapLog, nil)
}

func newServer(zapLog *zap.Logger) (*grpc.Server, net.Listener, *config, error) {
	conf := newConfig()
	if len(conf.Address) < 1 {
		return nil, nil, nil, errors.New("grpc监听地址必填")
	}
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
	lis, err := net.Listen("tcp", conf.Address)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("监听地址出错 %s: %w", conf.Address, err)
	}
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	return s, lis, conf, nil
}

func newClient(addr string, lc fx.Lifecycle, zapLog *zap.Logger, grpcResolver gresolver.Builder) (*grpc.ClientConn, error) {
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
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
		),
	}
	if grpcResolver != nil {
		options = append(options, grpc.WithResolvers(grpcResolver))
	}
	conn, err := grpc.NewClient(addr, options...)
	if err != nil {
		return nil, fmt.Errorf("创建grpc连接出错: %w", err)
	}
	lc.Append(fx.StopHook(func() error {
		return conn.Close()
	}))
	return conn, nil
}

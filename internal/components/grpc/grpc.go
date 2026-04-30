package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/hyetpang/go-frame/pkgs/logs"
	prometheus_client "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	gresolver "google.golang.org/grpc/resolver"
)

const grpcGracefulStopTimeout = 5 * time.Second

var (
	serverMetrics       = grpc_prometheus.NewServerMetrics()
	clientMetrics       = grpc_prometheus.NewClientMetrics()
	registerMetricsOnce sync.Once
)

// 不使用服务发现
func NewServer(lc fx.Lifecycle, zapLog *zap.Logger, conf *config) (*grpc.Server, error) {
	s, lis, conf, err := newServer(zapLog, conf)
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
			gracefulStopServer(ctx, s)
			return nil
		},
	})
	return s, nil
}

// 不使用服务发现
func NewClient(lc fx.Lifecycle, zapLog *zap.Logger, conf *config) (*grpc.ClientConn, error) {
	conf, err := newConfig(conf)
	if err != nil {
		return nil, err
	}
	if len(conf.Address) < 1 {
		return nil, errors.New("grpc客户端必须配置监听地址")
	}
	return newClient(conf.Address, lc, zapLog, nil)
}

func newServer(zapLog *zap.Logger, conf *config) (*grpc.Server, net.Listener, *config, error) {
	conf, err := newConfig(conf)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(conf.Address) < 1 {
		return nil, nil, nil, errors.New("grpc监听地址必填")
	}
	registerGRPCMetrics()
	logger := grpcLogger(zapLog)
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(
		serverMetrics.UnaryServerInterceptor(),
		grpc_logging.UnaryServerInterceptor(logger, grpc_logging.WithLevels(grpc_logging.DefaultServerCodeToLevel)),
		grpc_recovery.UnaryServerInterceptor(),
	),
		grpc.ChainStreamInterceptor(
			serverMetrics.StreamServerInterceptor(),
			grpc_logging.StreamServerInterceptor(logger, grpc_logging.WithLevels(grpc_logging.DefaultServerCodeToLevel)),
			grpc_recovery.StreamServerInterceptor(),
		),
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
	serverMetrics.InitializeMetrics(s)
	return s, lis, conf, nil
}

func newClient(addr string, lc fx.Lifecycle, zapLog *zap.Logger, grpcResolver gresolver.Builder) (*grpc.ClientConn, error) {
	registerGRPCMetrics()
	logger := grpcLogger(zapLog)
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Second * 60,
			Timeout:             time.Second * 20,
			PermitWithoutStream: false,
		}),
		grpc.WithChainStreamInterceptor(
			clientMetrics.StreamClientInterceptor(),
			grpc_logging.StreamClientInterceptor(logger, grpc_logging.WithLevels(grpc_logging.DefaultClientCodeToLevel)),
		),
		grpc.WithChainUnaryInterceptor(
			clientMetrics.UnaryClientInterceptor(),
			grpc_logging.UnaryClientInterceptor(logger, grpc_logging.WithLevels(grpc_logging.DefaultClientCodeToLevel)),
		),
	}
	if grpcResolver != nil {
		options = append(options, grpc.WithResolvers(grpcResolver))
	}
	conn, err := grpc.NewClient(addr, options...)
	if err != nil {
		return nil, fmt.Errorf("创建grpc连接出错: %w", err)
	}
	lc.Append(fx.StartHook(func() {
		conn.Connect()
	}))
	lc.Append(fx.StopHook(func() error {
		return conn.Close()
	}))
	return conn, nil
}

func gracefulStopServer(ctx context.Context, server *grpc.Server) {
	timeout := grpcGracefulStopTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}
	gracefulStopWithTimeout(server.GracefulStop, server.Stop, timeout)
}

func gracefulStopWithTimeout(gracefulStop func(), forceStop func(), timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		gracefulStop()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		forceStop()
	}
}

func grpcLogger(logger *zap.Logger) grpc_logging.Logger {
	return grpc_logging.LoggerFunc(func(ctx context.Context, level grpc_logging.Level, msg string, fields ...any) {
		zapFields := make([]zap.Field, 0, len(fields)/2)
		for i := 0; i+1 < len(fields); i += 2 {
			key, ok := fields[i].(string)
			if !ok {
				continue
			}
			zapFields = append(zapFields, zap.Any(key, fields[i+1]))
		}
		switch level {
		case grpc_logging.LevelDebug:
			logger.Debug(msg, zapFields...)
		case grpc_logging.LevelInfo:
			logger.Info(msg, zapFields...)
		case grpc_logging.LevelWarn:
			logger.Warn(msg, zapFields...)
		default:
			logger.Error(msg, zapFields...)
		}
	})
}

func registerGRPCMetrics() {
	registerMetricsOnce.Do(func() {
		registerCollector(serverMetrics)
		registerCollector(clientMetrics)
	})
}

func registerCollector(collector prometheus_client.Collector) {
	if err := prometheus_client.Register(collector); err != nil {
		if _, ok := err.(prometheus_client.AlreadyRegisteredError); !ok {
			logs.Warn("注册grpc prometheus指标出错", zap.Error(err))
		}
	}
}

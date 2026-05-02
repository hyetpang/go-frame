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
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	gresolver "google.golang.org/grpc/resolver"
)

const (
	grpcGracefulStopTimeout = 5 * time.Second
	// grpcReadyTimeout 为启动期 self-dial 健康检查总超时,避免长时间阻塞 fx 启动
	grpcReadyTimeout = 200 * time.Millisecond
)

var (
	serverMetrics       = grpc_prometheus.NewServerMetrics()
	clientMetrics       = grpc_prometheus.NewClientMetrics()
	registerMetricsOnce sync.Once
	// clientTargetReady 暴露每个 grpc client target 启动期健康检查结果,
	// 标签 status=success 表示可达、status=fail 表示启动期不可达(不阻断启动,
	// 仅用作可观测,便于发现"目标服务一个都没注册"等环境问题)。
	clientTargetReady = prometheus_client.NewCounterVec(
		prometheus_client.CounterOpts{
			Name: "grpc_client_target_ready_total",
			Help: "grpc client startup-time health check result per target",
		},
		[]string{"target", "status"},
	)
)

const (
	// grpcClientReadyCheckTimeout 启动期对每个 client target 做一次最大等待时间。
	// 失败不阻塞启动,只记录指标 + warn 日志。
	grpcClientReadyCheckTimeout = 500 * time.Millisecond
	// 默认 keepalive / 消息体上限,与 grpc-go 默认对齐。
	defaultKeepaliveTimeSec    = 60
	defaultKeepaliveTimeoutSec = 20
	defaultMaxConnIdleSec      = 180
	defaultMaxConnAgeSec       = 7200 // 2h
)

// 不使用服务发现
func NewServer(lc fx.Lifecycle, zapLog *zap.Logger, conf *config) (*grpc.Server, error) {
	s, lis, conf, err := newServer(zapLog, conf)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// errCh 长期保留,启动期由 OnStart 监听,启动成功后由后台 goroutine 继续监听 Serve 异常退出
			errCh := make(chan error, 1)
			go func() {
				if e := s.Serve(lis); e != nil {
					errCh <- e
				}
			}()
			if err := waitGRPCServerReady(ctx, lis.Addr().String(), errCh); err != nil {
				return err
			}
			// 启动成功后将 errCh 的监听权转交给后台 goroutine,避免 Serve 长时间运行后失败的错误被吞没
			go watchGRPCServeError(errCh, lis.Addr().String())
			for serviceName, serviceInfo := range s.GetServiceInfo() {
				zap.L().Info("注册GRPC服务", zap.String("服务名", serviceName), zap.Any("Metadata", serviceInfo.Metadata))
			}
			logs.Debug("grpc start success", zap.String("address", conf.Address))
			return nil
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
	creds, err := buildClientCreds(&conf.ClientTLS)
	if err != nil {
		return nil, err
	}
	return newClient(conf.Address, conf.Address, conf, lc, zapLog, nil, creds)
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
	serverOpts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
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
			MaxConnectionIdle:     secondsOrDefault(conf.MaxConnectionIdleSec, defaultMaxConnIdleSec),
			MaxConnectionAge:      secondsOrDefault(conf.MaxConnectionAgeSec, defaultMaxConnAgeSec),
			MaxConnectionAgeGrace: time.Second * 20,
			Time:                  secondsOrDefault(conf.KeepaliveTimeSec, defaultKeepaliveTimeSec),
			Timeout:               secondsOrDefault(conf.KeepaliveTimeoutSec, defaultKeepaliveTimeoutSec),
		}),
	}
	if conf.MaxRecvMsgMB > 0 {
		serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(conf.MaxRecvMsgMB*1024*1024))
	}
	if conf.MaxSendMsgMB > 0 {
		serverOpts = append(serverOpts, grpc.MaxSendMsgSize(conf.MaxSendMsgMB*1024*1024))
	}
	if conf.ServerTLS.IsEnabled() {
		tlsCfg, terr := conf.ServerTLS.BuildServerTLS()
		if terr != nil {
			return nil, nil, nil, fmt.Errorf("构建 grpc server TLS 配置出错: %w", terr)
		}
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}
	s := grpc.NewServer(serverOpts...)
	lis, err := net.Listen("tcp", conf.Address)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("监听地址出错 %s: %w", conf.Address, err)
	}
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	serverMetrics.InitializeMetrics(s)
	return s, lis, conf, nil
}

// buildClientCreds 根据配置返回 grpc 客户端使用的 TransportCredentials。
// 未启用 TLS 时回退到 insecure,保持向后兼容。
func buildClientCreds(tlsConf *frameTLSConfig) (credentials.TransportCredentials, error) {
	if tlsConf == nil || !tlsConf.IsEnabled() {
		return insecure.NewCredentials(), nil
	}
	cfg, err := tlsConf.BuildClientTLS()
	if err != nil {
		return nil, fmt.Errorf("构建 grpc client TLS 配置出错: %w", err)
	}
	return credentials.NewTLS(cfg), nil
}

func newClient(addr, target string, conf *config, lc fx.Lifecycle, zapLog *zap.Logger, grpcResolver gresolver.Builder, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
	if creds == nil {
		creds = insecure.NewCredentials()
	}
	registerGRPCMetrics()
	logger := grpcLogger(zapLog)
	callOpts := []grpc.CallOption{}
	if conf != nil && conf.MaxRecvMsgMB > 0 {
		callOpts = append(callOpts, grpc.MaxCallRecvMsgSize(conf.MaxRecvMsgMB*1024*1024))
	}
	if conf != nil && conf.MaxSendMsgMB > 0 {
		callOpts = append(callOpts, grpc.MaxCallSendMsgSize(conf.MaxSendMsgMB*1024*1024))
	}
	keepaliveTime := defaultKeepaliveTimeSec
	keepaliveTimeout := defaultKeepaliveTimeoutSec
	if conf != nil {
		if conf.KeepaliveTimeSec > 0 {
			keepaliveTime = conf.KeepaliveTimeSec
		}
		if conf.KeepaliveTimeoutSec > 0 {
			keepaliveTimeout = conf.KeepaliveTimeoutSec
		}
	}
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Duration(keepaliveTime) * time.Second,
			Timeout:             time.Duration(keepaliveTimeout) * time.Second,
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
	if len(callOpts) > 0 {
		options = append(options, grpc.WithDefaultCallOptions(callOpts...))
	}
	if grpcResolver != nil {
		options = append(options, grpc.WithResolvers(grpcResolver))
	}
	conn, err := grpc.NewClient(addr, options...)
	if err != nil {
		return nil, fmt.Errorf("创建grpc连接出错: %w", err)
	}
	if target == "" {
		target = addr
	}
	lc.Append(fx.StartHook(func() {
		conn.Connect()
		// 启动期最大 grpcClientReadyCheckTimeout 内做一次健康探测,失败不阻断启动:
		// 仅在 metric/日志暴露,便于发现"目标服务一个都没注册"等环境问题。
		go probeClientTargetReady(conn, target)
	}))
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// grpc.ClientConn.Close 通常 instant,但极端阻塞场景下也用 ctx 兜底。
			done := make(chan error, 1)
			go func() { done <- conn.Close() }()
			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return fmt.Errorf("关闭 grpc client conn 超时: %w", ctx.Err())
			}
		},
	})
	return conn, nil
}

// waitGRPCServerReady 通过本地 self-dial Health 服务来确认 grpc server 已经能接受请求,
// 替代旧版"睡 1 秒就算成功"的伪健康检查。errCh 中的错误优先返回。
func waitGRPCServerReady(ctx context.Context, addr string, errCh <-chan error) error {
	// 先快速检查 Serve 是否已经直接报错(端口被抢占等)
	select {
	case e, ok := <-errCh:
		if ok && e != nil {
			return fmt.Errorf("启动grpc serve出错: %w", e)
		}
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	dialCtx, cancel := context.WithTimeout(ctx, grpcReadyTimeout)
	defer cancel()
	// 直连本地监听地址,绕过 dns 解析
	conn, err := grpc.NewClient("passthrough:///"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("self-dial grpc server %s 失败: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()
	conn.Connect()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	resp, err := healthClient.Check(dialCtx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		// health 检查失败时再确认是否是 Serve 直接挂掉
		select {
		case e, ok := <-errCh:
			if ok && e != nil {
				return fmt.Errorf("启动grpc serve出错: %w", e)
			}
		default:
		}
		return fmt.Errorf("grpc health 检查失败: %w", err)
	}
	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("grpc health 状态异常: %s", resp.GetStatus())
	}
	return nil
}

// watchGRPCServeError 在启动成功后持续监听 Serve goroutine 的错误,
// 避免 Serve 长时间运行后异常退出导致错误被吞没。
func watchGRPCServeError(errCh <-chan error, addr string) {
	if e, ok := <-errCh; ok && e != nil {
		logs.Error("grpc Serve 异常退出", zap.String("address", addr), zap.Error(e))
	}
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
		return
	case <-timer.C:
	}
	// 超时后强制关闭。grpc.Server.Stop 会唤醒 GracefulStop,
	// 必须等 graceful goroutine 真正退出再返回,避免悬挂 goroutine。
	forceStop()
	<-done
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
		registerCollector(clientTargetReady)
	})
}

// secondsOrDefault 在配置值 <= 0 时回退默认值,以秒为单位返回 Duration。
func secondsOrDefault(seconds, defaultSec int) time.Duration {
	if seconds <= 0 {
		return time.Duration(defaultSec) * time.Second
	}
	return time.Duration(seconds) * time.Second
}

// probeClientTargetReady 启动期对一个 client conn 做最多 grpcClientReadyCheckTimeout
// 的连接探测。conn 进入 Ready 视为成功,否则记录 fail。失败不阻断 fx 启动 —
// 仅用作可观测,生产环境某个下游短暂不可达不应让本服务起不来。
func probeClientTargetReady(conn *grpc.ClientConn, target string) {
	ctx, cancel := context.WithTimeout(context.Background(), grpcClientReadyCheckTimeout)
	defer cancel()
	for {
		state := conn.GetState()
		if state.String() == "READY" {
			clientTargetReady.WithLabelValues(target, "success").Inc()
			return
		}
		if !conn.WaitForStateChange(ctx, state) {
			clientTargetReady.WithLabelValues(target, "fail").Inc()
			logs.Warn("grpc client target 启动期未就绪,继续启动",
				zap.String("target", target), zap.String("last_state", state.String()))
			return
		}
	}
}

func registerCollector(collector prometheus_client.Collector) {
	if err := prometheus_client.Register(collector); err != nil {
		if _, ok := err.(prometheus_client.AlreadyRegisteredError); !ok {
			logs.Warn("注册grpc prometheus指标出错", zap.Error(err))
		}
	}
}

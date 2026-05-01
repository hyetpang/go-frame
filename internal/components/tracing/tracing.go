// Package tracing 提供 OpenTelemetry 分布式追踪的 fx 组件。
//
// 当配置 enable=false 时返回 noop TracerProvider，保持向后兼容；
// enable=true 时根据 protocol 选择 OTLP HTTP 或 gRPC exporter，
// 并将 TracerProvider/Propagator 注册为 OTel 全局对象，方便各
// instrumentation（otelgrpc/otelgin/otelgorm/redisotel/otelsarama）
// 通过 otel.GetTracerProvider() 获取。
package tracing

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
)

// errInsecureTLSConflict 表示同时开启了 Insecure 与 TLS,这是显然冲突的配置,
// 必须由用户在 toml 中二选一(本地开发用 insecure=true、生产用 [tracing.tls] enable=true)。
var errInsecureTLSConflict = errors.New("tracing 配置冲突: insecure=true 与 tls.enable=true 不能同时启用,请二选一")

const (
	protocolHTTP        = "http"
	protocolGRPC        = "grpc"
	exporterInitTimeout = 10 * time.Second
	shutdownTimeout     = 5 * time.Second
)

// New 根据配置返回 OpenTelemetry TracerProvider。
//
// 关闭追踪（Enable=false）时直接返回 noop.NewTracerProvider()，
// 不会进行任何网络连接，保持零成本回退；启用时会创建 OTLP exporter
// 并通过 fx StopHook 在应用退出时优雅关闭。
func New(zapLog *zap.Logger, lc fx.Lifecycle, conf *config) (trace.TracerProvider, error) {
	if conf == nil || !conf.Enable {
		tp := noop.NewTracerProvider()
		otel.SetTracerProvider(tp)
		zapLog.Info("OpenTelemetry tracing 未启用，使用 noop TracerProvider")
		return tp, nil
	}

	// 配置冲突的 fail-fast 提前到 buildResource 之前,避免被无关错误(如 schema URL)掩盖
	if conf.Insecure && conf.TLS.IsEnabled() {
		return nil, errInsecureTLSConflict
	}

	res, err := buildResource(conf)
	if err != nil {
		return nil, fmt.Errorf("构建追踪 resource 出错: %w", err)
	}

	exporter, err := newExporter(conf)
	if err != nil {
		return nil, fmt.Errorf("创建 OTLP trace exporter 出错: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(conf.SampleRatio))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	lc.Append(fx.StopHook(func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			zapLog.Error("关闭 TracerProvider 出错", zap.Error(err))
			return err
		}
		return nil
	}))

	zapLog.Info("OpenTelemetry tracing 已启用",
		zap.String("service_name", conf.ServiceName),
		zap.String("endpoint", conf.Endpoint),
		zap.String("protocol", conf.Protocol),
		zap.Float64("sample_ratio", conf.SampleRatio),
		zap.Bool("insecure", conf.Insecure),
		zap.Bool("tls_enabled", conf.TLS.IsEnabled()),
	)
	if conf.Insecure && !isLoopbackOrInternalEndpoint(conf.Endpoint) {
		zapLog.Warn("OTel exporter 配置 insecure=true 但 endpoint 看起来是公网地址,trace 将明文外发",
			zap.String("endpoint", conf.Endpoint))
	}
	return tp, nil
}

// isLoopbackOrInternalEndpoint 判断 endpoint 是否指向本机 / k8s 内网 / .local 域,
// 用于在 insecure=true 时只对疑似公网地址打 WARN,避免本地开发噪音。
func isLoopbackOrInternalEndpoint(endpoint string) bool {
	if endpoint == "" {
		return true
	}
	// otlptrace endpoint 既可能带 scheme 也可能裸 host:port,这里统一剥离再用 SplitHostPort
	host := endpoint
	if idx := strings.Index(host, "://"); idx >= 0 {
		host = host[idx+3:]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified()
	}
	// 不带 dot 的纯 hostname 视为内网(docker-compose 服务名 / k8s 同 namespace 短名)
	if !strings.Contains(host, ".") {
		return true
	}
	// k8s service / local DNS 后缀视为内网
	if strings.HasSuffix(host, ".svc") || strings.HasSuffix(host, ".local") ||
		strings.Contains(host, ".svc.") || strings.HasSuffix(host, ".internal") {
		return true
	}
	return false
}

func buildResource(conf *config) (*sdkresource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(conf.ServiceName),
	}
	return sdkresource.Merge(
		sdkresource.Default(),
		sdkresource.NewWithAttributes(semconv.SchemaURL, attrs...),
	)
}

func newExporter(conf *config) (*otlptrace.Exporter, error) {
	// Insecure 与 TLS 同时开启属于明显冲突,提前 fail-fast 避免线上跑起来才发现配置矛盾
	if conf.Insecure && conf.TLS.IsEnabled() {
		return nil, errInsecureTLSConflict
	}
	tlsCfg, err := conf.TLS.BuildClientTLS()
	if err != nil {
		return nil, fmt.Errorf("构建 OTLP exporter TLS 配置出错: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), exporterInitTimeout)
	defer cancel()
	switch strings.ToLower(conf.Protocol) {
	case "", protocolHTTP:
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(conf.Endpoint)}
		if len(conf.Headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(conf.Headers))
		}
		if conf.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		} else if tlsCfg != nil {
			opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsCfg))
		}
		return otlptracehttp.New(ctx, opts...)
	case protocolGRPC:
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(conf.Endpoint)}
		if len(conf.Headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(conf.Headers))
		}
		if conf.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		} else if tlsCfg != nil {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
		}
		return otlptracegrpc.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("不支持的 OTLP 协议: %s", conf.Protocol)
	}
}

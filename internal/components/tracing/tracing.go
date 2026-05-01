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
	"fmt"
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
)

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
	)
	return tp, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), exporterInitTimeout)
	defer cancel()
	switch strings.ToLower(conf.Protocol) {
	case "", protocolHTTP:
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(conf.Endpoint)}
		if conf.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, opts...)
	case protocolGRPC:
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(conf.Endpoint)}
		if conf.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		return otlptracegrpc.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("不支持的 OTLP 协议: %s", conf.Protocol)
	}
}

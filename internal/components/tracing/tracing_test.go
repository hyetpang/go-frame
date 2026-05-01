package tracing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	frameconfig "github.com/hyetpang/go-frame/internal/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNewReturnsNoopTracerWhenDisabled(t *testing.T) {
	conf := &config{Enable: false}
	lc := fxtest.NewLifecycle(t)

	tp, err := New(zap.NewNop(), lc, conf)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil noop TracerProvider")
	}
	// noop TracerProvider 不应是 sdktrace.TracerProvider 类型
	if _, ok := tp.(*sdktrace.TracerProvider); ok {
		t.Fatal("expected noop tracer provider when disabled")
	}
	lc.RequireStart().RequireStop()
}

func TestNewReturnsNoopWhenConfigNil(t *testing.T) {
	lc := fxtest.NewLifecycle(t)

	tp, err := New(zap.NewNop(), lc, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil tracer provider")
	}
	lc.RequireStart().RequireStop()
}

func TestNewReturnsErrorForUnsupportedProtocol(t *testing.T) {
	conf := &config{
		Enable:      true,
		Protocol:    "unsupported",
		Endpoint:    "localhost:4318",
		SampleRatio: 1.0,
		ServiceName: "test",
	}
	lc := fxtest.NewLifecycle(t)

	if _, err := New(zap.NewNop(), lc, conf); err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
}

// TestNewExporterWithHeaders 验证 Headers 配置会真正注入到 OTLP HTTP 请求里。
// 通过 httptest.Server 截获 exporter 实际发出的 trace 上报请求,断言 Authorization 头存在。
func TestNewExporterWithHeaders(t *testing.T) {
	var (
		mu          sync.Mutex
		gotAuth     string
		gotAPIKey   string
		gotPath     string
		requestSeen = make(chan struct{}, 1)
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("X-Api-Key")
		gotPath = r.URL.Path
		mu.Unlock()
		select {
		case requestSeen <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	parsed, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("解析 httptest URL 出错: %v", err)
	}

	conf := &config{
		Enable:      true,
		Protocol:    "http",
		Endpoint:    parsed.Host, // OTLP HTTP exporter 期望 host:port,scheme 由 Insecure 决定
		Insecure:    true,        // httptest.Server 默认明文,这里走 insecure 不冲突
		SampleRatio: 1.0,
		ServiceName: "test",
		Headers: map[string]string{
			"Authorization": "Bearer secret-token",
			"X-Api-Key":     "abc123",
		},
	}

	exporter, err := newExporter(conf)
	if err != nil {
		t.Fatalf("newExporter 返回错误: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = exporter.Shutdown(ctx)
	})

	// 触发一次实际上报,exporter 才会真正写出 HTTP 请求
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
	})
	_, span := tp.Tracer("test").Start(context.Background(), "header-injection")
	span.End()
	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("ForceFlush 失败: %v", err)
	}

	select {
	case <-requestSeen:
	case <-time.After(3 * time.Second):
		t.Fatal("OTLP collector mock 未收到请求,headers 注入路径未被触达")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("Authorization header = %q, want %q", gotAuth, "Bearer secret-token")
	}
	if gotAPIKey != "abc123" {
		t.Fatalf("X-Api-Key header = %q, want %q", gotAPIKey, "abc123")
	}
	if gotPath == "" {
		t.Fatal("expected exporter to hit collector with non-empty path")
	}
}

// TestNewExporterRejectsInsecureAndTLSBoth 验证 insecure=true 与 tls.enable=true 互斥。
// 这种组合属于明显误配,必须在 newExporter 阶段直接返回错误,避免线上跑起来才暴露。
func TestNewExporterRejectsInsecureAndTLSBoth(t *testing.T) {
	conf := &config{
		Enable:      true,
		Protocol:    "grpc",
		Endpoint:    "127.0.0.1:4317",
		Insecure:    true,
		SampleRatio: 1.0,
		ServiceName: "test",
		TLS:         frameconfig.TLSConfig{Enable: true},
	}
	if _, err := newExporter(conf); err == nil {
		t.Fatal("expected newExporter to reject insecure=true & tls.enable=true combination")
	} else if !errors.Is(err, errInsecureTLSConflict) {
		t.Fatalf("expected errInsecureTLSConflict, got: %v", err)
	}

	// 通过 New 入口也应一并报错(被外层 fmt.Errorf 包裹,但 errors.Is 仍能识别)
	confHTTP := *conf
	confHTTP.Protocol = "http"
	lc := fxtest.NewLifecycle(t)
	if _, err := New(zap.NewNop(), lc, &confHTTP); err == nil {
		t.Fatal("expected New to surface insecure/tls conflict")
	} else if !errors.Is(err, errInsecureTLSConflict) {
		t.Fatalf("expected wrapped errInsecureTLSConflict, got: %v", err)
	}
}

// TestIsLoopbackOrInternalEndpoint 验证 endpoint 内网/公网判别覆盖典型场景,
// 直接关系到 insecure=true 时是否打 WARN 日志,逻辑分支较多需要单独覆盖。
func TestIsLoopbackOrInternalEndpoint(t *testing.T) {
	internal := []string{
		"",
		"localhost:4318",
		"127.0.0.1:4317",
		"10.0.0.5:4317",
		"192.168.1.10:4318",
		"http://otel-collector:4318",
		"otel.svc.cluster.local:4318",
		"otel-collector.observability.svc:4318",
		"my.internal:4318",
	}
	external := []string{
		"api.honeycomb.io:443",
		"otlp.eu.example.com:4318",
		"8.8.8.8:4318",
	}
	for _, ep := range internal {
		if !isLoopbackOrInternalEndpoint(ep) {
			t.Errorf("endpoint %q 应被识别为内网/本地,实际被识别为公网", ep)
		}
	}
	for _, ep := range external {
		if isLoopbackOrInternalEndpoint(ep) {
			t.Errorf("endpoint %q 应被识别为公网,实际被识别为内网/本地", ep)
		}
	}
}

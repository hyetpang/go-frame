package tracing

import (
	"testing"

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

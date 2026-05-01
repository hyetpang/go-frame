package logs

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRegisterNoticeHookForwardsCallerInfo(t *testing.T) {
	t.Cleanup(unregisterNoticeHook)

	var (
		mu       sync.Mutex
		gotMsg   string
		gotFile  string
		gotLine  int
		gotField string
		done     = make(chan struct{})
	)
	RegisterNoticeHook(func(msg string, filename string, line int, fields ...zap.Field) {
		mu.Lock()
		defer mu.Unlock()
		gotMsg = msg
		gotFile = filename
		gotLine = line
		if len(fields) > 0 {
			gotField = fields[0].Key
		}
		close(done)
	})

	Error("test message", zap.String("k", "v"))

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for hook")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotMsg != "test message" {
		t.Fatalf("msg = %q, want %q", gotMsg, "test message")
	}
	if !strings.HasSuffix(gotFile, "logs_test.go") {
		t.Fatalf("filename = %q, expected to end with logs_test.go", gotFile)
	}
	if gotLine == 0 {
		t.Fatal("line should not be zero")
	}
	if gotField != "k" {
		t.Fatalf("field key = %q, want k", gotField)
	}
}

func TestUnregisterNoticeHook(t *testing.T) {
	called := make(chan struct{}, 1)
	RegisterNoticeHook(func(msg string, filename string, line int, fields ...zap.Field) {
		called <- struct{}{}
	})
	unregisterNoticeHook()

	Error("after unregister")

	select {
	case <-called:
		t.Fatal("hook should not be called after unregister")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestCtxReturnsGlobalLoggerWhenNoSpan(t *testing.T) {
	logger := Ctx(context.Background())
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if logger != zap.L() {
		t.Fatal("expected global logger when ctx has no span")
	}
}

func TestCtxReturnsGlobalLoggerForNilContext(t *testing.T) {
	// nolint:staticcheck // 故意传 nil 验证容错
	logger := Ctx(nil)
	if logger != zap.L() {
		t.Fatal("expected global logger when ctx is nil")
	}
}

func TestCtxAppendsTraceIDAndSpanIDFields(t *testing.T) {
	core, recorded := observer.New(zap.DebugLevel)
	original := zap.L()
	t.Cleanup(func() { zap.ReplaceGlobals(original) })
	zap.ReplaceGlobals(zap.New(core))

	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	Ctx(ctx).Info("test")

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	if got := fields["trace_id"]; got != traceID.String() {
		t.Fatalf("trace_id = %v, want %s", got, traceID.String())
	}
	if got := fields["span_id"]; got != spanID.String() {
		t.Fatalf("span_id = %v, want %s", got, spanID.String())
	}
}

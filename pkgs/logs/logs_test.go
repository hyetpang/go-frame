package logs

import (
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
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

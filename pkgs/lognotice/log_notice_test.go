package lognotice

import (
	"sync"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
)

type recordingNotifier struct {
	count atomic.Int64
	last  atomic.Value // string
}

func (r *recordingNotifier) Notice(msg string, _ string, _ int, _ ...zap.Field) {
	r.count.Add(1)
	r.last.Store(msg)
}

// TestNoticeIsSafeByDefault 默认实现是 noop,直接调 Notice 不应 panic。
func TestNoticeIsSafeByDefault(t *testing.T) {
	defer resetLogNotice(t)
	Notice("default", "f.go", 1)
}

// TestInjectAndNoticeForwards 注入后 Notice 应转发到注入的实现。
func TestInjectAndNoticeForwards(t *testing.T) {
	defer resetLogNotice(t)

	r := &recordingNotifier{}
	Inject(r)
	Notice("hello", "f.go", 42)

	if got := r.count.Load(); got != 1 {
		t.Fatalf("Inject 后 Notice 调用次数 = %d, want 1", got)
	}
	if got := r.last.Load().(string); got != "hello" {
		t.Fatalf("Notice 转发的 msg = %q, want %q", got, "hello")
	}
}

// TestInjectNilFallsBackToNoop Inject(nil) 不应让请求路径 panic,
// 而是回退到默认 noop 实现(防止误注入 nil)。
func TestInjectNilFallsBackToNoop(t *testing.T) {
	defer resetLogNotice(t)

	r := &recordingNotifier{}
	Inject(r)
	Inject(nil) // 应被视作重置

	Notice("after-nil-inject", "f.go", 1)
	if got := r.count.Load(); got != 0 {
		t.Fatalf("Inject(nil) 后不应仍转发到旧 notifier,但被调用了 %d 次", got)
	}
}

// TestConcurrentInjectAndNotice race detector 守护:
// 并发 Inject + Notice 不应触发 race(atomic.Pointer 保证)。
func TestConcurrentInjectAndNotice(t *testing.T) {
	defer resetLogNotice(t)

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			Inject(&recordingNotifier{})
		}()
		go func() {
			defer wg.Done()
			Notice("concurrent", "f.go", 1)
		}()
	}
	wg.Wait()
}

// resetLogNotice 把全局 notifier 复位到默认 noop,避免测试串扰。
func resetLogNotice(_ *testing.T) {
	var n Notifier = defaultLogNotice{}
	logNotice.Store(&n)
}

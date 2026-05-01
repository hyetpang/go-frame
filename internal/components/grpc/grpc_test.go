package grpc

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestGracefulStopWithTimeoutFallsBackToStop(t *testing.T) {
	stopped := make(chan struct{})
	forced := make(chan struct{}, 1)

	// forceStop 模拟 grpc.Server.Stop 的真实语义:
	// 强制关闭后会唤醒被 GracefulStop 阻塞的 goroutine。
	gracefulStopWithTimeout(func() {
		<-stopped
	}, func() {
		forced <- struct{}{}
		close(stopped)
	}, time.Millisecond)

	select {
	case <-forced:
	case <-time.After(time.Second):
		t.Fatal("expected force stop after timeout")
	}
}

// TestGracefulStopWithTimeoutWaitsForGoroutineExit 验证超时分支会等待 graceful goroutine
// 真正退出后再返回,避免悬挂 goroutine。
func TestGracefulStopWithTimeoutWaitsForGoroutineExit(t *testing.T) {
	stopped := make(chan struct{})
	gracefulReturned := make(chan struct{})

	go func() {
		gracefulStopWithTimeout(func() {
			<-stopped
			close(gracefulReturned)
		}, func() {
			close(stopped)
		}, time.Millisecond)
	}()

	select {
	case <-gracefulReturned:
	case <-time.After(time.Second):
		t.Fatal("graceful goroutine 未在合理时间内退出")
	}
}

// TestNewServerStartsWithoutFixedOneSecondDelay 验证 grpc server 启动不再有固定 1s 等待
func TestNewServerStartsWithoutFixedOneSecondDelay(t *testing.T) {
	conf := &config{
		Address: "127.0.0.1:0",
	}
	lc := fxtest.NewLifecycle(t)

	if _, err := NewServer(lc, zap.NewNop(), conf); err != nil {
		t.Fatalf("NewServer 出错: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	if err := lc.Start(ctx); err != nil {
		t.Fatalf("lifecycle 启动失败: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer stopCancel()
		_ = lc.Stop(stopCtx)
	})

	if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
		t.Fatalf("lifecycle 启动耗时 %s,不应再有固定 1s 等待", elapsed)
	}
}

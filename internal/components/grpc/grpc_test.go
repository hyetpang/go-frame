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

	gracefulStopWithTimeout(func() {
		<-stopped
	}, func() {
		forced <- struct{}{}
	}, time.Millisecond)

	select {
	case <-forced:
	case <-time.After(time.Second):
		t.Fatal("expected force stop after timeout")
	}
	close(stopped)
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

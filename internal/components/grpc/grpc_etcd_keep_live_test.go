package grpc

import (
	"context"
	"testing"
	"time"
)

func TestSleepWithCtxReturnsTrueOnTimer(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	if !sleepWithCtx(ctx, 10*time.Millisecond) {
		t.Fatal("sleepWithCtx 在正常超时时应返回 true")
	}
	if elapsed := time.Since(start); elapsed < 10*time.Millisecond {
		t.Fatalf("sleepWithCtx 实际耗时 %s,小于配置时长", elapsed)
	}
}

func TestSleepWithCtxReturnsFalseOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if sleepWithCtx(ctx, time.Hour) {
		t.Fatal("sleepWithCtx 在 ctx 取消后应返回 false")
	}
}

func TestSleepWithCtxCancelMidway(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	if sleepWithCtx(ctx, time.Hour) {
		t.Fatal("sleepWithCtx 在等待途中被取消时应返回 false")
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("sleepWithCtx 取消后未及时返回,耗时 %s", elapsed)
	}
}

func TestSleepWithCtxZeroDuration(t *testing.T) {
	ctx := context.Background()
	if !sleepWithCtx(ctx, 0) {
		t.Fatal("sleepWithCtx(0) 在 ctx 未取消时应返回 true")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if sleepWithCtx(cancelled, 0) {
		t.Fatal("sleepWithCtx(0) 在 ctx 已取消时应返回 false")
	}
}

func TestJitterZero(t *testing.T) {
	if got := jitter(0); got != 0 {
		t.Fatalf("jitter(0) = %s, want 0", got)
	}
	if got := jitter(-time.Second); got != 0 {
		t.Fatalf("jitter(负数) = %s, want 0", got)
	}
}

func TestJitterRange(t *testing.T) {
	const base = 200 * time.Millisecond
	for range 64 {
		got := jitter(base)
		if got < base {
			t.Fatalf("jitter(%s) = %s, 小于下界 %s", base, got, base)
		}
		// 实现是 d + rand[0, d/2+1],上界 = d + d/2 = 1.5*d (向下取整后 +1ns 容忍)
		if upper := base + base/2 + time.Nanosecond; got > upper {
			t.Fatalf("jitter(%s) = %s, 超出上界 %s", base, got, upper)
		}
	}
}

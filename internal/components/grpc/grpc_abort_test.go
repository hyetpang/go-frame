package grpc

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
)

// TestAbortServerStartupReleasesPort 验证启动失败回收逻辑会:
//  1. 关闭 listener 释放端口(可被重新监听);
//  2. 停止 server 使后台 Serve goroutine 退出;
//  3. closeOnce 去重,后续 OnStop 再次关闭不会 double-close。
func TestAbortServerStartupReleasesPort(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	addr := lis.Addr().String()

	s := grpc.NewServer()
	// 模拟后台 Serve goroutine
	serveDone := make(chan struct{})
	go func() {
		defer close(serveDone)
		_ = s.Serve(lis)
	}()
	// 给 Serve 一点时间接管 listener
	time.Sleep(20 * time.Millisecond)

	var once sync.Once
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	abortServerStartup(ctx, s, lis, &once)

	// 后台 Serve goroutine 应已退出
	select {
	case <-serveDone:
	case <-time.After(time.Second):
		t.Fatal("Serve goroutine 未退出,存在泄漏")
	}

	// 端口应已释放,可被重新监听
	relis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("端口未释放,无法重新监听 %s: %v", addr, err)
	}
	_ = relis.Close()

	// 再次走 OnStop 风格的 Close 不应 panic / double-close 失败
	once.Do(func() { _ = lis.Close() })
}

// TestAbortServerStartupClosesListenerBeforeServe 验证在 Serve 尚未接管 listener 时
// 调用回收逻辑,listener 仍会被兜底关闭。
func TestAbortServerStartupClosesListenerBeforeServe(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	addr := lis.Addr().String()
	s := grpc.NewServer()

	var once sync.Once
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	abortServerStartup(ctx, s, lis, &once)

	relis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("端口未释放,无法重新监听 %s: %v", addr, err)
	}
	_ = relis.Close()
}

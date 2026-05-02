// Package lifecycle 提供 fx Lifecycle 关闭场景的通用 helper。
//
// 设计意图:
//   - 第三方库(go-redis/etcd/sarama/sql 等)的 Close 方法多数是同步阻塞的,
//     不接受 context,网络抖动时可能卡数秒。
//   - fx 的 Stop 流程串行执行 hook,前面卡住会让后面来不及关。
//   - 旧版组件用 fx.StopHook(func(){...}) 没有 ctx 入口,无法感知 fx Stop timeout。
//
// CloseWithContext 把 close 异步化并与 ctx 竞速,超时返回 ctx.Err 让 fx 继续推进,
// 避免单个慢 Close 拖死整个 Stop 流程。被搁置的 close goroutine 由进程退出回收。
package lifecycle

import (
	"context"
	"fmt"
)

// CloseWithContext 在 ctx deadline 内等待 close 完成。超时返回 ctx.Err 包装,
// 调用方可在外层做日志/指标。name 仅用于错误信息便于排查。
func CloseWithContext(ctx context.Context, name string, close func() error) error {
	done := make(chan error, 1)
	go func() {
		done <- close()
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("关闭 %s 超时: %w", name, ctx.Err())
	}
}

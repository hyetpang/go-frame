package common

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

const (
	redisLockTimeout = time.Second * 5
)

// redisClientPtr/redisSyncPtr 通过 atomic.Pointer 保护:
// InjectRedis 在启动期由 fx 写入,Lock 在请求路径并发读,
// 用 atomic 替代裸全局变量避免 race detector 报警与可见性问题。
var (
	redisClientPtr atomic.Pointer[redis.UniversalClient]
	redisSyncPtr   atomic.Pointer[redsync.Redsync]
)

func InjectRedis(redisC redis.UniversalClient) {
	redisClientPtr.Store(&redisC)
	pool := goredis.NewPool(redisC)
	rs := redsync.New(pool)
	redisSyncPtr.Store(rs)
}

// LockContext 通过 ctx 控制加锁等待:Redis 抖动时调用方的 ctx deadline/cancel 立即生效,
// 不会被 redsync 内置的 tries+delay 退避吞没。请求路径建议优先使用本函数。
// 返回的释放闭包也接收 ctx,保证 Unlock 的 Redis 命令受 ctx 约束。
func LockContext(ctx context.Context, key string, options ...redsync.Option) (func(context.Context) error, error) {
	rs := redisSyncPtr.Load()
	if rs == nil {
		return nil, errors.New("redis 未初始化,请确认已注册 WithRedis()")
	}
	merged := make([]redsync.Option, 0, len(options)+2)
	merged = append(merged,
		redsync.WithGenValueFunc(GenNanoID),
		redsync.WithExpiry(redisLockTimeout),
	)
	merged = append(merged, options...)
	lock := rs.NewMutex(key, merged...)
	if err := lock.LockContext(ctx); err != nil {
		return nil, err
	}
	return func(releaseCtx context.Context) error {
		ok, err := lock.UnlockContext(releaseCtx)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("redsync解锁失败")
		}
		return nil
	}, nil
}

// Deprecated: 请使用 LockContext。Lock 在请求路径会吞没调用方 ctx,
// Redis 抖动时按 redsync 默认 tries+delay 退避,可阻塞十几秒。
func Lock(key string, options ...redsync.Option) (func() error, error) {
	release, err := LockContext(context.Background(), key, options...)
	if err != nil {
		return nil, err
	}
	return func() error { return release(context.Background()) }, nil
}

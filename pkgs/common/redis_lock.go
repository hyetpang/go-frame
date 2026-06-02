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
//
// 警告:锁默认过期时间为 redisLockTimeout(5s)且无看门狗续约。
// 临界区耗时必须远小于 5s,否则锁会自动过期,导致其他协程/进程并发进入临界区,失去互斥保证!
// 若临界区可能较长,请通过 options 传入 redsync.WithExpiry(...) 自定义更大的过期时间。
func LockContext(ctx context.Context, key string, options ...redsync.Option) (func(context.Context) error, error) {
	rs := redisSyncPtr.Load()
	if rs == nil {
		return nil, errors.New("redis 未初始化,请确认已注册 WithRedis()")
	}
	merged := make([]redsync.Option, 0, len(options)+2)
	merged = append(merged,
		// 锁 value 只需唯一且必须有值。GenNanoID 在极端情况下会返回 error 导致加锁直接失败,
		// 这里改用带回退、永不失败的 GenID(),保证总能拿到一个唯一 value。
		redsync.WithGenValueFunc(func() (string, error) { return GenID(), nil }),
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
//
// 警告:锁默认过期时间为 redisLockTimeout(5s)且无看门狗续约。
// 临界区耗时必须远小于 5s,否则锁会自动过期,导致其他协程/进程并发进入临界区,失去互斥保证!
// 若临界区可能较长,请通过 options 传入 redsync.WithExpiry(...) 自定义更大的过期时间。
func Lock(key string, options ...redsync.Option) (func() error, error) {
	release, err := LockContext(context.Background(), key, options...)
	if err != nil {
		return nil, err
	}
	return func() error { return release(context.Background()) }, nil
}

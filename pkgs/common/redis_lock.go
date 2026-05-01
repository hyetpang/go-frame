package common

import (
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

func Lock(key string, options ...redsync.Option) (func() error, error) {
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
	err := lock.Lock()
	if err != nil {
		return nil, err
	}
	return func() error {
		ok, err := lock.Unlock()
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("redsync解锁失败")
		}
		return nil
	}, nil
}

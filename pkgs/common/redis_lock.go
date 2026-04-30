package common

import (
	"errors"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

const (
	redisLockTimeout = time.Second * 5
)

var redisClient redis.UniversalClient
var redisSync *redsync.Redsync

func InjectRedis(redisC redis.UniversalClient) {
	redisClient = redisC
	pool := goredis.NewPool(redisClient)
	redisSync = redsync.New(pool)
}

func Lock(key string, options ...redsync.Option) (func() error, error) {
	if redisSync == nil {
		return nil, errors.New("redis lock 未初始化，请先调用 WithRedis()")
	}
	merged := make([]redsync.Option, 0, len(options)+2)
	merged = append(merged,
		redsync.WithGenValueFunc(GenNanoID),
		redsync.WithExpiry(redisLockTimeout),
	)
	merged = append(merged, options...)
	lock := redisSync.NewMutex(key, merged...)
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

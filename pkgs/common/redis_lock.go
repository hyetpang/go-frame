package common

import (
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
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
	if len(options) < 1 {
		options = []redsync.Option{
			redsync.WithGenValueFunc(GenNanoID),
			redsync.WithExpiry(redisLockTimeout),
		}
	}
	lock := redisSync.NewMutex(key, options...)
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

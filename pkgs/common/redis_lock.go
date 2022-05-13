package common

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	redisLockTimeout = time.Second * 5
)

// 尝试获取分布式锁，获取不到会一直等待直到超时
func TryRedisLock(redisClient redis.UniversalClient, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), RedisTimeout)
	defer cancel()
	nanoID, err := GenNanoID()
	if err != nil {
		return "", err
	}
	for {
		select {
		case <-time.After(redisLockTimeout):
			return "", errors.New("获取锁超时")
		default:
			result := redisClient.SetNX(ctx, key, nanoID, redisLockTimeout)
			if err := result.Err(); err != nil {
				return "", err
			}
			if result.Val() {
				return nanoID, nil
			}
		}
	}
}

// 获取分布式锁，没获取到直接返回不等待
func MustRedisLock(redisClient redis.UniversalClient, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), RedisTimeout)
	defer cancel()
	nanoID, err := GenNanoID()
	if err != nil {
		return "", err
	}
	if redisClient.SetNX(ctx, key, nanoID, redisLockTimeout).Val() {
		return nanoID, nil
	}
	return "", errors.New("不能获取锁:" + key)
}

// 解锁
func RedisUnlock(redisClient redis.UniversalClient, key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), RedisTimeout)
	defer cancel()
	result := redisClient.Get(ctx, key)
	if err := result.Err(); err != nil {
		return err
	}
	if result.Val() != value {
		return errors.New("不是解的自己的锁")
	}
	if err := redisClient.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}

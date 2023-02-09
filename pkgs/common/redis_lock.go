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

var redisClient redis.UniversalClient

func InjectRedis(redisC redis.UniversalClient) {
	redisClient = redisC
}

// 尝试获取分布式锁，获取不到会一直等待直到超时
func TryRedisLock(redisClient redis.UniversalClient, key string, timeouts ...time.Duration) (string, error) {
	timeout := redisLockTimeout
	if len(timeouts) < 1 {
		timeout = timeouts[0]
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

func TryGetRedisLock(key string, timeouts ...time.Duration) (string, error) {
	return TryRedisLock(redisClient, key, timeouts...)
}

func MustGetRedisLock(key string, timeouts ...time.Duration) (string, error) {
	if len(timeouts) < 1 {
		return MustRedisLockWithTimeout(redisClient, key, timeouts[0])
	}
	return MustRedisLock(redisClient, key)
}

// 获取分布式锁，没获取到直接返回不等待
func MustRedisLock(redisClient redis.UniversalClient, key string) (string, error) {
	return MustRedisLockWithTimeout(redisClient, key, redisLockTimeout)
}

// 获取分布式锁，没获取到直接返回不等待
func MustRedisLockWithTimeout(redisClient redis.UniversalClient, key string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), RedisTimeout)
	defer cancel()
	nanoID, err := GenNanoID()
	if err != nil {
		return "", err
	}
	result := redisClient.SetNX(ctx, key, nanoID, timeout)
	if err := result.Err(); err != nil {
		return "", err
	}
	if result.Val() {
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

// 解锁
func RedisUnlockWithoutClient(key, value string) error {
	return RedisUnlock(redisClient, key, value)
}

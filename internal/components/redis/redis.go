package redis

import (
	"context"

	"github.com/HyetPang/go-frame/internal/constants"
	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func New() redis.UniversalClient {
	redisOptions := &redis.Options{
		Addr:     viper.GetString("redis.addr"),
		Password: viper.GetString("redis.pwd"),
		DB:       viper.GetInt("redis.db"),
	}
	redisClient := redis.NewClient(redisOptions)
	ctx, cancel := context.WithTimeout(context.Background(), constants.CtxTimeOut)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logs.Fatal("连接redis出错", zap.Error(err), zap.Any("redisOptions", redisOptions))
	}
	return redisClient
}

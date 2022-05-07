/*
 * @Date: 2022-04-30 10:35:09
 * @LastEditTime: 2022-05-07 17:20:13
 * @FilePath: /go-frame/internal/components/redis/redis.go
 */
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
	conf := new(config)
	err := viper.UnmarshalKey("redis", conf)
	if err != nil {
		logs.Fatal("mysql配置Unmarshal到对象出错", zap.Error(err))
	}
	return newRedis(conf)
}

func newRedis(conf *config) redis.UniversalClient {
	redisOptions := &redis.Options{
		Addr:     conf.Addr,
		Password: conf.Pwd,
		DB:       conf.DB,
	}
	redisClient := redis.NewClient(redisOptions)
	ctx, cancel := context.WithTimeout(context.Background(), constants.CtxTimeOut)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logs.Fatal("连接redis出错", zap.Error(err), zap.Any("conf", conf))
	}
	return redisClient
}

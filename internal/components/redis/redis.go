/*
 * @Date: 2022-04-30 10:35:09
 * @LastEditTime: 2022-04-30 16:56:25
 * @FilePath: \ultrasdk.hub.gof:\projects\ultrasdk.hub\go-frame\internal\components\redis\redis.go
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
	addr := viper.GetString("redis.addr")
	pwd := viper.GetString("redis.pwd")
	db := viper.GetInt("redis.db")
	redisOptions := &redis.Options{
		Addr:     addr,
		Password: pwd,
		DB:       db,
	}
	redisClient := redis.NewClient(redisOptions)
	ctx, cancel := context.WithTimeout(context.Background(), constants.CtxTimeOut)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logs.Fatal("连接redis出错", zap.Error(err), zap.String("addr", addr), zap.String("pwd", pwd), zap.Int("db", db))
	}
	return redisClient
}

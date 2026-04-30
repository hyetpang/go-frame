package redis

import (
	"context"

	"github.com/hyetpang/go-frame/internal/constants"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(lc fx.Lifecycle) redis.UniversalClient {
	conf := new(config)
	err := viper.UnmarshalKey("redis", conf)
	if err != nil {
		logs.Fatal("redis配置Unmarshal到对象出错", zap.Error(err))
	}
	common.MustValidate(conf)
	redisClient := newRedis(conf)
	lc.Append(fx.StopHook(func() {
		if e := redisClient.Close(); e != nil {
			logs.Error("关闭redis连接出错", zap.Error(e))
		}
	}))
	return redisClient
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
		logs.Fatal("连接redis出错", zap.Error(err), zap.String("addr", conf.Addr))
	}
	return redisClient
}

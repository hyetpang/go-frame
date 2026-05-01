package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/hyetpang/go-frame/internal/constants"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(lc fx.Lifecycle, conf *config) (redis.UniversalClient, error) {
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("redis配置验证不通过: %w", err)
	}
	redisClient, err := newRedis(conf)
	if err != nil {
		return nil, err
	}
	if err := redisotel.InstrumentTracing(redisClient); err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("redis 注入 OpenTelemetry tracing 出错: %w", err)
	}
	lc.Append(fx.StopHook(func() {
		if e := redisClient.Close(); e != nil {
			logs.Error("关闭redis连接出错", zap.Error(e))
		}
	}))
	return redisClient, nil
}

func newRedis(conf *config) (redis.UniversalClient, error) {
	tlsCfg, err := conf.TLS.BuildClientTLS()
	if err != nil {
		return nil, fmt.Errorf("构建 redis TLS 配置出错: %w", err)
	}
	redisOptions := &redis.Options{
		Addr:         conf.Addr,
		Username:     conf.Username,
		Password:     conf.Pwd,
		DB:           conf.DB,
		PoolSize:     conf.PoolSize,
		MinIdleConns: conf.MinIdleConns,
		DialTimeout:  time.Duration(conf.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(conf.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(conf.WriteTimeout) * time.Second,
		TLSConfig:    tlsCfg,
	}
	redisClient := redis.NewClient(redisOptions)
	ctx, cancel := context.WithTimeout(context.Background(), constants.CtxTimeOut)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("连接redis出错 addr=%s: %w", conf.Addr, err)
	}
	return redisClient, nil
}

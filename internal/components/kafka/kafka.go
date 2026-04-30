package kafka

import (
	"context"
	"fmt"
	"strings"

	"github.com/IBM/sarama"
	"github.com/hyetpang/go-frame/internal/adapter/log"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(lc fx.Lifecycle, zapLog *zap.Logger, conf *config) (sarama.Client, sarama.AsyncProducer, sarama.SyncProducer, sarama.Consumer, error) {
	if err := common.Validate(conf); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("kafka配置验证不通过: %w", err)
	}
	sarama.Logger = log.NewKafkaLog(zapLog)
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll          // 发送完数据需要leader和follow都确认
	config.Producer.Partitioner = sarama.NewRandomPartitioner // 新选出一个partition
	config.Producer.Return.Successes = true
	if len(conf.ClientID) > 0 {
		config.ClientID = conf.ClientID
	}
	// 连接kafka
	client, err := sarama.NewClient(strings.Split(conf.Addr, ","), config)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("连接kafka出错 addr=%s: %w", conf.Addr, err)
	}
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		_ = client.Close()
		return nil, nil, nil, nil, fmt.Errorf("创建kafka consumer出错 addr=%s: %w", conf.Addr, err)
	}
	asyncProducer, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		_ = consumer.Close()
		_ = client.Close()
		return nil, nil, nil, nil, fmt.Errorf("创建kafka async producer出错 addr=%s: %w", conf.Addr, err)
	}
	syncProducer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		_ = asyncProducer.Close()
		_ = consumer.Close()
		_ = client.Close()
		return nil, nil, nil, nil, fmt.Errorf("创建kafka sync producer出错 addr=%s: %w", conf.Addr, err)
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			if e := asyncProducer.Close(); e != nil {
				logs.Error("关闭kafka异步producer出错", zap.Error(e))
			}
			if e := syncProducer.Close(); e != nil {
				logs.Error("关闭kafka同步producer出错", zap.Error(e))
			}
			if e := consumer.Close(); e != nil {
				logs.Error("关闭kafka consumer出错", zap.Error(e))
			}
			if e := client.Close(); e != nil {
				logs.Error("关闭kafka client出错", zap.Error(e))
			}
			return nil
		},
	})
	return client, asyncProducer, syncProducer, consumer, nil
}

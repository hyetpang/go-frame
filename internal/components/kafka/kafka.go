package kafka

import (
	"context"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/hyetpang/go-frame/internal/adapter/log"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(lc fx.Lifecycle, zapLog *zap.Logger) (sarama.Client, sarama.AsyncProducer, sarama.SyncProducer, sarama.Consumer, error) {
	conf := new(config)
	err := viper.UnmarshalKey("kafka", &conf)
	if err != nil {
		logs.Fatal("kafka配置Unmarshal到对象出错", zap.Error(err), zap.Any("conf", conf))
	}
	common.MustValidate(conf)
	sarama.Logger = log.NewKafkaLog(zapLog)
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll          // 发送完数据需要leader和follow都确认
	config.Producer.Partitioner = sarama.NewRandomPartitioner // 新选出一个partition
	config.Producer.Return.Successes = true
	if len(conf.ClientId) > 0 {
		config.ClientID = conf.ClientId
	}
	// 连接kafka
	client, err := sarama.NewClient(strings.Split(conf.Addr, ","), config)
	if err != nil {
		logs.Fatal("连接kafka出错", zap.Error(err), zap.Any("conf", conf))
	}
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		logs.Fatal("连接kafka出错", zap.Error(err), zap.Any("conf", conf))
	}
	asyncProducer, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		logs.Fatal("连接kafka出错", zap.Error(err), zap.Any("conf", conf))
	}
	syncProducer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		logs.Fatal("连接kafka出错", zap.Error(err), zap.Any("conf", conf))
	}
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return nil
		},
		OnStop: func(context.Context) error {
			err = asyncProducer.Close()
			if err != nil {
				logs.Error("关闭kafka异步producer出错", zap.Error(err))
			}
			err = syncProducer.Close()
			if err != nil {
				logs.Error("关闭kafka同步producer出错", zap.Error(err))
			}
			err = consumer.Close()
			if err != nil {
				logs.Error("关闭kafka consumer出错", zap.Error(err))
			}
			err = client.Close()
			if err != nil {
				logs.Error("关闭kafka client出错", zap.Error(err))
			}
			return nil
		},
	})
	return client, asyncProducer, syncProducer, consumer, nil
}

package kafka

import (
	"fmt"
	"strings"

	"github.com/IBM/sarama"
	"github.com/hyetpang/go-frame/internal/adapter/log"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewClient 创建 sarama.Client,作为 Producer/Consumer 的共享底座。
// 拆分原因:历史 New 一次性创建 Client+SyncProducer+AsyncProducer+Consumer,
// 强制纯生产或纯消费服务也要建立全部资源。现按需 Provide,各自管理 fx 生命周期。
func NewClient(lc fx.Lifecycle, zapLog *zap.Logger, conf *config) (sarama.Client, error) {
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("kafka配置验证不通过: %w", err)
	}
	sarama.Logger = log.NewKafkaLog(zapLog)
	saramaCfg, err := buildSaramaConfig(conf)
	if err != nil {
		return nil, err
	}
	client, err := sarama.NewClient(strings.Split(conf.Addr, ","), saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("连接kafka出错 addr=%s: %w", conf.Addr, err)
	}
	lc.Append(fx.StopHook(func() {
		if e := client.Close(); e != nil {
			logs.Error("关闭kafka client出错", zap.Error(e))
		}
	}))
	return client, nil
}

// NewSyncProducer 基于共享 sarama.Client 创建同步 Producer,关闭顺序先于 Client。
func NewSyncProducer(lc fx.Lifecycle, client sarama.Client) (sarama.SyncProducer, error) {
	syncProducer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("创建kafka sync producer出错: %w", err)
	}
	lc.Append(fx.StopHook(func() {
		if e := syncProducer.Close(); e != nil {
			logs.Error("关闭kafka同步producer出错", zap.Error(e))
		}
	}))
	return syncProducer, nil
}

// NewAsyncProducer 基于共享 sarama.Client 创建异步 Producer,关闭顺序先于 Client。
func NewAsyncProducer(lc fx.Lifecycle, client sarama.Client) (sarama.AsyncProducer, error) {
	asyncProducer, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("创建kafka async producer出错: %w", err)
	}
	lc.Append(fx.StopHook(func() {
		if e := asyncProducer.Close(); e != nil {
			logs.Error("关闭kafka异步producer出错", zap.Error(e))
		}
	}))
	return asyncProducer, nil
}

// NewConsumer 基于共享 sarama.Client 创建 Consumer,关闭顺序先于 Client。
func NewConsumer(lc fx.Lifecycle, client sarama.Client) (sarama.Consumer, error) {
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("创建kafka consumer出错: %w", err)
	}
	lc.Append(fx.StopHook(func() {
		if e := consumer.Close(); e != nil {
			logs.Error("关闭kafka consumer出错", zap.Error(e))
		}
	}))
	return consumer, nil
}

// buildSaramaConfig 构造 sarama.Config,保留默认 Producer 行为。
// SASL/TLS 在后续提交中按需叠加,避免一次性引入过多变更。
func buildSaramaConfig(conf *config) (*sarama.Config, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll          // 发送完数据需要leader和follow都确认
	saramaCfg.Producer.Partitioner = sarama.NewRandomPartitioner // 新选出一个partition
	saramaCfg.Producer.Return.Successes = true
	if len(conf.ClientID) > 0 {
		saramaCfg.ClientID = conf.ClientID
	}
	return saramaCfg, nil
}

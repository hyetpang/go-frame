package kafka

import (
	"context"
	"fmt"
	"strings"

	"github.com/IBM/sarama"
	"github.com/dnwe/otelsarama"
	"github.com/hyetpang/go-frame/internal/adapter/log"
	"github.com/hyetpang/go-frame/internal/lifecycle"
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
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if e := lifecycle.CloseWithContext(ctx, "kafka-client", client.Close); e != nil {
				logs.Error("关闭kafka client出错", zap.Error(e))
				return e
			}
			return nil
		},
	})
	return client, nil
}

// NewSyncProducer 基于共享 sarama.Client 创建同步 Producer,关闭顺序先于 Client。
// 通过 otelsarama.WrapSyncProducer 注入 OTel tracing。
func NewSyncProducer(lc fx.Lifecycle, client sarama.Client) (sarama.SyncProducer, error) {
	rawSyncProducer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("创建kafka sync producer出错: %w", err)
	}
	syncProducer := otelsarama.WrapSyncProducer(client.Config(), rawSyncProducer)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if e := lifecycle.CloseWithContext(ctx, "kafka-sync-producer", syncProducer.Close); e != nil {
				logs.Error("关闭kafka同步producer出错", zap.Error(e))
				return e
			}
			return nil
		},
	})
	return syncProducer, nil
}

// NewAsyncProducer 基于共享 sarama.Client 创建异步 Producer,关闭顺序先于 Client。
// 通过 otelsarama.WrapAsyncProducer 注入 OTel tracing。
func NewAsyncProducer(lc fx.Lifecycle, client sarama.Client) (sarama.AsyncProducer, error) {
	rawAsyncProducer, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("创建kafka async producer出错: %w", err)
	}
	asyncProducer := otelsarama.WrapAsyncProducer(client.Config(), rawAsyncProducer)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// AsyncProducer.Close 不返回 error,包一层适配 lifecycle 签名。
			closer := func() error { asyncProducer.Close(); return nil }
			if e := lifecycle.CloseWithContext(ctx, "kafka-async-producer", closer); e != nil {
				logs.Error("关闭kafka异步producer出错", zap.Error(e))
				return e
			}
			return nil
		},
	})
	return asyncProducer, nil
}

// NewConsumer 基于共享 sarama.Client 创建 Consumer,关闭顺序先于 Client。
// 通过 otelsarama.WrapConsumer 注入 OTel tracing。
func NewConsumer(lc fx.Lifecycle, client sarama.Client) (sarama.Consumer, error) {
	rawConsumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("创建kafka consumer出错: %w", err)
	}
	consumer := otelsarama.WrapConsumer(rawConsumer)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if e := lifecycle.CloseWithContext(ctx, "kafka-consumer", consumer.Close); e != nil {
				logs.Error("关闭kafka consumer出错", zap.Error(e))
				return e
			}
			return nil
		},
	})
	return consumer, nil
}

// buildSaramaConfig 在默认 Producer 配置之上叠加 SASL 与 TLS。
// SASL: Username 非空时启用,根据 Mechanism 选择 PLAIN/SCRAM-SHA-256/SCRAM-SHA-512。
// TLS:  TLS.Enable 时复用通用 BuildClientTLS 构造 *tls.Config。
// 两项均可独立开启,例如 SASL_PLAINTEXT 仅开 SASL,SASL_SSL 同时启用。
func buildSaramaConfig(conf *config) (*sarama.Config, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll          // 发送完数据需要leader和follow都确认
	saramaCfg.Producer.Partitioner = sarama.NewRandomPartitioner // 新选出一个partition
	saramaCfg.Producer.Return.Successes = true
	if len(conf.ClientID) > 0 {
		saramaCfg.ClientID = conf.ClientID
	}
	if conf.Username != "" {
		mechanism, err := resolveSASLMechanism(conf.Mechanism)
		if err != nil {
			return nil, err
		}
		saramaCfg.Net.SASL.Enable = true
		saramaCfg.Net.SASL.Handshake = true
		saramaCfg.Net.SASL.User = conf.Username
		saramaCfg.Net.SASL.Password = conf.Password
		saramaCfg.Net.SASL.Mechanism = mechanism
	}
	if conf.TLS.IsEnabled() {
		tlsCfg, err := conf.TLS.BuildClientTLS()
		if err != nil {
			return nil, fmt.Errorf("构建 kafka TLS 配置出错: %w", err)
		}
		saramaCfg.Net.TLS.Enable = true
		saramaCfg.Net.TLS.Config = tlsCfg
	}
	return saramaCfg, nil
}

// resolveSASLMechanism 将 toml 中的字符串映射为 sarama.SASLMechanism。
// 空字符串视为 PLAIN,与历史"明文连接"语义保持兼容。
func resolveSASLMechanism(name string) (sarama.SASLMechanism, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "", "PLAIN":
		return sarama.SASLTypePlaintext, nil
	case "SCRAM-SHA-256":
		return sarama.SASLTypeSCRAMSHA256, nil
	case "SCRAM-SHA-512":
		return sarama.SASLTypeSCRAMSHA512, nil
	default:
		return "", fmt.Errorf("不支持的 kafka SASL mechanism=%q,允许值: PLAIN/SCRAM-SHA-256/SCRAM-SHA-512", name)
	}
}

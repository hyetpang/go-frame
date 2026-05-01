package options

import (
	"github.com/hyetpang/go-frame/internal/components/kafka"
	"go.uber.org/fx"
)

// WithKafkaClient 仅注入 sarama.Client,适用于自行管理 Producer/Consumer 的场景。
func WithKafkaClient() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(kafka.NewClient))
	}
}

// WithKafkaSyncProducer 同时注入 sarama.Client 与 sarama.SyncProducer,
// 适用于"只发同步消息"的纯生产服务,免去为消费者额外建连。
func WithKafkaSyncProducer() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions,
			fx.Provide(kafka.NewClient),
			fx.Provide(kafka.NewSyncProducer),
		)
	}
}

// WithKafkaAsyncProducer 同时注入 sarama.Client 与 sarama.AsyncProducer。
func WithKafkaAsyncProducer() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions,
			fx.Provide(kafka.NewClient),
			fx.Provide(kafka.NewAsyncProducer),
		)
	}
}

// WithKafkaConsumer 同时注入 sarama.Client 与 sarama.Consumer,
// 适用于"只消费"的纯消费服务,无需创建 Producer 资源。
func WithKafkaConsumer() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions,
			fx.Provide(kafka.NewClient),
			fx.Provide(kafka.NewConsumer),
		)
	}
}

// WithKafka 同时注入 sarama.Client、SyncProducer、AsyncProducer、Consumer 四种依赖。
//
// Deprecated: 兼容老用法。新代码请按需选择
// WithKafkaClient/WithKafkaSyncProducer/WithKafkaAsyncProducer/WithKafkaConsumer,
// 避免纯生产或纯消费服务被迫建立全部资源。
func WithKafka() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions,
			fx.Provide(kafka.NewClient),
			fx.Provide(kafka.NewSyncProducer),
			fx.Provide(kafka.NewAsyncProducer),
			fx.Provide(kafka.NewConsumer),
		)
	}
}

package options

import (
	"github.com/HyetPang/go-frame/internal/components/kafka"
	"go.uber.org/fx"
)

// 使用kafka的底层sarama.Client, sarama.AsyncProducer, sarama.SyncProducer, sarama.Consumer 依赖
func WithKafka() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(kafka.New))
	}
}

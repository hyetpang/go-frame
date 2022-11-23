package options

import (
	"github.com/HyetPang/go-frame/internal/components/kafka"
	"go.uber.org/fx"
)

// 使用kafka的sarama.SyncProducer 依赖
// func WithKafkaSyncProducer() Option {
// 	return func(o *Options) {
// 		o.FxOptions = append(o.FxOptions, fx.Provide(
// 			func(client sarama.Client) (sarama.SyncProducer, error) {
// 				return sarama.NewSyncProducerFromClient(client)
// 			},
// 			kafka.NewKafka,
// 		))
// 	}
// }

// // 使用kafka的sarama.AsyncProducer 依赖
// func WithKafkaASyncProducer() Option {
// 	return func(o *Options) {
// 		o.FxOptions = append(o.FxOptions, fx.Provide(
// 			func(client sarama.Client) (sarama.AsyncProducer, error) {
// 				return sarama.NewAsyncProducerFromClient(client)
// 			},
// 			kafka.NewKafka,
// 		))
// 	}
// }

// // 使用kafka的sarama.Consumer 依赖
// func WithKafkaConsumer() Option {
// 	return func(o *Options) {
// 		o.FxOptions = append(o.FxOptions, fx.Provide(
// 			func(client sarama.Client) (sarama.Consumer, error) {
// 				return sarama.NewConsumerFromClient(client)
// 			},
// 			kafka.NewKafka,
// 		))
// 	}
// }

// 使用kafka的底层sarama.Client 依赖
func WithKafka() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(kafka.NewKafka))
	}
}

// // 使用kafka的底层sarama.Client,sarama.SyncProducer sarama.AsyncProducer sarama.Consumer 依赖
// func WithKafkaClients() Option {
// 	return func(o *Options) {
// 		o.FxOptions = append(o.FxOptions, fx.Provide(
// 			kafka.NewKafka,
// 			func(client sarama.Client, lc fx.Lifecycle) (sarama.Consumer, error) {
// 				consumer, err := sarama.NewConsumerFromClient(client)
// 				lc.Append(fx.Hook{
// 					OnStop: func(context.Context) error {
// 						return consumer.Close()
// 					},
// 				})
// 				return consumer, err
// 			},
// 			func(client sarama.Client, lc fx.Lifecycle) (sarama.AsyncProducer, error) {
// 				producer, err := sarama.NewAsyncProducerFromClient(client)
// 				lc.Append(fx.Hook{
// 					OnStop: func(context.Context) error {
// 						return producer.Close()
// 					},
// 				})
// 				return producer, err
// 			},
// 			func(client sarama.Client, lc fx.Lifecycle) (sarama.SyncProducer, error) {
// 				producer, err := sarama.NewSyncProducerFromClient(client)
// 				lc.Append(fx.Hook{
// 					OnStop: func(context.Context) error {
// 						return producer.Close()
// 					},
// 				})
// 				return producer, err
// 			},
// 		))
// 	}
// }

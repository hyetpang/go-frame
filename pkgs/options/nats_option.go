package options

import (
	"github.com/hyetpang/go-frame/internal/components/nats"
	"go.uber.org/fx"
)

// WithNats 仅注入 *nats.Conn,适用于核心 pub/sub、request/reply 场景。
func WithNats() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(nats.NewConn))
	}
}

// WithNatsJetStream 注入 *nats.Conn 与 jetstream.JetStream,
// 适用于可靠消息/持久化流场景。分层哲学与 WithKafkaClient/WithKafkaConsumer 一致。
// 注意:本选项已内置 *nats.Conn,请勿再与 WithNats 同时使用,
// 否则 fx 会因 *nats.Conn 被重复 provide 而启动报错。
func WithNatsJetStream() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions,
			fx.Provide(nats.NewConn),
			fx.Provide(nats.NewJetStream),
		)
	}
}

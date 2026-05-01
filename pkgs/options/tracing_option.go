package options

import (
	"github.com/hyetpang/go-frame/internal/components/tracing"
	"go.uber.org/fx"
)

// 启用 OpenTelemetry 分布式追踪。
//
// 调用此选项后，框架会根据配置文件中的 [tracing] 段创建 TracerProvider，
// 并将其注册为 OTel 全局对象供各 instrumentation 复用：
//   - gRPC server/client：otelgrpc StatsHandler
//   - HTTP（gin）：otelgin Middleware
//   - MySQL（gorm）：opentelemetry/tracing 插件
//   - Redis：redisotel.InstrumentTracing
//   - Kafka（sarama）：otelsarama Wrap*
//
// Tracing.Enable=false 时返回 noop TracerProvider，零成本回退；
// 因此推荐生产环境也调用 WithTracing()，按配置开关控制是否上报。
func WithTracing() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(tracing.New))
	}
}

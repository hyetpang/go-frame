package options

import (
	"github.com/HyetPang/go-frame/internal/components/redis"
	"go.uber.org/fx"
)

func WithRedis() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(redis.New))
	}
}

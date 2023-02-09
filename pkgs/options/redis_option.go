package options

import (
	"github.com/HyetPang/go-frame/internal/components/redis"
	"github.com/HyetPang/go-frame/pkgs/common"
	"go.uber.org/fx"
)

// 使用redis
func WithRedis() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(redis.New), fx.Invoke(common.InjectRedis))
	}
}

package options

import (
	"github.com/HyetPang/go-frame/internal/components/mysql"
	"go.uber.org/fx"
)

func WithMysql() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(mysql.New))
	}
}

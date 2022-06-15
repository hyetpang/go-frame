/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-04-30 16:26:27
 * @FilePath: \go-frame\pkgs\options\gin_option.go
 */
package options

import (
	"github.com/HyetPang/go-frame/internal/components/gin"
	"go.uber.org/fx"
)

// 使用gin框架
func WithHttp() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(gin.New))
	}
}

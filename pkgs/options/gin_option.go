/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-04-30 16:26:27
 * @FilePath: \go-frame\pkgs\options\gin_option.go
 */
package options

import (
	"github.com/HyetPang/go-frame/internal/components/gin"
	"github.com/swaggo/swag"
	"go.uber.org/fx"
)

// 使用gin框架，这里传递的文档参数主要用来注册文档，如果不传，在配置文件开启或者关闭文档，那么文档也不起作用
// 即使传了生成的文档对象，如果配置文件关闭了，那么也不起作用，所以在代码中就直接传递参数，使用配置文件来控制文档的关闭
func WithHttp(_ *swag.Spec) Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(gin.New))
	}
}

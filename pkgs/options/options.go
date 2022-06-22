/*
 * @Date: 2022-04-24 09:56:04
 * @LastEditTime: 2022-04-29 18:32:17
 * @FilePath: /github.com/HyetPang/go-frame/pkgs/options/options.go
 */
package options

import (
	"go.uber.org/fx"
)

type Options struct {
	FxOptions  []fx.Option
	ConfigFile string
}
type Option func(*Options)

// 注册需要be注入的对象
func WithProviders(providers ...any) Option {
	return func(o *Options) {
		for _, provider := range providers {
			o.FxOptions = append(o.FxOptions, fx.Provide(provider))
		}
	}
}

// 注册需要被调用的函数
func WithInvokes(Invokes ...any) Option {
	return func(o *Options) {
		for _, Invoke := range Invokes {
			o.FxOptions = append(o.FxOptions, fx.Invoke(Invoke))
		}
	}
}

// 一次注册单个fx.Option
func WithFxOption(fxOptions ...fx.Option) Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fxOptions...)
	}
}

// 一次注册多个fx.Option
func WithFxOptions(fxOptions ...[]fx.Option) Option {
	return func(o *Options) {
		for _, op := range fxOptions {
			o.FxOptions = append(o.FxOptions, op...)
		}
	}
}

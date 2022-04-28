package options

import (
	"go.uber.org/fx"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	FxOptions  []fx.Option
	ConfigFile string
	LogFile    string
	LogLevel   zapcore.Level
}
type Option func(*Options)

func WithProviders(providers ...any) Option {
	return func(o *Options) {
		for _, provider := range providers {
			o.FxOptions = append(o.FxOptions, fx.Provide(provider))
		}
	}
}

func WithInvokes(Invokes ...any) Option {
	return func(o *Options) {
		for _, Invoke := range Invokes {
			o.FxOptions = append(o.FxOptions, fx.Invoke(Invoke))
		}
	}
}

func WithFxOption(fxOptions ...fx.Option) Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fxOptions...)
	}
}

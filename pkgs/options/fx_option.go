package options

// 使用start运行，运行完毕自行退出
func WithStart() Option {
	return func(o *Options) {
		o.IsStart = true
	}
}

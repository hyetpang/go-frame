package options

// WithHttp 启用 gin HTTP 服务组件；swagger 文档通过配置文件 http.is_doc 开关控制。
func WithHttp() Option {
	return func(o *Options) {
		o.UseHttp = true
	}
}

package options

// 配置文件路径,默认是./conf/app.toml路径
func WithConfigFile(configFile string) Option {
	return func(o *Options) {
		o.ConfigFile = configFile
	}
}

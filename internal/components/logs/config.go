package logs

// 日志相关配置
type config struct {
	File  string `mapstructure:"file"`  // 日志文件路径
	Level *int   `mapstructure:"level"` // 日志级别,级别参考zap.Level类型定义的值
	// WecomUrl string `mapstructure:"wecom_url"` // 错误日志企业微信通知url
	// // IsNotice bool   `mapstructure:"is_notice"` // 错误日志是否开启通知
}

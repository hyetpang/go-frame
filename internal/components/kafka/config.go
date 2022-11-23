package kafka

// kafka 配置
type config struct {
	Addr     string `mapstructure:"addr" validate:"required"` // 监听地址,多个地址使用逗号分隔
	ClientId string `mapstructure:"client_id"`                // kafka client id,可不传，默认是sarama
}

package lognotice

type config struct {
	WecomURL string `mapstructure:"wecom_url" validate:"required"` // 企业微信通知url
	Name     string `mapstructure:"name" validate:"required"`      // 通知名字,通常是服务名字
}

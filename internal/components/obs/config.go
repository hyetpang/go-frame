package obs

// 配置
type config struct {
	AK       string `mapstructure:"ak" validate:"required"`
	SK       string `mapstructure:"sk" validate:"required"`
	Endpoint string `mapstructure:"endpoint" validate:"required"`
}

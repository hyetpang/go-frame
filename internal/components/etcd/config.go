package etcd

// 配置
type config struct {
	Addresses string `mapstructure:"addresses" validate:"required"` // etcd地址,多个地址逗号分隔
}

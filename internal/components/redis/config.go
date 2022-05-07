package redis

type config struct {
	Addr string `mapstructure:"addr" validate:"required"` // 连接地址
	Pwd  string `mapstructure:"pwd"`                      // 连接密码
	DB   int    `mapstructure:"db" validate:"min=0"`      // 连接的数据库
}

package redis

type config struct {
	Addr string `mapstructure:"addr"` // 连接地址
	Pwd  string `mapstructure:"pwd"`  // 连接密码
	DB   int    `mapstructure:"db"`   // 连接的数据库
}

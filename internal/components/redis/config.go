package redis

type config struct {
	Addr string `toml:"addr"` // 连接地址
	Pwd  string `toml:"pwd"`  // 连接密码
	DB   int    `toml:"db"`   // 连接的数据库
}

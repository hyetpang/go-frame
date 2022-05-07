package mysql

type config struct {
	ConnectString string `toml:"connect_string"`
	MaxIdleTime   int    `toml:"max_idle_time"`
	MaxLifeTime   int    `toml:"max_life_time"`
	MaxIdleConns  int    `toml:"max_idle_conns"`
	MaxOpenConns  int    `toml:"max_open_conns"`
	TablePrefix   string `toml:"table_prefix"`
}

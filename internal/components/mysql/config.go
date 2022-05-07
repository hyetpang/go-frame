package mysql

type config struct {
	ConnectString string `mapstructure:"connect_string"`
	MaxIdleTime   int    `mapstructure:"max_idle_time"`
	MaxLifeTime   int    `mapstructure:"max_life_time"`
	MaxIdleConns  int    `mapstructure:"max_idle_conns"`
	MaxOpenConns  int    `mapstructure:"max_open_conns"`
	TablePrefix   string `mapstructure:"table_prefix"`
}

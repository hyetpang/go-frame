package mysql

type config struct {
	ConnectString                    string `mapstructure:"connect_string" validate:"required"`
	TablePrefix                      string `mapstructure:"table_prefix"`
	Name                             string `mapstructure:"name" validate:"required"` // 必须存在一个name是default的配置
	MaxIdleTime                      int    `mapstructure:"max_idle_time"`
	MaxLifeTime                      int    `mapstructure:"max_life_time"`
	MaxIdleConns                     int    `mapstructure:"max_idle_conns"`
	MaxOpenConns                     int    `mapstructure:"max_open_conns"`
	GormLogLevel                     int    `mapstructure:"gorm_log_level" validate:"oneof=1 2 3 4"` // gorm日志级别,1=>静默，什么都不打印,2=>error,3=>warn,4=>info,sql在4级别打印
	GormLogIgnoreRecordNotFoundError bool   `mapstructure:"gorm_log_ignore_record_not_found_error"`  // 是否忽略gorm没找到记录的日志打印
}

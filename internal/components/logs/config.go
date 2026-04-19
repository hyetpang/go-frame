package logs

// 日志相关配置
type config struct {
	Path        string `mapstructure:"path"`                            // 日志文件路径
	ServiceName string `mapstructure:"service_name"`                    // 服务名称
	Level       int    `mapstructure:"level" validate:"oneof=-1 0 1 2"` // 日志级别,-1=>debug,0=>info,1=>warn,2=>error
	IsLogFile   bool   `mapstructure:"is_log_file"`                     // 是否记录文件
}

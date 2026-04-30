package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server         Server
	HTTP           HTTP
	MySQL          []MySQL
	Redis          Redis
	Mail           Mail
	SMS            SMS
	LogNotice      LogNotice
	ZapLog         ZapLog
	GRPC           GRPC
	Etcd           Etcd
	Kafka          Kafka
	Gout           Gout
	configFilePath string
}

type Server struct {
	RunMode string `mapstructure:"run_mode"`
}

type HTTP struct {
	Addr          string `mapstructure:"addr"`
	DocPath       string `mapstructure:"doc_path" validate:"required_with=IsDoc"`
	PprofPrefix   string `mapstructure:"pprof_prefix"`
	PprofUsername string `mapstructure:"pprof_username"`
	PprofPassword string `mapstructure:"pprof_password"`
	MetricsPath   string `mapstructure:"metrics_path" validate:"required_with=IsMetrics"`
	IsDoc         bool   `mapstructure:"is_doc"`
	IsPprof       bool   `mapstructure:"is_pprof"`
	IsMetrics     bool   `mapstructure:"is_metrics"`
	IsProd        bool   `mapstructure:"is_prod"`
}

type MySQL struct {
	ConnectString                    string `mapstructure:"connect_string" validate:"required"`
	TablePrefix                      string `mapstructure:"table_prefix"`
	Name                             string `mapstructure:"name" validate:"required"`
	MaxIdleTime                      int    `mapstructure:"max_idle_time"`
	MaxLifeTime                      int    `mapstructure:"max_life_time"`
	MaxIdleConns                     int    `mapstructure:"max_idle_conns"`
	MaxOpenConns                     int    `mapstructure:"max_open_conns"`
	GormLogLevel                     int    `mapstructure:"gorm_log_level" validate:"oneof=1 2 3 4"`
	GormLogIgnoreRecordNotFoundError bool   `mapstructure:"gorm_log_ignore_record_not_found_error"`
}

type Redis struct {
	Addr string `mapstructure:"addr" validate:"required"`
	Pwd  string `mapstructure:"pwd"`
	DB   int    `mapstructure:"db" validate:"min=0,max=15"`
}

type Mail struct {
	User string `mapstructure:"user"`
	Pass string `mapstructure:"pass"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type SMS struct {
	Secret string `mapstructure:"secret"`
	Group  string `mapstructure:"group"`
	URL    string `mapstructure:"url"`
}

type LogNotice struct {
	Notice             string `mapstructure:"notice" validate:"required"`
	Name               string `mapstructure:"name" validate:"required"`
	ChatID             string `mapstructure:"chat_id" validate:"required_if=NoticeType 4"`
	NoticeType         int    `mapstructure:"notice_type" validate:"required,oneof=1 2 3 4"`
	LimitWindowSeconds int    `mapstructure:"limit_window_seconds"`
	LimitMaxKeys       int    `mapstructure:"limit_max_keys"`
	IsLimitDisabled    bool   `mapstructure:"is_limit_disabled"`
}

type ZapLog struct {
	Path            string `mapstructure:"path"`
	ServiceName     string `mapstructure:"service_name"`
	Level           int    `mapstructure:"level" validate:"oneof=-1 0 1 2"`
	StacktraceLevel int    `mapstructure:"stacktrace_level" validate:"oneof=-1 0 1 2"`
	LogMaxSize      int    `mapstructure:"log_max_size"`
	LogMaxBackups   int    `mapstructure:"log_max_backups"`
	LogMaxAge       int    `mapstructure:"log_max_age"`
	IsLogFile       bool   `mapstructure:"is_log_file"`
}

type GRPC struct {
	Address       string   `mapstructure:"address"`
	ServicePrefix string   `mapstructure:"service_prefix"`
	ServiceNames  []string `mapstructure:"service_names"`
}

type Etcd struct {
	Addresses string `mapstructure:"addresses" validate:"required"`
}

type Kafka struct {
	Addr     string `mapstructure:"addr" validate:"required"`
	ClientID string `mapstructure:"client_id"`
}

type Gout struct {
	Debug   bool `mapstructure:"debug"`
	Timeout int  `mapstructure:"timeout"`
}

const (
	defaultGRPCServicePrefix           = "grpc_services"
	defaultZapLogMaxSize               = 128
	defaultZapLogMaxBackups            = 30
	defaultZapLogMaxAge                = 7
	defaultZapStacktraceLevel          = 1
	defaultLogNoticeLimitWindowSeconds = 60
	defaultLogNoticeLimitMaxKeys       = 1024
)

func Load(configFile string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	conf := &Config{configFilePath: configFile}
	if err := v.UnmarshalKey("server", &conf.Server); err != nil {
		return nil, fmt.Errorf("server配置Unmarshal到对象出错: %w", err)
	}
	if err := v.UnmarshalKey("http", &conf.HTTP); err != nil {
		return nil, fmt.Errorf("http配置Unmarshal到对象出错: %w", err)
	}
	if err := unmarshalMySQL(v, conf); err != nil {
		return nil, err
	}
	if err := v.UnmarshalKey("redis", &conf.Redis); err != nil {
		return nil, fmt.Errorf("redis配置Unmarshal到对象出错: %w", err)
	}
	if err := v.UnmarshalKey("mail", &conf.Mail); err != nil {
		return nil, fmt.Errorf("mail配置Unmarshal到对象出错: %w", err)
	}
	if err := v.UnmarshalKey("sms", &conf.SMS); err != nil {
		return nil, fmt.Errorf("sms配置Unmarshal到对象出错: %w", err)
	}
	if err := v.UnmarshalKey("log_notice", &conf.LogNotice); err != nil {
		return nil, fmt.Errorf("log_notice配置Unmarshal到对象出错: %w", err)
	}
	if err := v.UnmarshalKey("zap_log", &conf.ZapLog); err != nil {
		return nil, fmt.Errorf("zap_log配置Unmarshal到对象出错: %w", err)
	}
	if err := v.UnmarshalKey("grpc", &conf.GRPC); err != nil {
		return nil, fmt.Errorf("grpc配置Unmarshal到对象出错: %w", err)
	}
	if err := v.UnmarshalKey("etcd", &conf.Etcd); err != nil {
		return nil, fmt.Errorf("etcd配置Unmarshal到对象出错: %w", err)
	}
	if err := v.UnmarshalKey("kafka", &conf.Kafka); err != nil {
		return nil, fmt.Errorf("kafka配置Unmarshal到对象出错: %w", err)
	}
	if err := v.UnmarshalKey("gout", &conf.Gout); err != nil {
		return nil, fmt.Errorf("gout配置Unmarshal到对象出错: %w", err)
	}
	conf.applyDefaults()
	return conf, nil
}

func (conf *Config) ConfigFilePath() string {
	return conf.configFilePath
}

func unmarshalMySQL(v *viper.Viper, conf *Config) error {
	if err := v.UnmarshalKey("mysql", &conf.MySQL); err == nil && len(conf.MySQL) > 0 {
		return nil
	}

	var one MySQL
	if err := v.UnmarshalKey("mysql", &one); err != nil {
		return fmt.Errorf("mysql配置Unmarshal到对象出错: %w", err)
	}
	if one != (MySQL{}) {
		conf.MySQL = []MySQL{one}
	}
	return nil
}

func SectionProviders() []any {
	return []any{
		provideServer,
		provideHTTP,
		provideMySQL,
		provideRedis,
		provideLogNotice,
		provideZapLog,
		provideGRPC,
		provideEtcd,
		provideKafka,
		provideGout,
	}
}

func (conf *Config) applyDefaults() {
	if conf.GRPC.ServicePrefix == "" {
		conf.GRPC.ServicePrefix = defaultGRPCServicePrefix
	}
	conf.ZapLog.applyDefaults()
	conf.LogNotice.applyDefaults()
}

func (conf *ZapLog) applyDefaults() {
	if conf.LogMaxSize == 0 {
		conf.LogMaxSize = defaultZapLogMaxSize
	}
	if conf.LogMaxBackups == 0 {
		conf.LogMaxBackups = defaultZapLogMaxBackups
	}
	if conf.LogMaxAge == 0 {
		conf.LogMaxAge = defaultZapLogMaxAge
	}
	if conf.StacktraceLevel == 0 {
		conf.StacktraceLevel = defaultZapStacktraceLevel
	}
}

func (conf *LogNotice) applyDefaults() {
	if conf.LimitWindowSeconds == 0 {
		conf.LimitWindowSeconds = defaultLogNoticeLimitWindowSeconds
	}
	if conf.LimitMaxKeys == 0 {
		conf.LimitMaxKeys = defaultLogNoticeLimitMaxKeys
	}
}

func provideServer(conf *Config) *Server       { return &conf.Server }
func provideHTTP(conf *Config) *HTTP           { return &conf.HTTP }
func provideMySQL(conf *Config) []MySQL        { return conf.MySQL }
func provideRedis(conf *Config) *Redis         { return &conf.Redis }
func provideLogNotice(conf *Config) *LogNotice { return &conf.LogNotice }
func provideZapLog(conf *Config) *ZapLog       { return &conf.ZapLog }
func provideGRPC(conf *Config) *GRPC           { return &conf.GRPC }
func provideEtcd(conf *Config) *Etcd           { return &conf.Etcd }
func provideKafka(conf *Config) *Kafka         { return &conf.Kafka }
func provideGout(conf *Config) *Gout           { return &conf.Gout }

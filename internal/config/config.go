package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	Server    Server    `mapstructure:"server"`
	HTTP      HTTP      `mapstructure:"http"`
	MySQL     []MySQL   `mapstructure:"-"` // mysql 段允许单实例与数组两种写法,统一在 unmarshalMySQL 中处理
	Redis     Redis     `mapstructure:"redis"`
	Mail      Mail      `mapstructure:"mail"`
	SMS       SMS       `mapstructure:"sms"`
	LogNotice LogNotice `mapstructure:"log_notice"`
	ZapLog    ZapLog    `mapstructure:"zap_log"`
	GRPC      GRPC      `mapstructure:"grpc"`
	Etcd      Etcd      `mapstructure:"etcd"`
	Kafka     Kafka     `mapstructure:"kafka"`
	Gout      Gout      `mapstructure:"gout"`

	configFilePath string `mapstructure:"-"`
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

// EnvVar 是 LoadWithEnv 读取的环境变量名,用来推导环境覆盖配置文件后缀。
const EnvVar = "APP_ENV"

// Load 读取并解析单个 toml 配置文件,等价于早期版本的逐段 UnmarshalKey,
// 但通过 viper.Unmarshal 一次性完成除 mysql 外所有段的反序列化。
// mysql 段因兼容单实例与数组两种写法,在 unmarshalMySQL 中单独处理。
func Load(configFile string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	return buildConfig(v, configFile)
}

// LoadWithEnv 在 Load 基础上叠加环境覆盖文件:
//
//	app.toml + app.${APP_ENV}.toml(若存在)
//
// 文件名按 baseFile 推导,例如 baseFile=./conf/app.toml、APP_ENV=dev 时,
// 会尝试合并 ./conf/app.dev.toml。环境覆盖文件不存在时静默跳过,保留 baseFile 的语义。
// 当 APP_ENV 为空时,行为与 Load(baseFile) 等价,便于向后兼容。
func LoadWithEnv(baseFile string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(baseFile)
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	if env := strings.TrimSpace(os.Getenv(EnvVar)); env != "" {
		envFile := envOverlayPath(baseFile, env)
		if _, err := os.Stat(envFile); err == nil {
			v.SetConfigFile(envFile)
			if mergeErr := v.MergeInConfig(); mergeErr != nil {
				return nil, fmt.Errorf("合并环境配置 %s 出错: %w", envFile, mergeErr)
			}
			// MergeInConfig 把当前 ConfigFile 切到 overlay,这里恢复回 baseFile 以便外部观感正确
			v.SetConfigFile(baseFile)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("读取环境配置 %s 出错: %w", envFile, err)
		}
	}

	return buildConfig(v, baseFile)
}

// envOverlayPath 根据 baseFile 与 env 推导环境覆盖文件路径。
// 例如 base="conf/app.toml" + env="dev" -> "conf/app.dev.toml"。
// base 没有扩展名时按 base + "." + env 处理,确保不丢信息。
func envOverlayPath(baseFile, env string) string {
	dir := filepath.Dir(baseFile)
	name := filepath.Base(baseFile)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	if ext == "" {
		return filepath.Join(dir, stem+"."+env)
	}
	return filepath.Join(dir, stem+"."+env+ext)
}

// reloadCallbacks 保护 watch 注册的回调列表,避免并发触发 onChange 时数据竞争。
type reloadCallbacks struct {
	mu  sync.RWMutex
	fns []func(*Config)
}

func (r *reloadCallbacks) add(fn func(*Config)) {
	r.mu.Lock()
	r.fns = append(r.fns, fn)
	r.mu.Unlock()
}

func (r *reloadCallbacks) snapshot() []func(*Config) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]func(*Config), len(r.fns))
	copy(out, r.fns)
	return out
}

// 全局 reload 回调表;一个进程里 Load/LoadWithEnv + WatchAndReload 通常只调用一次,
// 这里用包级变量把回调与 watch 解耦,保持 Config 结构体本身可序列化、可拷贝。
var globalReloadCallbacks reloadCallbacks

// WatchAndReload 监听 configFilePath 的变更,文件变化时重新调用 Load 解析新配置,
// 并依次同步执行所有注册的回调,把新 *Config 交给业务方决定如何应用。
//
// 重要:本方法不会替换任何已经注入到 fx 容器中的单例指针 — 业务方需要在回调里
// 自行用 atomic.Value 等机制更新 *ZapLog / *Gout 等被消费的字段,从而保证 fx
// 单例的引用不失效。一次只能 Watch 一个文件,后续调用会追加回调。
//
// 注:这是"打基础"的能力 — 仅观察 baseFile 的变化,不会重新 merge 环境覆盖文件。
func (conf *Config) WatchAndReload(onChange func(*Config)) {
	if onChange == nil {
		return
	}
	globalReloadCallbacks.add(onChange)
	startWatchOnce(conf.configFilePath)
}

var startWatchOnce = func() func(string) {
	var once sync.Once
	return func(path string) {
		once.Do(func() {
			if path == "" {
				return
			}
			v := viper.New()
			v.SetConfigFile(path)
			v.SetConfigType("toml")
			if err := v.ReadInConfig(); err != nil {
				return
			}
			v.OnConfigChange(func(_ fsnotify.Event) {
				newConf, err := buildConfig(v, path)
				if err != nil {
					return
				}
				for _, fn := range globalReloadCallbacks.snapshot() {
					fn(newConf)
				}
			})
			v.WatchConfig()
		})
	}
}()

// buildConfig 把已经 ReadInConfig 完成的 viper 实例反序列化成 *Config,
// 抽出来便于 LoadWithEnv 等多文件合并入口复用。
func buildConfig(v *viper.Viper, configFile string) (*Config, error) {
	conf := &Config{configFilePath: configFile}
	if err := v.Unmarshal(conf); err != nil {
		return nil, fmt.Errorf("配置Unmarshal到对象出错: %w", err)
	}
	if err := unmarshalMySQL(v, conf); err != nil {
		return nil, err
	}
	conf.applyDefaults()
	return conf, nil
}

func (conf *Config) ConfigFilePath() string {
	return conf.configFilePath
}

// unmarshalMySQL 兼容 toml 中两种 mysql 配置写法:
//  1. [[mysql]] 数组形式 -> 直接命中 []MySQL
//  2. [mysql]   单表形式  -> 视作长度为 1 的数组
//
// 是否真有配置改用必填字段显式判定,避免依赖结构体相等比较 — 后续给 MySQL 新增 bool 等
// 字段时,one != (MySQL{}) 会因零值差异误判,而 Name/ConnectString 都是必填的稳定锚点。
func unmarshalMySQL(v *viper.Viper, conf *Config) error {
	if err := v.UnmarshalKey("mysql", &conf.MySQL); err == nil && len(conf.MySQL) > 0 {
		return nil
	}
	conf.MySQL = nil

	var one MySQL
	if err := v.UnmarshalKey("mysql", &one); err != nil {
		return fmt.Errorf("mysql配置Unmarshal到对象出错: %w", err)
	}
	if one.Name != "" || one.ConnectString != "" {
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

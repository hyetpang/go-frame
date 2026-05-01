package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/hyetpang/go-frame/pkgs/logs"
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
	Tracing   Tracing   `mapstructure:"tracing"`

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
	MaxBodyBytes  int64  `mapstructure:"max_body_bytes"`
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
	Addr         string    `mapstructure:"addr" validate:"required"`
	Pwd          string    `mapstructure:"pwd"`
	Username     string    `mapstructure:"username"` // Redis 6+ ACL 用户名,留空兼容旧版本
	DB           int       `mapstructure:"db" validate:"min=0,max=15"`
	PoolSize     int       `mapstructure:"pool_size"`      // 连接池大小,0 时由 applyDefaults 取 10*GOMAXPROCS
	MinIdleConns int       `mapstructure:"min_idle_conns"` // 最小空闲连接数,0 时取默认 5
	DialTimeout  int       `mapstructure:"dial_timeout"`   // 拨号超时,单位秒,0 时取默认 5
	ReadTimeout  int       `mapstructure:"read_timeout"`   // 读超时,单位秒,0 时取默认 5
	WriteTimeout int       `mapstructure:"write_timeout"`  // 写超时,单位秒,0 时取默认 5
	TLS          TLSConfig `mapstructure:"tls"`
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
	Notice             string   `mapstructure:"notice" validate:"required"`
	Name               string   `mapstructure:"name" validate:"required"`
	ChatID             string   `mapstructure:"chat_id" validate:"required_if=NoticeType 4"`
	AllowedHosts       []string `mapstructure:"allowed_hosts"`
	NoticeType         int      `mapstructure:"notice_type" validate:"required,oneof=1 2 3 4"`
	LimitWindowSeconds int      `mapstructure:"limit_window_seconds"`
	LimitMaxKeys       int      `mapstructure:"limit_max_keys"`
	IsLimitDisabled    bool     `mapstructure:"is_limit_disabled"`
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
	Address       string    `mapstructure:"address"`
	ServicePrefix string    `mapstructure:"service_prefix"`
	ServiceNames  []string  `mapstructure:"service_names"`
	ServerTLS     TLSConfig `mapstructure:"server_tls"`
	ClientTLS     TLSConfig `mapstructure:"client_tls"`
}

type Etcd struct {
	Addresses string    `mapstructure:"addresses" validate:"required"`
	Username  string    `mapstructure:"username"`
	Password  string    `mapstructure:"password"`
	TLS       TLSConfig `mapstructure:"tls"`
}

// TLSConfig 是 gRPC、etcd 等组件共用的 TLS 选项。
// Enable=false 时整个 TLS 关闭,保持向后兼容。
type TLSConfig struct {
	CAFile             string `mapstructure:"ca_file"`
	CertFile           string `mapstructure:"cert_file"`
	KeyFile            string `mapstructure:"key_file"`
	ServerName         string `mapstructure:"server_name"`
	Enable             bool   `mapstructure:"enable"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

// IsEnabled 返回是否启用 TLS。
func (c *TLSConfig) IsEnabled() bool {
	return c != nil && c.Enable
}

// BuildClientTLS 构建客户端使用的 *tls.Config。
// 调用前应确认 IsEnabled()==true。
func (c *TLSConfig) BuildClientTLS() (*tls.Config, error) {
	if c == nil || !c.Enable {
		return nil, nil
	}
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         c.ServerName,
		InsecureSkipVerify: c.InsecureSkipVerify, // #nosec G402 -- 由调用方在配置中显式开启
	}
	if c.CAFile != "" {
		caPEM, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("读取 CA 证书 %s 出错: %w", c.CAFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("CA 证书 %s 解析失败", c.CAFile)
		}
		tlsCfg.RootCAs = pool
	}
	if c.CertFile != "" || c.KeyFile != "" {
		if c.CertFile == "" || c.KeyFile == "" {
			return nil, errors.New("TLS cert_file 与 key_file 必须同时配置")
		}
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("加载客户端证书出错: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return tlsCfg, nil
}

// BuildServerTLS 构建服务端使用的 *tls.Config(必须配置 cert_file 与 key_file)。
func (c *TLSConfig) BuildServerTLS() (*tls.Config, error) {
	if c == nil || !c.Enable {
		return nil, nil
	}
	if c.CertFile == "" || c.KeyFile == "" {
		return nil, errors.New("启用 TLS 时必须配置 cert_file 与 key_file")
	}
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("加载服务端证书出错: %w", err)
	}
	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	if c.CAFile != "" {
		caPEM, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("读取 CA 证书 %s 出错: %w", c.CAFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("CA 证书 %s 解析失败", c.CAFile)
		}
		tlsCfg.ClientCAs = pool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return tlsCfg, nil
}

type Kafka struct {
	Addr      string    `mapstructure:"addr" validate:"required"`
	ClientID  string    `mapstructure:"client_id"`
	Username  string    `mapstructure:"username"`  // SASL 用户名,留空表示明文连接
	Password  string    `mapstructure:"password"`  // SASL 密码
	Mechanism string    `mapstructure:"mechanism"` // PLAIN(默认) / SCRAM-SHA-256 / SCRAM-SHA-512
	TLS       TLSConfig `mapstructure:"tls"`
}

type Gout struct {
	Debug   bool `mapstructure:"debug"`
	Timeout int  `mapstructure:"timeout"`
}

// Tracing 用于配置 OpenTelemetry 分布式追踪。
type Tracing struct {
	// ServiceName 上报到 collector 的服务名，留空时回落到 ZapLog.ServiceName。
	ServiceName string `mapstructure:"service_name"`
	// Endpoint OTLP collector 接收端点，例如 "localhost:4318"（HTTP）或 "localhost:4317"（gRPC）。
	Endpoint string `mapstructure:"endpoint"`
	// Protocol 传输协议，可选 "http" 或 "grpc"，留空时默认 "http"。
	Protocol string `mapstructure:"protocol" validate:"omitempty,oneof=http grpc"`
	// SampleRatio 采样率，取值范围 [0.0, 1.0]，留空或 0 时默认 1.0（全采样）。
	SampleRatio float64 `mapstructure:"sample_ratio" validate:"gte=0,lte=1"`
	// Headers OTLP exporter 请求附加的鉴权/路由 header，常见用法是写入
	// "Authorization": "Bearer xxx" 或 collector 要求的 API key。
	// 对 SaaS collector(Honeycomb/Grafana Cloud/NewRelic 等)是必备项。
	Headers map[string]string `mapstructure:"headers"`
	// TLS OTLP exporter 的 TLS 客户端配置，启用后由 TLSConfig 统一构建 *tls.Config。
	// 与 Insecure=true 互斥，二者同时为真时 newExporter 拒绝启动以避免误配。
	TLS TLSConfig `mapstructure:"tls"`
	// Enable 是否启用分布式追踪，关闭时使用 noop TracerProvider 保持向后兼容。
	Enable bool `mapstructure:"enable"`
	// Insecure OTLP exporter 是否走明文链路，仅在测试环境置 true。
	// 生产环境强烈建议改为 false 并配置 [tracing.tls],防止 SQL/参数/header 明文外发。
	Insecure bool `mapstructure:"insecure"`
}

const (
	defaultGRPCServicePrefix           = "grpc_services"
	defaultZapLogMaxSize               = 128
	defaultZapLogMaxBackups            = 30
	defaultZapLogMaxAge                = 7
	defaultZapStacktraceLevel          = 1
	defaultLogNoticeLimitWindowSeconds = 60
	defaultLogNoticeLimitMaxKeys       = 1024
	defaultHTTPMaxBodyBytes      int64 = 10 << 20 // 10 MiB
	defaultRedisMinIdleConns           = 5
	defaultRedisDialTimeoutSec         = 5
	defaultRedisReadTimeoutSec         = 5
	defaultRedisWriteTimeoutSec        = 5
	// defaultRedisPoolSizeMultiplier 用于按 GOMAXPROCS 推导 PoolSize:与 go-redis 默认一致 (10*GOMAXPROCS)。
	defaultRedisPoolSizeMultiplier = 10
	defaultTracingProtocol         = "http"
	defaultTracingSampleRatio      = 1.0
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

// reloadMetrics 持有热加载相关 prometheus 指标,用 sync.Once 保证只注册一次。
var (
	reloadMetricsOnce    sync.Once
	reloadSuccessCounter prometheus.Counter
	reloadFailedCounter  prometheus.Counter
)

func initReloadMetrics() {
	reloadMetricsOnce.Do(func() {
		reloadSuccessCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "config_reload_success_total",
			Help: "配置文件热加载成功次数",
		})
		reloadFailedCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "config_reload_failed_total",
			Help: "配置文件热加载失败次数",
		})
	})
}

// WatchAndReload 监听 configFilePath 的变更,文件变化时重新调用 Load 解析新配置,
// 并依次同步执行所有注册的回调,把新 *Config 交给业务方决定如何应用。
//
// 重要:本方法不会替换任何已经注入到 fx 容器中的单例指针 — 业务方需要在回调里
// 自行用 atomic.Value 等机制更新 *ZapLog / *Gout 等被消费的字段,从而保证 fx
// 单例的引用不失效。一次只能 Watch 一个文件,后续调用会追加回调。
//
// 注意: WatchAndReload 仅重读 baseFile,不重新合并 APP_ENV overlay。生产环境请勿依赖该方法做环境覆盖热更。
func (conf *Config) WatchAndReload(onChange func(*Config)) {
	if onChange == nil {
		return
	}
	globalReloadCallbacks.add(onChange)
	startWatchOnce(conf.configFilePath)
}

// watchPath 启动对 path 的文件监听,带 200ms 去抖动。
// 供 startWatchOnce 和测试共用。callbacks 为 nil 时使用 globalReloadCallbacks。
func watchPath(path string, callbacks *reloadCallbacks) {
	if path == "" {
		return
	}
	if callbacks == nil {
		callbacks = &globalReloadCallbacks
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return
	}
	initReloadMetrics()
	var (
		debounceTimer *time.Timer
		debounceMu    sync.Mutex
	)
	v.OnConfigChange(func(_ fsnotify.Event) {
		debounceMu.Lock()
		defer debounceMu.Unlock()
		if debounceTimer != nil {
			debounceTimer.Reset(200 * time.Millisecond)
			return
		}
		debounceTimer = time.AfterFunc(200*time.Millisecond, func() {
			debounceMu.Lock()
			debounceTimer = nil
			debounceMu.Unlock()

			// 用独立 viper 实例重新读取文件,避免复用 v(viper watch 内部
			// 已调用 ReadInConfig,解析失败时会保留旧数据导致无法感知错误)。
			fresh := viper.New()
			fresh.SetConfigFile(path)
			fresh.SetConfigType("toml")
			if err := fresh.ReadInConfig(); err != nil {
				reloadFailedCounter.Inc()
				logs.Error("配置热加载失败:文件解析出错", zap.Error(err), zap.String("path", path))
				return
			}
			newConf, err := buildConfig(fresh, path)
			if err != nil {
				reloadFailedCounter.Inc()
				logs.Error("配置热加载失败:Unmarshal 出错", zap.Error(err), zap.String("path", path))
				return
			}
			reloadSuccessCounter.Inc()
			for _, fn := range callbacks.snapshot() {
				fn(newConf)
			}
		})
	})
	v.WatchConfig()
}

var startWatchOnce = func() func(string) {
	var once sync.Once
	return func(path string) {
		once.Do(func() {
			watchPath(path, nil)
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
		return checkMySQLDuplicateNames(conf.MySQL)
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

// checkMySQLDuplicateNames 拒绝同名 [[mysql]] 配置,避免单实例 pickOneConfig 静默
// 命中第一条、多实例 newMysqls 才报错的不一致行为。空 Name 视作 default,与 pickOneConfig 默认选择保持一致。
func checkMySQLDuplicateNames(configs []MySQL) error {
	seen := make(map[string]struct{}, len(configs))
	for _, c := range configs {
		key := c.Name
		if key == "" {
			key = "default"
		}
		if _, dup := seen[key]; dup {
			return fmt.Errorf("mysql 配置存在重复 name=%s", key)
		}
		seen[key] = struct{}{}
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
		provideTracing,
	}
}

func (conf *Config) applyDefaults() {
	if conf.GRPC.ServicePrefix == "" {
		conf.GRPC.ServicePrefix = defaultGRPCServicePrefix
	}
	conf.HTTP.applyDefaults()
	conf.ZapLog.applyDefaults()
	conf.LogNotice.applyDefaults()
	conf.Redis.applyDefaults()
	conf.Tracing.applyDefaults(&conf.ZapLog)
}

// applyDefaults 给 Redis 连接池/超时未显式配置时填默认值,
// PoolSize 缺省按 go-redis 推荐取 10*GOMAXPROCS。
func (conf *Redis) applyDefaults() {
	if conf.PoolSize == 0 {
		conf.PoolSize = defaultRedisPoolSizeMultiplier * runtime.GOMAXPROCS(0)
	}
	if conf.MinIdleConns == 0 {
		conf.MinIdleConns = defaultRedisMinIdleConns
	}
	if conf.DialTimeout == 0 {
		conf.DialTimeout = defaultRedisDialTimeoutSec
	}
	if conf.ReadTimeout == 0 {
		conf.ReadTimeout = defaultRedisReadTimeoutSec
	}
	if conf.WriteTimeout == 0 {
		conf.WriteTimeout = defaultRedisWriteTimeoutSec
	}
}

func (conf *HTTP) applyDefaults() {
	if conf.MaxBodyBytes <= 0 {
		conf.MaxBodyBytes = defaultHTTPMaxBodyBytes
	}
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

func (conf *Tracing) applyDefaults(zapLog *ZapLog) {
	if conf.Protocol == "" {
		conf.Protocol = defaultTracingProtocol
	}
	if conf.SampleRatio <= 0 {
		conf.SampleRatio = defaultTracingSampleRatio
	}
	if conf.ServiceName == "" && zapLog != nil {
		conf.ServiceName = zapLog.ServiceName
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
func provideTracing(conf *Config) *Tracing     { return &conf.Tracing }

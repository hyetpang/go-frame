package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoadReadsTypedSectionsWithoutGlobalViper(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.toml")
	err := os.WriteFile(path, []byte(`
[server]
run_mode = "dev"

[http]
addr = ":8081"
is_pprof = true
pprof_username = "admin"
pprof_password = "secret"

[redis]
addr = "127.0.0.1:6379"
db = 1

[mysql]
connect_string = "user:pass@tcp(127.0.0.1:3306)/db"
name = "default"
gorm_log_level = 4

[zap_log]
level = -1
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	conf, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if conf.Server.RunMode != "dev" {
		t.Fatalf("server run_mode = %q, want dev", conf.Server.RunMode)
	}
	if conf.HTTP.PprofUsername != "admin" {
		t.Fatalf("http pprof username = %q, want admin", conf.HTTP.PprofUsername)
	}
	if conf.Redis.DB != 1 {
		t.Fatalf("redis db = %d, want 1", conf.Redis.DB)
	}
	if len(conf.MySQL) != 1 || conf.MySQL[0].Name != "default" {
		t.Fatalf("mysql config = %+v, want one default config", conf.MySQL)
	}
	if conf.ZapLog.Level != -1 {
		t.Fatalf("zap log level = %d, want -1", conf.ZapLog.Level)
	}
}

func TestLoadReadsMultiMySQLConfigs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.toml")
	err := os.WriteFile(path, []byte(`
[[mysql]]
connect_string = "one"
name = "default"
gorm_log_level = 4

[[mysql]]
connect_string = "two"
name = "analytics"
gorm_log_level = 3
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	conf, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(conf.MySQL) != 2 {
		t.Fatalf("mysql config count = %d, want 2", len(conf.MySQL))
	}
	if conf.MySQL[1].Name != "analytics" {
		t.Fatalf("second mysql name = %q, want analytics", conf.MySQL[1].Name)
	}
}

func TestLoadReturnsErrorForMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err == nil {
		t.Fatal("expected missing config file error")
	}
}

func TestLoadReturnsErrorForInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte("[server\nrun_mode = \"dev\""), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected invalid config format error")
	}
}

func TestSectionProvidersExposeConfiguredSections(t *testing.T) {
	conf := &Config{
		Server: Server{RunMode: "dev"},
		HTTP:   HTTP{Addr: ":8080"},
	}

	if got := provideServer(conf); got.RunMode != "dev" {
		t.Fatalf("server run mode = %q, want dev", got.RunMode)
	}
	if got := provideHTTP(conf); got.Addr != ":8080" {
		t.Fatalf("http addr = %q, want :8080", got.Addr)
	}
}

func TestSectionProvidersDoNotExposeGraceRestart(t *testing.T) {
	if got, want := len(SectionProviders()), 11; got != want {
		t.Fatalf("section provider count = %d, want %d without graceful restart provider", got, want)
	}
}

func TestConfigDefaultsAreApplied(t *testing.T) {
	conf := &Config{}

	conf.applyDefaults()

	if conf.GRPC.ServicePrefix != "grpc_services" {
		t.Fatalf("grpc service prefix = %q, want grpc_services", conf.GRPC.ServicePrefix)
	}
	if conf.ZapLog.LogMaxSize == 0 || conf.ZapLog.LogMaxBackups == 0 || conf.ZapLog.LogMaxAge == 0 {
		t.Fatalf("zap log defaults not applied: %+v", conf.ZapLog)
	}
	if conf.LogNotice.LimitWindowSeconds == 0 || conf.LogNotice.LimitMaxKeys == 0 {
		t.Fatalf("log notice defaults not applied: %+v", conf.LogNotice)
	}
	if conf.Tracing.Protocol != "http" {
		t.Fatalf("tracing protocol = %q, want http", conf.Tracing.Protocol)
	}
	if conf.Tracing.SampleRatio != 1.0 {
		t.Fatalf("tracing sample ratio = %v, want 1.0", conf.Tracing.SampleRatio)
	}
}

func TestTracingDefaultsFallbackToZapLogServiceName(t *testing.T) {
	conf := &Config{ZapLog: ZapLog{ServiceName: "myapp"}}

	conf.applyDefaults()

	if conf.Tracing.ServiceName != "myapp" {
		t.Fatalf("tracing service name = %q, want myapp", conf.Tracing.ServiceName)
	}
}

func TestTracingDefaultsRespectExplicitServiceName(t *testing.T) {
	conf := &Config{
		ZapLog:  ZapLog{ServiceName: "myapp"},
		Tracing: Tracing{ServiceName: "explicit"},
	}

	conf.applyDefaults()

	if conf.Tracing.ServiceName != "explicit" {
		t.Fatalf("tracing service name = %q, want explicit", conf.Tracing.ServiceName)
	}
}

func TestLoadWithEnvMergesEnvironmentOverride(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "app.toml")
	dev := filepath.Join(dir, "app.dev.toml")

	// 基础配置:redis.db=1、http.addr=":8080"
	if err := os.WriteFile(base, []byte(`
[server]
run_mode = "prod"

[http]
addr = ":8080"

[redis]
addr = "127.0.0.1:6379"
db = 1

[mysql]
connect_string = "user:pass@tcp(127.0.0.1:3306)/db"
name = "default"
gorm_log_level = 4
`), 0600); err != nil {
		t.Fatal(err)
	}
	// dev 覆盖:server.run_mode=dev、http.addr=":9090",其它字段保留 base
	if err := os.WriteFile(dev, []byte(`
[server]
run_mode = "dev"

[http]
addr = ":9090"
`), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("APP_ENV", "dev")
	conf, err := LoadWithEnv(base)
	if err != nil {
		t.Fatalf("LoadWithEnv 返回错误: %v", err)
	}
	// dev 覆盖项生效
	if conf.Server.RunMode != "dev" {
		t.Fatalf("server run_mode = %q, want dev (来自 app.dev.toml 覆盖)", conf.Server.RunMode)
	}
	if conf.HTTP.Addr != ":9090" {
		t.Fatalf("http addr = %q, want :9090 (来自 app.dev.toml 覆盖)", conf.HTTP.Addr)
	}
	// base 未被覆盖项保留
	if conf.Redis.DB != 1 {
		t.Fatalf("redis db = %d, want 1 (来自 app.toml 基础配置)", conf.Redis.DB)
	}
	if len(conf.MySQL) != 1 || conf.MySQL[0].Name != "default" {
		t.Fatalf("mysql config = %+v, want one default config from base", conf.MySQL)
	}
	// configFilePath 仍指向基础文件,保持外部可观测语义
	if conf.ConfigFilePath() != base {
		t.Fatalf("ConfigFilePath = %q, want base %q", conf.ConfigFilePath(), base)
	}
}

func TestLoadWithEnvWithoutEnvFallsBackToBase(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "app.toml")
	if err := os.WriteFile(base, []byte(`
[server]
run_mode = "prod"

[http]
addr = ":8080"

[redis]
addr = "127.0.0.1:6379"
db = 0
`), 0600); err != nil {
		t.Fatal(err)
	}

	// APP_ENV 未设置时,应等价 Load(base)
	t.Setenv("APP_ENV", "")
	conf, err := LoadWithEnv(base)
	if err != nil {
		t.Fatalf("LoadWithEnv 返回错误: %v", err)
	}
	if conf.Server.RunMode != "prod" {
		t.Fatalf("server run_mode = %q, want prod", conf.Server.RunMode)
	}
}

func TestLoadWithEnvSkipsMissingOverlay(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "app.toml")
	if err := os.WriteFile(base, []byte(`
[server]
run_mode = "prod"

[http]
addr = ":8080"

[redis]
addr = "127.0.0.1:6379"
db = 0
`), 0600); err != nil {
		t.Fatal(err)
	}

	// staging 环境覆盖文件不存在时不应报错,直接使用 base
	t.Setenv("APP_ENV", "staging")
	conf, err := LoadWithEnv(base)
	if err != nil {
		t.Fatalf("LoadWithEnv 在 overlay 不存在时不应报错, got: %v", err)
	}
	if conf.Server.RunMode != "prod" {
		t.Fatalf("server run_mode = %q, want prod", conf.Server.RunMode)
	}
}

func TestUnmarshalMySQLDetectsConfigByExplicitFields(t *testing.T) {
	// 仅给 connect_string,验证不再依赖结构体相等比较来识别"是否有配置"
	dir := t.TempDir()
	path := filepath.Join(dir, "app.toml")
	if err := os.WriteFile(path, []byte(`
[mysql]
connect_string = "user:pass@tcp(127.0.0.1:3306)/db"
name = "default"
gorm_log_level = 4
gorm_log_ignore_record_not_found_error = false
`), 0600); err != nil {
		t.Fatal(err)
	}
	conf, err := Load(path)
	if err != nil {
		t.Fatalf("Load 返回错误: %v", err)
	}
	if len(conf.MySQL) != 1 {
		t.Fatalf("mysql config count = %d, want 1", len(conf.MySQL))
	}
	if conf.MySQL[0].Name != "default" {
		t.Fatalf("mysql name = %q, want default", conf.MySQL[0].Name)
	}
}

// TestUnmarshalMySQLRejectsDuplicateNames 验证 [[mysql]] 同名配置在 Load 阶段就被拦下,
// 避免单实例 pickOneConfig 静默命中第一条、多实例 newMysqls 才报错的不一致行为。
func TestUnmarshalMySQLRejectsDuplicateNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.toml")
	if err := os.WriteFile(path, []byte(`
[[mysql]]
name = "primary"
connect_string = "user:pass@tcp(127.0.0.1:3306)/a"
gorm_log_level = 4

[[mysql]]
name = "primary"
connect_string = "user:pass@tcp(127.0.0.1:3306)/b"
gorm_log_level = 4
`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("期望同名 [[mysql]] 配置返回错误,但 Load 通过了")
	}
}

func TestLoadExampleConfigs(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))

	for _, name := range []string{"app.dev.toml", "app.prod.toml"} {
		t.Run(name, func(t *testing.T) {
			conf, err := Load(filepath.Join(repoRoot, "example", "conf", name))
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}
			if conf.HTTP.Addr == "" {
				t.Fatal("expected http addr in example config")
			}
			if conf.ZapLog.LogMaxSize == 0 {
				t.Fatal("expected zap log defaults or explicit rotation config")
			}
		})
	}
}

// validToml 是测试用的最小合法 toml 配置。
const validToml = `
[server]
run_mode = "dev"

[http]
addr = ":8080"

[redis]
addr = "127.0.0.1:6379"
`

// TestWatchAndReloadLogsErrorOnInvalidContent 验证写入非法 toml 后:
// 1. onChange 回调不被调用
// 2. 错误通过 zap 全局 logger 记录(level=Error)
func TestWatchAndReloadLogsErrorOnInvalidContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.toml")
	if err := os.WriteFile(path, []byte(validToml), 0600); err != nil {
		t.Fatal(err)
	}

	// 用 observer 替换全局 zap logger,捕获日志输出
	core, recorded := observer.New(zapcore.ErrorLevel)
	original := zap.L()
	zap.ReplaceGlobals(zap.New(core))
	t.Cleanup(func() { zap.ReplaceGlobals(original) })

	// 初始化 reload 指标(避免 nil counter panic)
	initReloadMetrics()

	var callbacks reloadCallbacks
	var callCount atomic.Int32
	callbacks.add(func(_ *Config) {
		callCount.Add(1)
	})

	watchPath(path, &callbacks)

	// 写入非法 toml
	if err := os.WriteFile(path, []byte("[server\nrun_mode = \"bad\""), 0600); err != nil {
		t.Fatal(err)
	}

	// 等待去抖动 timer 触发(200ms)+ 留余量
	time.Sleep(500 * time.Millisecond)

	if callCount.Load() != 0 {
		t.Fatalf("onChange 被调用了 %d 次,期望 0(非法配置不应触发回调)", callCount.Load())
	}

	entries := recorded.All()
	if len(entries) == 0 {
		t.Fatal("期望记录到错误日志,但没有任何日志输出")
	}
	found := false
	for _, e := range entries {
		if e.Level == zapcore.ErrorLevel {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("期望 Error 级别日志,实际日志: %v", entries)
	}
}

// TestWatchAndReloadDebouncesRapidChanges 验证在 100ms 内连续触发 3 次文件变更,
// 去抖动后 onChange 只被调用一次。
func TestWatchAndReloadDebouncesRapidChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.toml")
	if err := os.WriteFile(path, []byte(validToml), 0600); err != nil {
		t.Fatal(err)
	}

	// 初始化 reload 指标
	initReloadMetrics()

	var callbacks reloadCallbacks
	var callCount atomic.Int32
	callbacks.add(func(_ *Config) {
		callCount.Add(1)
	})

	watchPath(path, &callbacks)

	// 在 100ms 内连续写入 3 次合法配置
	for i := 0; i < 3; i++ {
		content := validToml + "\n# change " + string(rune('0'+i)) + "\n"
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		time.Sleep(30 * time.Millisecond)
	}

	// 等待去抖动 timer 触发(200ms)+ 留余量
	time.Sleep(500 * time.Millisecond)

	if got := callCount.Load(); got != 1 {
		t.Fatalf("onChange 被调用了 %d 次,期望 1(去抖动应合并多次变更)", got)
	}
}

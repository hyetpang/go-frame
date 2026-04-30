package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
	if got, want := len(SectionProviders()), 10; got != want {
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

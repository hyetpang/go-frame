package config

import (
	"os"
	"path/filepath"
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

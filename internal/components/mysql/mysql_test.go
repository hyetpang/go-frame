package mysql

import (
	"testing"

	"github.com/spf13/viper"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNewOneReturnsErrorForInvalidConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("mysql", map[string]any{
		"name":           "default",
		"gorm_log_level": 4,
	})

	db, err := NewOne(zap.NewNop(), fxtest.NewLifecycle(t))
	if err == nil {
		t.Fatal("expected invalid mysql config to return error")
	}
	if db != nil {
		t.Fatal("expected nil mysql db on config error")
	}
}

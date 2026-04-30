package redis

import (
	"testing"

	"github.com/spf13/viper"
	"go.uber.org/fx/fxtest"
)

func TestNewReturnsErrorForInvalidConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("redis", map[string]any{
		"db": 16,
	})

	client, err := New(fxtest.NewLifecycle(t))
	if err == nil {
		t.Fatal("expected invalid redis config to return error")
	}
	if client != nil {
		t.Fatal("expected nil redis client on config error")
	}
}

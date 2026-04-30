package redis

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

func TestNewReturnsErrorForInvalidConfig(t *testing.T) {
	conf := &config{DB: 16}

	client, err := New(fxtest.NewLifecycle(t), conf)
	if err == nil {
		t.Fatal("expected invalid redis config to return error")
	}
	if client != nil {
		t.Fatal("expected nil redis client on config error")
	}
}

package mysql

import (
	"testing"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNewOneReturnsErrorForInvalidConfig(t *testing.T) {
	configs := []config{{
		Name:         "default",
		GormLogLevel: 4,
	}}

	db, err := NewOne(zap.NewNop(), fxtest.NewLifecycle(t), configs)
	if err == nil {
		t.Fatal("expected invalid mysql config to return error")
	}
	if db != nil {
		t.Fatal("expected nil mysql db on config error")
	}
}

func TestPickOneConfigSelectsDefaultDatabase(t *testing.T) {
	configs := []config{
		{Name: "analytics", ConnectString: "analytics", GormLogLevel: 4},
		{Name: "default", ConnectString: "default", GormLogLevel: 4},
	}

	conf, err := pickOneConfig(configs)
	if err != nil {
		t.Fatalf("pickOneConfig returned error: %v", err)
	}
	if conf.Name != "default" {
		t.Fatalf("selected mysql name = %q, want default", conf.Name)
	}
}

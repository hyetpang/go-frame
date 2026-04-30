package gin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func resetTestConfig(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

func setHTTPConfig(values map[string]any) {
	viper.Set("http", values)
}

func TestNewGinReturnsErrorWhenPprofHasNoAuth(t *testing.T) {
	resetTestConfig(t)
	setHTTPConfig(map[string]any{
		"addr":       "127.0.0.1:0",
		"is_pprof":   true,
		"is_metrics": false,
		"is_doc":     false,
	})

	_, _, err := newGin(zap.NewNop())
	if err == nil {
		t.Fatal("expected pprof without auth to return error")
	}
}

func TestNewGinProtectsPprofWithBasicAuth(t *testing.T) {
	resetTestConfig(t)
	setHTTPConfig(map[string]any{
		"addr":           "127.0.0.1:0",
		"is_pprof":       true,
		"pprof_username": "admin",
		"pprof_password": "secret",
		"is_metrics":     false,
		"is_doc":         false,
	})

	router, _, err := newGin(zap.NewNop())
	if err != nil {
		t.Fatalf("newGin returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rsp := httptest.NewRecorder()
	router.ServeHTTP(rsp, req)
	if rsp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated pprof status = %d, want %d", rsp.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.SetBasicAuth("admin", "secret")
	rsp = httptest.NewRecorder()
	router.ServeHTTP(rsp, req)
	if rsp.Code != http.StatusOK {
		t.Fatalf("authenticated pprof status = %d, want %d", rsp.Code, http.StatusOK)
	}
}

func TestNewStartsWithoutFixedOneSecondDelay(t *testing.T) {
	resetTestConfig(t)
	setHTTPConfig(map[string]any{
		"addr":       "127.0.0.1:0",
		"is_metrics": false,
		"is_doc":     false,
		"is_pprof":   false,
		"is_prod":    true,
	})
	lc := fxtest.NewLifecycle(t)

	_, err := New(zap.NewNop(), lc)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	if err := lc.Start(ctx); err != nil {
		t.Fatalf("lifecycle start returned error: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer stopCancel()
		_ = lc.Stop(stopCtx)
	})

	if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
		t.Fatalf("lifecycle start took %s, expected no fixed one second delay", elapsed)
	}
}

package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
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

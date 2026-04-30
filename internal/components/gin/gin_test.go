package gin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNewGinReturnsErrorWhenPprofHasNoAuth(t *testing.T) {
	conf := &config{
		Addr:    "127.0.0.1:0",
		IsPprof: true,
	}

	_, _, err := newGin(zap.NewNop(), conf)
	if err == nil {
		t.Fatal("expected pprof without auth to return error")
	}
}

func TestNewGinProtectsPprofWithBasicAuth(t *testing.T) {
	conf := &config{
		Addr:          "127.0.0.1:0",
		IsPprof:       true,
		PprofUsername: "admin",
		PprofPassword: "secret",
	}

	router, _, err := newGin(zap.NewNop(), conf)
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
	conf := &config{
		Addr:   "127.0.0.1:0",
		IsProd: true,
	}
	lc := fxtest.NewLifecycle(t)

	_, err := New(zap.NewNop(), lc, conf)
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

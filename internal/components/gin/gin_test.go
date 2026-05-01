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
	const (
		pprofUser = "ops-team"
		pprofPass = "Strong!Pprof#Password" // 长度 >= 12,且不在弱口令列表中
	)
	conf := &config{
		Addr:          "127.0.0.1:0",
		IsPprof:       true,
		PprofUsername: pprofUser,
		PprofPassword: pprofPass,
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
	req.SetBasicAuth(pprofUser, pprofPass)
	rsp = httptest.NewRecorder()
	router.ServeHTTP(rsp, req)
	if rsp.Code != http.StatusOK {
		t.Fatalf("authenticated pprof status = %d, want %d", rsp.Code, http.StatusOK)
	}
}

func TestNewGinRejectsWeakPprofPassword(t *testing.T) {
	cases := []struct {
		name     string
		username string
		password string
	}{
		{"empty password", "ops-team", ""},
		{"placeholder password", "ops-team", "CHANGE_ME"},
		{"placeholder password lowercase", "ops-team", "change_me"},
		{"common weak password", "ops-team", "secret"},
		{"common weak password mixed case", "ops-team", "Secret"},
		{"too short", "ops-team", "abc12345"},
		{"weak username admin", "admin", "Strong!Pprof#Password"},
		{"weak username uppercase", "ROOT", "Strong!Pprof#Password"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf := &config{
				Addr:          "127.0.0.1:0",
				IsPprof:       true,
				PprofUsername: tc.username,
				PprofPassword: tc.password,
			}
			if _, _, err := newGin(zap.NewNop(), conf); err == nil {
				t.Fatalf("expected weak pprof credentials to be rejected")
			}
		})
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

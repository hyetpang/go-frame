package gin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	gingonic "github.com/gin-gonic/gin"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNewGinReturnsErrorWhenPprofHasNoAuth(t *testing.T) {
	conf := &config{
		Addr:    "127.0.0.1:0",
		IsPprof: true,
	}

	_, _, err := newGin(zap.NewNop(), conf, nil)
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

	router, _, err := newGin(zap.NewNop(), conf, nil)
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
			if _, _, err := newGin(zap.NewNop(), conf, nil); err == nil {
				t.Fatalf("expected weak pprof credentials to be rejected")
			}
		})
	}
}

// TestNoRouteAndNoMethodDoNotTriggerNotice 验证 404/405 只记录日志、不触发告警通知钩子,
// 防止扫描器/爬虫刷不存在路由或非法方法时把 lognotice 告警链路打爆。
func TestNoRouteAndNoMethodDoNotTriggerNotice(t *testing.T) {
	var noticeCalls int32
	// 注册一个会计数的通知钩子;只有走 logs.Error(带通知)才会命中,
	// logs.ErrorWithoutNotice 不应触发它。
	logs.RegisterNoticeHook(func(msg string, filename string, line int, fields ...zap.Field) {
		atomic.AddInt32(&noticeCalls, 1)
	})
	// 测试结束后用空操作钩子覆盖,避免影响其它测试。
	t.Cleanup(func() {
		logs.RegisterNoticeHook(func(string, string, int, ...zap.Field) {})
	})

	conf := &config{
		Addr:   "127.0.0.1:0",
		IsProd: true,
	}
	router, _, err := newGin(zap.NewNop(), conf, nil)
	if err != nil {
		t.Fatalf("newGin returned error: %v", err)
	}

	// 不存在的路由 -> 404
	req := httptest.NewRequest(http.MethodGet, "/this-route-does-not-exist", nil)
	rsp := httptest.NewRecorder()
	router.ServeHTTP(rsp, req)

	// 直接触发 noMethodHandler,验证 405 分支也不走告警通知。
	// (gin.New() 默认 HandleMethodNotAllowed=false,这里直接调用 handler 以稳定覆盖该分支)
	req = httptest.NewRequest(http.MethodPost, "/health_check", nil)
	rsp = httptest.NewRecorder()
	c, _ := gingonic.CreateTestContext(rsp)
	c.Request = req
	noMethodHandler(c)
	if rsp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("noMethodHandler status = %d, want %d", rsp.Code, http.StatusMethodNotAllowed)
	}

	if calls := atomic.LoadInt32(&noticeCalls); calls != 0 {
		t.Fatalf("notice hook triggered %d times for 404/405, want 0", calls)
	}
}

func TestNewStartsWithoutFixedOneSecondDelay(t *testing.T) {
	conf := &config{
		Addr:   "127.0.0.1:0",
		IsProd: true,
	}
	lc := fxtest.NewLifecycle(t)

	_, err := New(zap.NewNop(), lc, conf, nil)
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

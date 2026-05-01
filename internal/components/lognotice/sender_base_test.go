package lognotice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSenderBaseSafeDialerRejectsLoopback 验证拨号期复检会拒绝指向 127.0.0.1 的请求,
// 即使启动期 validateWebhookURL 通过(未触发,因为 host 为 IP 字面量)。
// 这堵的是 DNS rebinding TOCTOU:启动校验的 IP 与运行时拨号的 IP 可能不同。
func TestSenderBaseSafeDialerRejectsLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	base := newSenderBase(nil) // 不配白名单 → 拨号期应拒绝 127.0.0.1
	err := base.postJSON(context.Background(), srv.URL, map[string]any{"a": 1}, nil)
	if err == nil {
		t.Fatal("期望 safeDialer 拒绝拨号到 127.0.0.1,但请求成功了")
	}
	if !strings.Contains(err.Error(), "私网/回环") {
		t.Fatalf("错误信息未提示私网/回环 IP: %v", err)
	}
}

// TestSenderBaseAllowsWhitelistedHost 验证白名单 host 跳过私网检查,
// 与 validateWebhookURL 行为一致(便于开发环境用 127.0.0.1)。
func TestSenderBaseAllowsWhitelistedHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	t.Cleanup(srv.Close)

	base := newSenderBase([]string{"127.0.0.1"})
	var resp struct {
		Errcode int `json:"errcode"`
	}
	if err := base.postJSON(context.Background(), srv.URL, map[string]any{"a": 1}, &resp); err != nil {
		t.Fatalf("白名单 host 应允许拨号: %v", err)
	}
	if resp.Errcode != 0 {
		t.Fatalf("响应反序列化错误: errcode=%d", resp.Errcode)
	}
}

// TestSenderBasePostJSONSendsAndDecodes 验证完整的 JSON 往返:
// Content-Type 正确、payload 序列化、响应反序列化。
func TestSenderBasePostJSONSendsAndDecodes(t *testing.T) {
	var receivedCT, receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCT = r.Header.Get("Content-Type")
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		_ = json.NewEncoder(w).Encode(map[string]any{"echo": "ok"})
	}))
	t.Cleanup(srv.Close)

	base := newSenderBase([]string{"127.0.0.1"})
	var resp map[string]string
	if err := base.postJSON(context.Background(), srv.URL, map[string]string{"k": "v"}, &resp); err != nil {
		t.Fatalf("postJSON 失败: %v", err)
	}
	if receivedCT != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", receivedCT)
	}
	if !strings.Contains(receivedBody, `"k":"v"`) {
		t.Fatalf("server 收到的 body = %q, 缺少 payload", receivedBody)
	}
	if resp["echo"] != "ok" {
		t.Fatalf("响应反序列化错误: %+v", resp)
	}
}

// TestSenderBasePostJSONReportsNon2xx 验证 webhook 返回 4xx/5xx 时返回带 body 的错误,
// 便于排查上游 webhook 接口给出的具体错误描述。
func TestSenderBasePostJSONReportsNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	}))
	t.Cleanup(srv.Close)

	base := newSenderBase([]string{"127.0.0.1"})
	err := base.postJSON(context.Background(), srv.URL, map[string]any{}, nil)
	if err == nil {
		t.Fatal("期望非 2xx 返回错误")
	}
	if !strings.Contains(err.Error(), "status=400") || !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("错误信息缺少 status/body 细节: %v", err)
	}
}

package lognotice

import (
	"strings"
	"testing"
)

func TestValidateWebhookURL_RejectsLoopbackIP(t *testing.T) {
	err := validateWebhookURL("https://127.0.0.1/hook", nil)
	if err == nil || !strings.Contains(err.Error(), "私网") {
		t.Fatalf("expected loopback IP rejection, got %v", err)
	}
}

func TestValidateWebhookURL_RejectsRFC1918IP(t *testing.T) {
	cases := []string{
		"https://10.0.0.1/x",
		"https://192.168.1.1/x",
		"https://172.16.0.1/x",
		"https://169.254.1.1/x",
	}
	for _, raw := range cases {
		if err := validateWebhookURL(raw, nil); err == nil {
			t.Fatalf("expected rejection for %s", raw)
		}
	}
}

func TestValidateWebhookURL_RejectsIPv6Loopback(t *testing.T) {
	err := validateWebhookURL("https://[::1]/hook", nil)
	if err == nil {
		t.Fatal("expected rejection for IPv6 loopback")
	}
}

func TestValidateWebhookURL_RejectsHTTPWhenNoAllowList(t *testing.T) {
	err := validateWebhookURL("http://qyapi.weixin.qq.com/x", nil)
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected https requirement error, got %v", err)
	}
}

func TestValidateWebhookURL_AllowsAllowListedHostExactMatch(t *testing.T) {
	err := validateWebhookURL(
		"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=foo",
		[]string{"qyapi.weixin.qq.com"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWebhookURL_AllowsSuffixMatch(t *testing.T) {
	err := validateWebhookURL(
		"https://api.telegram.org/bot/sendMessage",
		[]string{"telegram.org"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWebhookURL_RejectsHostNotInAllowList(t *testing.T) {
	err := validateWebhookURL(
		"https://evil.example.com/x",
		[]string{"qyapi.weixin.qq.com"},
	)
	if err == nil || !strings.Contains(err.Error(), "白名单") {
		t.Fatalf("expected allow-list rejection, got %v", err)
	}
}

func TestValidateWebhookURL_AllowListBypassesPrivateIPCheck(t *testing.T) {
	// 显式列入白名单的 host(IP 形式)允许走 http,即便是私网,用于开发场景。
	err := validateWebhookURL(
		"http://127.0.0.1/local-webhook",
		[]string{"127.0.0.1"},
	)
	if err != nil {
		t.Fatalf("unexpected error for explicit allow-listed loopback: %v", err)
	}
}

func TestValidateWebhookURL_RejectsEmptyURL(t *testing.T) {
	if err := validateWebhookURL("", nil); err == nil {
		t.Fatal("expected error for empty URL")
	}
	if err := validateWebhookURL("   ", nil); err == nil {
		t.Fatal("expected error for blank URL")
	}
}

func TestValidateWebhookURL_RejectsInvalidScheme(t *testing.T) {
	err := validateWebhookURL("ftp://example.com/x", nil)
	if err == nil {
		t.Fatal("expected error for non-https scheme")
	}
}

func TestValidateWebhookURL_RejectsMissingHost(t *testing.T) {
	err := validateWebhookURL("https:///x", nil)
	if err == nil {
		t.Fatal("expected error for missing host")
	}
}

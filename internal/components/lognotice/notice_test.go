package lognotice

import (
	"strings"
	"testing"
	"time"
)

type fakeSender struct {
	sent []noticeContent
}

func (s *fakeSender) Send(_, _ string, msg noticeContent) error {
	s.sent = append(s.sent, msg)
	return nil
}

func TestRateLimiterAggregatesRepeatedMessagesInWindow(t *testing.T) {
	sender := &fakeSender{}
	limiter := newNoticeLimiter(time.Minute, func() time.Time {
		return time.Unix(100, 0)
	})
	base := noticeContent{msg: "db down", filename: "repo/service.go", line: 42}

	limiter.handle(sender, "svc", "url", base)
	limiter.handle(sender, "svc", "url", base)
	limiter.handle(sender, "svc", "url", base)

	if len(sender.sent) != 1 {
		t.Fatalf("sent count in window = %d, want 1", len(sender.sent))
	}

	limiter.flushExpired(sender, "svc", "url", time.Unix(161, 0))

	if len(sender.sent) != 2 {
		t.Fatalf("sent count after flush = %d, want 2", len(sender.sent))
	}
	if !strings.Contains(sender.sent[1].msg, "重复2次") {
		t.Fatalf("aggregate message = %q, want repeated count", sender.sent[1].msg)
	}
	if sender.sent[1].filename != base.filename || sender.sent[1].line != base.line {
		t.Fatalf("aggregate location = %s:%d, want %s:%d", sender.sent[1].filename, sender.sent[1].line, base.filename, base.line)
	}
}

func TestRateLimiterDoesNotSuppressDifferentMessages(t *testing.T) {
	sender := &fakeSender{}
	limiter := newNoticeLimiter(time.Minute, func() time.Time {
		return time.Unix(100, 0)
	})

	limiter.handle(sender, "svc", "url", noticeContent{msg: "db down", filename: "repo/service.go", line: 42})
	limiter.handle(sender, "svc", "url", noticeContent{msg: "cache down", filename: "repo/service.go", line: 42})

	if len(sender.sent) != 2 {
		t.Fatalf("sent count = %d, want 2", len(sender.sent))
	}
}

func TestNoticeLimiterUsesConfigWindowAndCanBeDisabled(t *testing.T) {
	conf := &config{
		LimitWindowSeconds: 2,
		LimitMaxKeys:       10,
	}
	limiter := newNoticeLimiterFromConfig(conf)
	if limiter.window != 2*time.Second {
		t.Fatalf("limit window = %s, want 2s", limiter.window)
	}

	conf.IsLimitDisabled = true
	if limiter := newNoticeLimiterFromConfig(conf); limiter != nil {
		t.Fatal("expected nil limiter when disabled")
	}
}

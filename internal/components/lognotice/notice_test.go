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
	// 首条立即发送计为 1 次,3 次 handle 即 repeats=3,口径与真实发生次数对齐
	if !strings.Contains(sender.sent[1].msg, "重复3次") {
		t.Fatalf("aggregate message = %q, want repeated count", sender.sent[1].msg)
	}
	if sender.sent[1].filename != base.filename || sender.sent[1].line != base.line {
		t.Fatalf("aggregate location = %s:%d, want %s:%d", sender.sent[1].filename, sender.sent[1].line, base.filename, base.line)
	}
}

func TestRateLimiterSingleOccurrenceHasNoSummary(t *testing.T) {
	// 窗口内仅发生一次:flush 时不应补发"重复1次"的聚合摘要
	sender := &fakeSender{}
	limiter := newNoticeLimiter(time.Minute, func() time.Time {
		return time.Unix(100, 0)
	})
	base := noticeContent{msg: "db down", filename: "repo/service.go", line: 42}

	limiter.handle(sender, "svc", "url", base)
	limiter.flushExpired(sender, "svc", "url", time.Unix(161, 0))

	if len(sender.sent) != 1 {
		t.Fatalf("sent count = %d, want 1 (no summary for single occurrence)", len(sender.sent))
	}
}

func TestRateLimiterHandleOnExpiredKeyFlushesAndResends(t *testing.T) {
	// 同一 key 跨窗口:第二个窗口的首条 handle 命中已过期 entry,
	// 应先聚合上报上一窗口(重复3次),再把本条作为新通知立即发送
	sender := &fakeSender{}
	tick := time.Unix(100, 0)
	limiter := newNoticeLimiter(time.Minute, func() time.Time { return tick })
	base := noticeContent{msg: "db down", filename: "repo/service.go", line: 42}

	limiter.handle(sender, "svc", "url", base) // 立即发送(1)
	limiter.handle(sender, "svc", "url", base) // repeats=2
	limiter.handle(sender, "svc", "url", base) // repeats=3

	// 推进时钟越过窗口,再次 handle 同一 key
	tick = tick.Add(2 * time.Minute)
	limiter.handle(sender, "svc", "url", base)

	// 期望:首条立即发送 + 聚合摘要 + 新窗口首条立即发送 = 3 条
	if len(sender.sent) != 3 {
		t.Fatalf("sent count = %d, want 3", len(sender.sent))
	}
	if !strings.Contains(sender.sent[1].msg, "重复3次") {
		t.Fatalf("summary = %q, want 重复3次", sender.sent[1].msg)
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

func TestLimiterEvictsOldestOnOverflow(t *testing.T) {
	// maxKeys=2:先 put k1/k2,等 1ms 后 put k3,验证 k1(最旧)被淘汰,k2/k3 仍在 pending
	sender := &fakeSender{}

	tick := time.Unix(1000, 0)
	limiter := newNoticeLimiter(time.Minute, func() time.Time { return tick })
	limiter.maxKeys = 2

	k1 := noticeContent{msg: "k1", filename: "f.go", line: 1}
	k2 := noticeContent{msg: "k2", filename: "f.go", line: 2}
	k3 := noticeContent{msg: "k3", filename: "f.go", line: 3}

	// put k1
	limiter.handle(sender, "svc", "url", k1)
	// 推进时钟 1ms,让 k2 的 addedAt 晚于 k1
	tick = tick.Add(time.Millisecond)
	// put k2
	limiter.handle(sender, "svc", "url", k2)

	if len(limiter.pending) != 2 {
		t.Fatalf("after k1+k2: pending len = %d, want 2", len(limiter.pending))
	}

	// 推进时钟 1ms,让 k3 的 addedAt 更晚
	tick = tick.Add(time.Millisecond)
	// put k3,此时 map 已满,应淘汰最旧的 k1
	limiter.handle(sender, "svc", "url", k3)

	if len(limiter.pending) != 2 {
		t.Fatalf("after k3: pending len = %d, want 2", len(limiter.pending))
	}
	if _, ok := limiter.pending[k1.key()]; ok {
		t.Fatal("k1 应被 LRU 淘汰,但仍在 pending 中")
	}
	if _, ok := limiter.pending[k2.key()]; !ok {
		t.Fatal("k2 应保留在 pending 中")
	}
	if _, ok := limiter.pending[k3.key()]; !ok {
		t.Fatal("k3 应写入 pending 中")
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

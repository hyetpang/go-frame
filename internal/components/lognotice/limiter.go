package lognotice

import (
	"fmt"
	"time"
)

type noticeLimiter struct {
	window  time.Duration
	now     func() time.Time
	maxKeys int
	pending map[string]*limitedNotice
}

type limitedNotice struct {
	content   noticeContent
	expiresAt time.Time
	addedAt   time.Time // 用于 LRU 淘汰:记录 entry 写入 pending map 的时间
	repeats   int
}

func newNoticeLimiter(window time.Duration, now func() time.Time) *noticeLimiter {
	return &noticeLimiter{
		window:  window,
		now:     now,
		maxKeys: 1024,
		pending: make(map[string]*limitedNotice),
	}
}

func newNoticeLimiterFromConfig(conf *config) *noticeLimiter {
	if conf.IsLimitDisabled {
		return nil
	}
	window := noticeLimitWindow
	if conf.LimitWindowSeconds > 0 {
		window = time.Duration(conf.LimitWindowSeconds) * time.Second
	}
	limiter := newNoticeLimiter(window, time.Now)
	if conf.LimitMaxKeys > 0 {
		limiter.maxKeys = conf.LimitMaxKeys
	}
	return limiter
}

func (limiter *noticeLimiter) handle(sender sender, serviceName, url string, msg noticeContent) {
	if limiter == nil {
		_ = sender.Send(serviceName, url, msg)
		return
	}
	current := limiter.now()
	limiter.flushExpired(sender, serviceName, url, current)

	key := msg.key()
	if item, ok := limiter.pending[key]; ok && current.Before(item.expiresAt) {
		item.repeats++
		return
	}

	_ = sender.Send(serviceName, url, msg)
	if len(limiter.pending) >= limiter.maxKeys {
		// pending map 已满,淘汰 addedAt 最早的 entry,确保新 key 能被限流聚合
		var oldestKey string
		var oldestTime time.Time
		for k, v := range limiter.pending {
			if oldestKey == "" || v.addedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.addedAt
			}
		}
		delete(limiter.pending, oldestKey)
		if lognoticeEvicted != nil {
			lognoticeEvicted.Inc()
		}
	}
	limiter.pending[key] = &limitedNotice{
		content:   msg,
		expiresAt: current.Add(limiter.window),
		addedAt:   current,
	}
}

func (limiter *noticeLimiter) flushExpired(sender sender, serviceName, url string, current time.Time) {
	if limiter == nil {
		return
	}
	for key, item := range limiter.pending {
		if current.Before(item.expiresAt) {
			continue
		}
		limiter.sendSummary(sender, serviceName, url, item)
		delete(limiter.pending, key)
	}
}

func (limiter *noticeLimiter) flushAll(sender sender, serviceName, url string) {
	if limiter == nil {
		return
	}
	for key, item := range limiter.pending {
		limiter.sendSummary(sender, serviceName, url, item)
		delete(limiter.pending, key)
	}
}

func (limiter *noticeLimiter) sendSummary(sender sender, serviceName, url string, item *limitedNotice) {
	if item.repeats <= 0 {
		return
	}
	summary := item.content
	summary.msg = fmt.Sprintf("%s，%s内重复%d次", item.content.msg, limiter.window, item.repeats)
	_ = sender.Send(serviceName, url, summary)
}

func (content noticeContent) key() string {
	return fmt.Sprintf("%s:%d:%s", content.filename, content.line, content.msg)
}

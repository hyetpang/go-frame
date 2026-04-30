package lognotice

import (
	"fmt"
	"time"
)

type noticeLimiter struct {
	window  time.Duration
	now     func() time.Time
	pending map[string]*limitedNotice
}

type limitedNotice struct {
	content   noticeContent
	expiresAt time.Time
	repeats   int
}

func newNoticeLimiter(window time.Duration, now func() time.Time) *noticeLimiter {
	return &noticeLimiter{
		window:  window,
		now:     now,
		pending: make(map[string]*limitedNotice),
	}
}

func (limiter *noticeLimiter) handle(sender sender, serviceName, url string, msg noticeContent) {
	current := limiter.now()
	limiter.flushExpired(sender, serviceName, url, current)

	key := msg.key()
	if item, ok := limiter.pending[key]; ok && current.Before(item.expiresAt) {
		item.repeats++
		return
	}

	_ = sender.Send(serviceName, url, msg)
	limiter.pending[key] = &limitedNotice{
		content:   msg,
		expiresAt: current.Add(limiter.window),
	}
}

func (limiter *noticeLimiter) flushExpired(sender sender, serviceName, url string, current time.Time) {
	for key, item := range limiter.pending {
		if current.Before(item.expiresAt) {
			continue
		}
		limiter.sendSummary(sender, serviceName, url, item)
		delete(limiter.pending, key)
	}
}

func (limiter *noticeLimiter) flushAll(sender sender, serviceName, url string) {
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

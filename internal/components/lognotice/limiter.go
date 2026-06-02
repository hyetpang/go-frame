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
	// 过期 flush 不再在每条消息处理时全表遍历 + 同步 HTTP 发送(5s 超时会阻塞
	// watch 处理循环导致 noticeCh 写满丢弃),改为依赖 watchOnce 的 flushTicker 周期触发。
	// 这里仅对命中的 key 做就地过期判断:已过期则先聚合上报再视为一条新通知重新计数。
	key := msg.key()
	if item, ok := limiter.pending[key]; ok {
		if current.Before(item.expiresAt) {
			item.repeats++
			return
		}
		// 命中 key 已过期:先把窗口内累计上报,再删除让下方按新通知重新立即发送
		limiter.sendSummary(sender, serviceName, url, item)
		delete(limiter.pending, key)
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
		repeats:   1, // 首条立即发送即计为 1 次,后续命中递增,summary 口径与真实发生次数对齐
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
	// repeats 含首条立即发送那次,故 <=1 表示窗口内仅发生一次,无需补发聚合摘要。
	if item.repeats <= 1 {
		return
	}
	summary := item.content
	summary.msg = fmt.Sprintf("%s，%s内重复%d次", item.content.msg, limiter.window, item.repeats)
	_ = sender.Send(serviceName, url, summary)
}

func (content noticeContent) key() string {
	return fmt.Sprintf("%s:%d:%s", content.filename, content.line, content.msg)
}

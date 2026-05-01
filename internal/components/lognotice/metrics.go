package lognotice

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// 指标使用 sync.Once 包装,避免重复注册触发 promauto panic(包级 init 在测试反复 build 时也安全)。
var (
	metricsOnce      sync.Once
	noticeDropped    prometheus.Counter
	noticeRestart    prometheus.Counter
	noticeAliveGauge prometheus.Gauge
)

func initMetrics() {
	metricsOnce.Do(func() {
		noticeDropped = promauto.NewCounter(prometheus.CounterOpts{
			Name: "lognotice_dropped_total",
			Help: "通知通道已满而被丢弃的消息总数",
		})
		noticeRestart = promauto.NewCounter(prometheus.CounterOpts{
			Name: "lognotice_watch_restart_total",
			Help: "Watch goroutine 因 panic 自我重启的次数",
		})
		noticeAliveGauge = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "lognotice_alive",
			Help: "Watch goroutine 心跳标记,1 表示存活",
		})
	})
}

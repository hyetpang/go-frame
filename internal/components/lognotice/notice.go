package lognotice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hyetpang/go-frame/pkgs/logs"
	lognoticepkg "github.com/hyetpang/go-frame/pkgs/lognotice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	noticeChanBuffer    = 128
	noticeLimitWindow   = time.Minute
	noticeAliveInterval = 30 * time.Second
)

type notice struct {
	conf      *config
	noticeCh  chan noticeContent
	done      chan struct{}
	closeOnce sync.Once
	limiter   *noticeLimiter
	sender
}

func newNotice(conf *config, lc fx.Lifecycle) (lognoticepkg.Notifier, error) {
	initMetrics()
	if err := validateWebhookURL(conf.Notice, conf.AllowedHosts); err != nil {
		return nil, fmt.Errorf("log_notice webhook 校验失败: %w", err)
	}
	// senderBase 复用同一个 http.Client 与 safeDialer:
	// 取代旧 gout.SetTimeout 全局副作用,并在拨号期复检 IP 防 DNS rebinding。
	base := newSenderBase(conf.AllowedHosts)
	var sender sender
	switch conf.NoticeType {
	case noticeTypeWecom:
		sender = newWecomNotice(base)
	case noticeTypeFeiShu:
		sender = newFeiShuNotice(base)
	case noticeTypeTelegram:
		sender = newTelegramSender(base, conf.ChatID)
	case noticeTypeEmail:
		return nil, errors.New("日志错误通知尚未支持邮件")
	default:
		return nil, fmt.Errorf("错误日志配置的通知的类型有误:%+v", conf)
	}
	n := &notice{
		conf:     conf,
		noticeCh: make(chan noticeContent, noticeChanBuffer),
		done:     make(chan struct{}),
		limiter:  newNoticeLimiterFromConfig(conf),
		sender:   sender,
	}
	go n.Watch()
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			// Close 仅关闭内部 channel 触发 watch goroutine 退出,语义瞬时,无需 ctx 兜底。
			n.Close()
			return nil
		},
	})
	return n, nil
}

// Notice 接收一条出错通知,filename/line 由调用方(pkgs/logs)给出,
// 因此这里不再使用 runtime.Caller 自行解析栈帧,可避免栈深耦合。
func (notice *notice) Notice(msg string, filename string, line int, fields ...zap.Field) {
	content := noticeContent{
		msg:      msg,
		filename: filename,
		line:     line,
	}
	select {
	case notice.noticeCh <- content:
	case <-notice.done:
	default:
		// 通道已满,丢弃本条通知避免阻塞调用方
		if noticeDropped != nil {
			noticeDropped.Inc()
		}
	}
}

func (notice *notice) Watch() {
	for {
		// Close 后 done 已关:即使 watchOnce 因 panic 提前 defer 返回(exit=false),
		// 这里也直接退出,避免误增 noticeRestart 计数与多跑一轮 ticker 创建/释放。
		select {
		case <-notice.done:
			if noticeAliveGauge != nil {
				noticeAliveGauge.Set(0)
			}
			return
		default:
		}
		exit := notice.watchOnce()
		if exit {
			if noticeAliveGauge != nil {
				noticeAliveGauge.Set(0)
			}
			return
		}
		if noticeRestart != nil {
			noticeRestart.Inc()
		}
		logs.ErrorWithoutNotice("Watch goroutine crashed, restarting...")
	}
}

func (notice *notice) watchOnce() (exit bool) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				logs.ErrorWithoutNotice("错误通知panic", zap.Error(err))
			} else {
				logs.ErrorWithoutNotice("错误通知panic", zap.Any("recover", r))
			}
		}
	}()
	logs.Info("开始watch出错消息...")
	if noticeAliveGauge != nil {
		noticeAliveGauge.Set(1)
	}
	// limiter == nil(IsLimitDisabled)时不创建 flushTicker,
	// 用 nil channel 让 select 永远不命中 flush 分支,避免 1min ticker 空转。
	var flushC <-chan time.Time
	if notice.limiter != nil {
		flushTicker := time.NewTicker(notice.limiter.window)
		defer flushTicker.Stop()
		flushC = flushTicker.C
	}
	aliveTicker := time.NewTicker(noticeAliveInterval)
	defer aliveTicker.Stop()
	for {
		select {
		case msg := <-notice.noticeCh:
			notice.limiter.handle(notice.sender, notice.conf.Name, notice.conf.Notice, msg)
		case now := <-flushC:
			notice.limiter.flushExpired(notice.sender, notice.conf.Name, notice.conf.Notice, now)
		case <-aliveTicker.C:
			if noticeAliveGauge != nil {
				noticeAliveGauge.Set(1)
			}
		case <-notice.done:
			// drain缓冲区中剩余的消息
			for {
				select {
				case msg := <-notice.noticeCh:
					notice.limiter.handle(notice.sender, notice.conf.Name, notice.conf.Notice, msg)
				default:
					notice.limiter.flushAll(notice.sender, notice.conf.Name, notice.conf.Notice)
					exit = true
					return
				}
			}
		}
	}
}

func (notice *notice) Close() {
	notice.closeOnce.Do(func() {
		logs.Info("结束watch出错消息...")
		close(notice.done)
	})
}

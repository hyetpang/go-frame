package lognotice

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/hyetpang/go-frame/pkgs/interfaces"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	noticeChanBuffer  = 128
	noticeLimitWindow = time.Minute
)

type notice struct {
	conf      *config
	noticeCh  chan noticeContent
	done      chan struct{}
	closeOnce sync.Once
	limiter   *noticeLimiter
	sender
}

func newNotice(conf *config, lc fx.Lifecycle) (interfaces.LogNoticeInterface, error) {
	var sender sender
	switch conf.NoticeType {
	case noticeTypeWecom:
		sender = newWecomNotice()
	case noticeTypeFeiShu:
		sender = newFeiShuNotice()
	case noticeTypeTelegram:
		sender = newTelegramSender(conf.ChatID)
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
	lc.Append(fx.StopHook(func() {
		n.Close()
	}))
	return n, nil
}

func (notice *notice) Notice(msg string, fields ...zap.Field) {
	_, filename, line, _ := runtime.Caller(3)
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
	}
}

func (notice *notice) Watch() {
	for {
		exit := notice.watchOnce()
		if exit {
			return
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
	flushInterval := noticeLimitWindow
	if notice.limiter != nil {
		flushInterval = notice.limiter.window
	}
	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()
	for {
		select {
		case msg := <-notice.noticeCh:
			notice.limiter.handle(notice.sender, notice.conf.Name, notice.conf.Notice, msg)
		case now := <-flushTicker.C:
			notice.limiter.flushExpired(notice.sender, notice.conf.Name, notice.conf.Notice, now)
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

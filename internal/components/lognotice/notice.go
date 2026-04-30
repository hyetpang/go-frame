package lognotice

import (
	"log"
	"runtime"
	"sync"

	"github.com/hyetpang/go-frame/pkgs/interfaces"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const noticeChanBuffer = 128

type notice struct {
	conf     *config
	noticeCh chan noticeContent
	done     chan struct{}
	closeOnce sync.Once
	sender
}

func newNotice(conf *config, lc fx.Lifecycle) interfaces.LogNoticeInterface {
	var sender sender
	switch conf.NoticeType {
	case noticeTypeWecom:
		sender = newWecomNotice()
	case noticeTypeFeiShu:
		sender = newFeiShuNotice()
	case noticeTypeTelegram:
		sender = newTelegramSender(conf.ChatID)
	case noticeTypeEmail:
		log.Fatal("日志错误通知尚未支持邮件")
	default:
		log.Fatalf("错误日志配置的通知的类型有误:%+v", conf)
	}
	n := &notice{
		conf:     conf,
		noticeCh: make(chan noticeContent, noticeChanBuffer),
		done:     make(chan struct{}),
		sender:   sender,
	}
	go n.Watch()
	lc.Append(fx.StopHook(func() {
		n.Close()
	}))
	return n
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
	for {
		select {
		case msg := <-notice.noticeCh:
			_ = notice.sender.Send(notice.conf.Name, notice.conf.Notice, msg)
		case <-notice.done:
			// drain缓冲区中剩余的消息
			for {
				select {
				case msg := <-notice.noticeCh:
					_ = notice.sender.Send(notice.conf.Name, notice.conf.Notice, msg)
				default:
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

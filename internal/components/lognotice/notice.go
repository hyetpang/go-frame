package lognotice

import (
	"log"
	"runtime"

	"github.com/hyetpang/go-frame/pkgs/interfaces"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type notice struct {
	conf     *config
	noticeCh chan noticeContent
	sender
}

func newNotice(conf *config, lc fx.Lifecycle) interfaces.LogNoticeInterface {
	var sender sender
	if conf.NoticeType == noticeTypeWecom {
		// 企业微信
		sender = newWecomNotice()
	} else if conf.NoticeType == noticeTypeEmail {
		panic("日志错误通知尚未支持邮件")
		// noticeInterface = &emailNotice{
		// 	conf: conf,
		// }
	} else if conf.NoticeType == noticeTypeFeiShu {
		// 飞书
		sender = newFeiShuNotice()
	} else {
		log.Fatal("错误日志配置的通知的类型有误", conf)
	}
	n := &notice{
		conf:     conf,
		noticeCh: make(chan noticeContent, 1),
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
	notice.noticeCh <- noticeContent{
		msg:      msg,
		filename: filename,
		line:     line,
	}
}

func (notice *notice) Watch() {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if ok {
				logs.ErrorWithoutNotice("错误通知panic", zap.Error(err))
			} else {
				logs.ErrorWithoutNotice("错误通知panic")
			}
		}
	}()
	logs.Info("开始watch出错消息...")
	for noticeMsg := range notice.noticeCh {
		notice.sender.Send(notice.conf.Name, notice.conf.Notice, noticeMsg)
	}
}

func (notice *notice) Close() {
	logs.Info("结束watch出错消息...")
	close(notice.noticeCh)
}

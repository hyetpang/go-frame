package lognotice

import (
	"log"

	"github.com/HyetPang/go-frame/pkgs/interfaces"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// 错误日志通知返回具体实例
func New() interfaces.LogNoticeInterface {
	conf := new(config)
	err := viper.UnmarshalKey("log_notice", &conf)
	if err != nil {
		log.Fatal("log_notice配置Unmarshal到对象出错", zap.Error(err))
	}
	if conf.NoticeType == noticeTypeWecom {
		wecomNotice := &wecomNotice{
			conf:     conf,
			noticeCh: make(chan noticeContent, 1),
		}
		go wecomNotice.noticeMsg()
		return wecomNotice
	} else if conf.NoticeType == noticeTypeEmail {
		return &emailNotice{
			conf: conf,
		}
	}
	log.Fatal("错误日志配置的通知的类型有误", conf)
	return nil
}

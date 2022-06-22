package lognotice

import (
	"go.uber.org/zap"
)

// 企业微信
type emailNotice struct {
	conf *config
}

func (emailNotice *emailNotice) Notice(msg string, fields ...zap.Field) {
	// TODO 待实现
	// err := gomail.NewDialer(conf.Host, conf.Port, conf.User, conf.Pass).DialAndSend(m)
}

package lognotice

import frameconfig "github.com/hyetpang/go-frame/internal/config"

type config = frameconfig.LogNotice

const (
	noticeTypeWecom    = iota + 1 // 企业微信
	noticeTypeEmail               // 邮件,尚未实现
	noticeTypeFeiShu              // 飞书
	noticeTypeTelegram            // telegram
)

package lognotice

type config struct {
	Notice     string `mapstructure:"notice" validate:"required"`                    // 通知地址，可能是一个url也可能是一个邮箱地址
	Name       string `mapstructure:"name" validate:"required"`                      // 通知名字,通常是服务名字
	ChatID     string `mapstructure:"chat_id" validate:"required_if=NoticeType 4"`   // 使用telegram需要配置chat_id
	NoticeType int    `mapstructure:"notice_type" validate:"required,oneof=1 2 3 4"` // 通知类型,1=>企业微信,2=>邮件,3=>飞书，4=>telegram
}

const (
	noticeTypeWecom    = iota + 1 // 企业微信
	noticeTypeEmail               // 邮件,尚未实现
	noticeTypeFeiShu              // 飞书
	noticeTypeTelegram            // telegram
)

package lognotice

type config struct {
	WecomURL   string `mapstructure:"wecom_url" validate:"required_if=NoticeType 1"` // 企业微信通知url
	Name       string `mapstructure:"name" validate:"required"`                      // 通知名字,通常是服务名字
	NoticeType int    `mapstructure:"notice_type" validate:"required,oneof=1 2"`     // 通知类型,1=>企业微信,2=>邮件
	// Email      *Email `mapstructure:"email" validate:"required_if=NoticeType 2"`     // 邮件通知
}

type Email struct {
	Receiver string `mapstructure:"receiver" validate:"required"` // 邮件接收人
	Sender   string `mapstructure:"sender" validate:"required"`   // 邮件发送人
	Host     string `mapstructure:"host" validate:"required"`     // 邮件主机
	Pwd      string `mapstructure:"pwd" validate:"required"`      // 发件人密码
	Port     int    `mapstructure:"port" validate:"required"`     // 邮件端口
}

const (
	noticeTypeWecom = iota + 1
	noticeTypeEmail
)

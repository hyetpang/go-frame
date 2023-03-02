package lognotice

type config struct {
	Notice     string `mapstructure:"notice" validate:"required"`                  // 通知地址，可能是一个url也可能是一个邮箱地址
	Name       string `mapstructure:"name" validate:"required"`                    // 通知名字,通常是服务名字
	NoticeType int    `mapstructure:"notice_type" validate:"required,oneof=1 2 3"` // 通知类型,1=>企业微信,2=>邮件,3=>飞书
}

type Email struct {
	Receiver string `mapstructure:"receiver" validate:"required"` // 邮件接收人
	Sender   string `mapstructure:"sender" validate:"required"`   // 邮件发送人
	Host     string `mapstructure:"host" validate:"required"`     // 邮件主机
	Pwd      string `mapstructure:"pwd" validate:"required"`      // 发件人密码
	Port     int    `mapstructure:"port" validate:"required"`     // 邮件端口
}

const (
	noticeTypeWecom  = iota + 1 // 企业微信
	noticeTypeEmail             // 邮件,尚未实现
	noticeTypeFeiShu            // 飞书
)

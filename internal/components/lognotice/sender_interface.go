package lognotice

type sender interface {
	Send(serviceName, url string, msg noticeContent) error
}

// 通知消息
type noticeContent struct {
	msg, filename string
	line          int
}

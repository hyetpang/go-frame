package lognotice

// Inject 注入一个 Notifier 实现,业务侧通过该函数把具体通知器接入到全局 Notice 调用上。
func Inject(notifier Notifier) {
	logNotice = notifier
}

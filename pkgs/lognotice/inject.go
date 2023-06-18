package lognotice

import "github.com/hyetpang/go-frame/pkgs/interfaces"

// 注入一个实现
func Inject(logNoticeTemp interfaces.LogNoticeInterface) {
	logNotice = logNoticeTemp
}

package lognotice

import "github.com/HyetPang/go-frame/pkgs/interfaces"

// 注入一个实现
func Inject(logNoticeTemp interfaces.LogNoticeInterface) {
	logNotice = logNoticeTemp
}

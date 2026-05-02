// Package lognotice 提供错误日志通知的对外注入入口与默认实现。
package lognotice

import (
	"sync/atomic"

	"go.uber.org/zap"
)

// Notifier 定义错误日志通知接口,实现该接口的组件可被注入到日志通知链路中。
type Notifier interface {
	// Notice 接收一条出错消息,filename 与 line 表示触发点的源代码位置。
	Notice(msg string, filename string, line int, fields ...zap.Field)
}

// logNotice 用 atomic.Pointer 保护:Inject 在启动期写入,Notice 在请求路径并发读,
// 与 pkgs/logs.noticeHook 保持一致的并发安全契约。默认指向 noop 实现避免未注入时 panic。
var logNotice atomic.Pointer[Notifier]

func init() {
	var n Notifier = defaultLogNotice{}
	logNotice.Store(&n)
}

type defaultLogNotice struct{}

// Notice 默认实现不做任何事情。
func (defaultLogNotice) Notice(msg string, filename string, line int, fields ...zap.Field) {}

// Notice 触发一次错误日志通知,会转发到当前注入的 Notifier。
func Notice(msg string, filename string, line int, fields ...zap.Field) {
	if p := logNotice.Load(); p != nil {
		(*p).Notice(msg, filename, line, fields...)
	}
}

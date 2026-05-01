package common

import "sync/atomic"

// devFlag 标记当前是否为 Dev 运行模式。
// 启动期由 SetDev 写入,运行期由 IsDev 并发读取,使用 atomic.Bool 避免
// race detector 报警与可见性问题。export 函数而非裸变量,防止外部直接读写。
var devFlag atomic.Bool

// SetDev 设置当前是否为 Dev 模式,通常仅在应用启动期调用一次。
func SetDev(dev bool) {
	devFlag.Store(dev)
}

// IsDev 返回当前是否处于 Dev 模式。
func IsDev() bool {
	return devFlag.Load()
}

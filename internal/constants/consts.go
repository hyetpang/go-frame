package constants

import "time"

const (
	// StartupCtxTimeout 用于组件启动期单次健康探测(redis Ping 等)的 ctx 超时,
	// 与请求路径 timeout 区分开,避免与业务 timeout 命名冲突。
	StartupCtxTimeout = time.Second * 5
)

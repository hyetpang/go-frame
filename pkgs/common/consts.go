/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-04-30 19:47:21
 * @FilePath: \go-frame\pkgs\common\consts.go
 */
package common

import "time"

const (
	DevMode  = "dev"
	TestMode = "test"
)

const (
	RedisTimeout = time.Second * 5
)

const (
	False = iota
	True
)

// http调用超时
const (
	HttpCallTimeOut = time.Second * 5
)

// 默认的数据库连接名字
const (
	DefaultDb = "default"
)

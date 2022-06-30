/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-04-30 16:38:26
 * @FilePath: \go-frame\internal\adapter\logadapter\gin_recovety_zap_adapter.go
 */
package log

import (
	"github.com/HyetPang/go-frame/pkgs/lognotice"
	"go.uber.org/zap"
)

type ginRecoveryZapLog struct {
	*zap.Logger
}

func NewGinRecoveryZapLog() *ginRecoveryZapLog {
	return &ginRecoveryZapLog{zap.L()}
}

func (ginRecoveryZapLog *ginRecoveryZapLog) Error(msg string, fields ...zap.Field) {
	lognotice.Notice("panic=====>"+msg, fields...)
	ginRecoveryZapLog.Logger.Error(msg, fields...)
}

package log

import (
	"github.com/hyetpang/go-frame/pkgs/lognotice"
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

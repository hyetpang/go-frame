package logadapter

import "go.uber.org/zap"

type ginRecoveryZapLog struct {
	*zap.Logger
}

func NewGinRecoveryZapLog() *ginRecoveryZapLog {
	return &ginRecoveryZapLog{zap.L()}
}

func (ginRecoveryZapLog *ginRecoveryZapLog) Write(p []byte) (n int, err error) {
	ginRecoveryZapLog.Logger.Error("panic", zap.String("=====", string(p)))
	return 0, nil
}

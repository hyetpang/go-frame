/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-04-30 16:38:26
 * @FilePath: \go-frame\internal\adapter\logadapter\gin_recovety_zap_adapter.go
 */
package log

import (
	"github.com/HyetPang/go-frame/pkgs/common"
	"go.uber.org/zap"
)

type ginRecoveryZapLog struct {
	*zap.Logger
}

func NewGinRecoveryZapLog() *ginRecoveryZapLog {
	return &ginRecoveryZapLog{zap.L()}
}

func (ginRecoveryZapLog *ginRecoveryZapLog) Write(p []byte) (n int, err error) {
	ginRecoveryZapLog.Logger.Sugar().Error(common.BytesString(p))
	ginRecoveryZapLog.Logger.Error("=====>panic", zap.String("stack", common.BytesString(p)))
	return 0, nil
}

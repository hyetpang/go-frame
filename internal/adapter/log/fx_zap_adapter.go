/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-04-30 16:38:21
 * @FilePath: \go-frame\internal\adapter\logadapter\fx_zap_adapter.go
 */
package log

import (
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func NewFxZap(log *zap.Logger) fxevent.Logger {
	return &fxevent.ZapLogger{Logger: log}
}

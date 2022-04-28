package logadapter

import (
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func NewFxZap() fxevent.Logger {
	return &fxevent.ZapLogger{Logger: zap.L()}
}

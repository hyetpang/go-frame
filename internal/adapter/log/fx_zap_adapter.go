package log

import (
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func NewFxZap(log *zap.Logger) fxevent.Logger {
	return &fxevent.ZapLogger{Logger: log}
}

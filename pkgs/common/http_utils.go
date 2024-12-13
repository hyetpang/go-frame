package common

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func LoggerReqCtx(ctx *gin.Context) *logs.Log {
	return logs.With(zap.String("method", ctx.Request.Method), zap.String("url", ctx.Request.RequestURI), zap.Object("header", ObjectMarshaler(func(oe zapcore.ObjectEncoder) {
		for k, v := range ctx.Request.Header {
			oe.AddString(k, strings.Join(v, ","))
		}
	})))
}

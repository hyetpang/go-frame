package common

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var sensitiveHeaders = map[string]struct{}{
	"Authorization":       {},
	"Cookie":              {},
	"Set-Cookie":          {},
	"X-Api-Key":           {},
	"Proxy-Authorization": {},
}

func LoggerReqCtx(ctx *gin.Context) *logs.Log {
	return logs.With(zap.String("method", ctx.Request.Method), zap.String("url", ctx.Request.RequestURI), zap.Object("header", ObjectMarshaler(func(oe zapcore.ObjectEncoder) {
		for k, v := range ctx.Request.Header {
			if _, ok := sensitiveHeaders[k]; ok {
				oe.AddString(k, "***")
				continue
			}
			oe.AddString(k, strings.Join(v, ","))
		}
	})))
}

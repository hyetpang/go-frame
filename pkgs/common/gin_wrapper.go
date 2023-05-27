package common

import (
	"net/http"

	"github.com/HyetPang/go-frame/pkgs/base"
	"github.com/gin-gonic/gin"
)

type ginContextWrapper struct {
	ctx *gin.Context
}

func Wrap(ctx *gin.Context) *ginContextWrapper {
	return &ginContextWrapper{ctx: ctx}
}

func (gcw *ginContextWrapper) Success(data any) {
	gcw.ctx.PureJSON(http.StatusOK, gin.H{
		"msg":  "success",
		"code": 0,
		"data": data,
	})
}

func (gcw *ginContextWrapper) Fail(err base.CodeErrI, data ...any) {
	gcw.ctx.PureJSON(http.StatusOK, gin.H{
		"msg":  err.GetMsg(),
		"code": err.GetCode(),
	})
}

func (gcw *ginContextWrapper) AbortFail(err base.CodeErrI, data ...any) {
	gcw.ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
		"msg":  err.GetMsg(),
		"code": err.GetCode(),
	})
}

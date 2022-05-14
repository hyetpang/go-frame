package wrapper

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
	gcw.ctx.JSON(http.StatusOK, gin.H{
		"msg":  "success",
		"code": 0,
		"data": data,
	})
}

func (gcw *ginContextWrapper) UnwrapSuccess(data any) {
	gcw.ctx.JSON(http.StatusOK, data)
}

func (gcw *ginContextWrapper) Fail(err base.CodeErrI, data ...any) {
	gcw.ctx.JSON(http.StatusOK, gin.H{
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

func (gcw *ginContextWrapper) OldPaginationRsp(totalCount int64, data any) {
	gcw.ctx.JSON(http.StatusOK, gin.H{
		"msg":       "success",
		"code":      0,
		"pageTotal": totalCount,
		"data":      data,
		"version":   "v1.0.0",
	})
}

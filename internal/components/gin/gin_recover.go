package gin

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/lognotice"
	"go.uber.org/zap"
)

func recoveryWithZap(logger *zap.Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				_, recoverFile, recoverLine, _ := runtime.Caller(0)
				lognotice.Notice("[Recovery from panic]", recoverFile, recoverLine)
				// 复用 sensitiveHeaders 白名单脱敏,防止 Cookie/Authorization 等敏感头落日志或外发到 webhook
				httpRequest := common.SanitizeRequestForLog(c.Request)
				if brokenPipe {
					logger.Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", httpRequest),
					)
					// If the connection is dead, we can't write a status to it.
					if e, ok := err.(error); ok {
						c.Error(e) // nolint: errcheck
					} else {
						c.Error(fmt.Errorf("%v", err)) // nolint: errcheck
					}
					c.Abort()
					return
				}
				if stack {
					logger.Error("[Recovery from panic]",
						zap.Time("time", time.Now()),
						zap.Any("error", err),
						zap.String("request", httpRequest),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					logger.Error("[Recovery from panic]",
						zap.Time("time", time.Now()),
						zap.Any("error", err),
						zap.String("request", httpRequest),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

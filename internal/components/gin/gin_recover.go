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
				panicFile, panicLine := panicCaller()
				lognotice.Notice("[Recovery from panic]", panicFile, panicLine)
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

// panicCaller 在 deferred recover 内被调用,定位 panic 真正的触发位置。
//
// 旧实现 runtime.Caller(0) 只会拿到 gin_recover.go 自己的行号,
// 通知文案里"代码行数"永远指向 recover 函数,失去定位价值。
//
// 这里走 runtime.Callers 自底向上扫栈,跳过 Go runtime 帧(panic.go 等)
// 与本文件的 recover wrapper,首个用户帧即 panic 触发点。
func panicCaller() (string, int) {
	const maxFrames = 32
	var pcs [maxFrames]uintptr
	n := runtime.Callers(2, pcs[:])
	if n == 0 {
		return "", 0
	}
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		// 跳过 Go runtime 自身帧(runtime.gopanic / runtime.panicmem / 反射调用桥等)
		// 与本文件自身,定位到第一个真正的用户调用帧。
		if !strings.Contains(frame.File, "/runtime/") &&
			!strings.HasSuffix(frame.File, "gin_recover.go") {
			return frame.File, frame.Line
		}
		if !more {
			return frame.File, frame.Line
		}
	}
}

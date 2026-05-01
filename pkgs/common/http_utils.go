package common

import (
	"fmt"
	"net/http"
	"sort"
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

// IsSensitiveHeader 判断 header 名是否属于敏感字段(如 Authorization、Cookie 等)。
// 输入应已是 http.CanonicalHeaderKey 处理过的形式。
func IsSensitiveHeader(name string) bool {
	_, ok := sensitiveHeaders[name]
	return ok
}

// SanitizeRequestForLog 返回一个用于日志记录的请求文本表示,会把敏感 header 的值替换为 ***。
// 与 httputil.DumpRequest 相比不会泄露 Cookie/Authorization 等敏感字段。
func SanitizeRequestForLog(r *http.Request) string {
	if r == nil {
		return ""
	}
	var buf strings.Builder
	reqURI := r.RequestURI
	if reqURI == "" && r.URL != nil {
		reqURI = r.URL.RequestURI()
	}
	fmt.Fprintf(&buf, "%s %s %s\r\n", r.Method, reqURI, r.Proto)
	if r.Host != "" {
		fmt.Fprintf(&buf, "Host: %s\r\n", r.Host)
	}
	keys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if IsSensitiveHeader(k) {
			fmt.Fprintf(&buf, "%s: ***\r\n", k)
			continue
		}
		for _, v := range r.Header[k] {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
		}
	}
	return buf.String()
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

package gin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	frameconfig "github.com/hyetpang/go-frame/internal/config"
	"github.com/hyetpang/go-frame/pkgs/base"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/penglongli/gin-metrics/ginmetrics"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// httpReadyTimeout 启动期 self-check 的总超时,避免新检查拖慢 fx 启动
	httpReadyTimeout = 200 * time.Millisecond
	// httpMaxHeaderBytes 限制 HTTP 请求 header 总大小,避免恶意大 header 攻击
	httpMaxHeaderBytes = 1 << 20 // 1 MiB
	// httpMaxMultipartMemory gin 解析 multipart 表单时驻留内存上限,超出转写临时文件
	httpMaxMultipartMemory = 8 << 20 // 8 MiB
	// httpDefaultMaxBodyBytes body 限流中间件默认上限,可通过配置覆盖
	httpDefaultMaxBodyBytes int64 = 10 << 20 // 10 MiB
)

func New(zapLog *zap.Logger, lc fx.Lifecycle, conf *config, tracingConf *frameconfig.Tracing) (gin.IRouter, error) {
	router, conf, err := newGin(zapLog, conf, tracingConf)
	if err != nil {
		return nil, err
	}
	lis, err := net.Listen("tcp4", conf.Addr)
	if err != nil {
		return nil, fmt.Errorf("http服务地址监听出错 %s: %w", conf.Addr, err)
	}

	srv := &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    httpMaxHeaderBytes,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return startHTTPServer(ctx, srv, lis, router, conf.IsProd)
		},
		OnStop: func(ctx context.Context) error {
			if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			logs.Info("http服务器已关闭...")
			return nil
		},
	})
	return router, nil
}

func startHTTPServer(ctx context.Context, srv *http.Server, lis net.Listener, router *gin.Engine, isProd bool) error {
	errC := make(chan error, 1)
	go func() {
		if err := srv.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errC <- err
		}
	}()

	// 用一个短超时 + 主动 self-check 替代 select 的 default 分支,
	// 既避免固定 1s 等待,又能在启动失败时快速感知。
	checkCtx, cancel := context.WithTimeout(ctx, httpReadyTimeout)
	defer cancel()
	if err := selfCheckHTTPReady(checkCtx, lis.Addr().String(), errC); err != nil {
		return err
	}

	if router != nil && !isProd {
		for _, r := range router.Routes() {
			logs.Info("注册的路由=>", zap.String("method", r.Method), zap.String("url", r.Path), zap.String("handler", r.Handler))
		}
	}
	logs.Info("HTTP服务器启动成功", zap.String("监听地址", lis.Addr().String()))
	return nil
}

// selfCheckHTTPReady 主动 GET /health_check 验证 HTTP 服务真的能接受请求。
// errC 中的 Serve 错误优先返回,避免误报启动成功。
func selfCheckHTTPReady(ctx context.Context, addr string, errC <-chan error) error {
	select {
	case err := <-errC:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	url := fmt.Sprintf("http://%s/health_check", addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("构造http健康检查请求失败: %w", err)
	}
	client := &http.Client{Timeout: httpReadyTimeout}
	resp, err := client.Do(req)
	if err != nil {
		// self-check 失败时再确认 Serve 是否已经直接挂掉
		select {
		case e := <-errC:
			if e != nil {
				return e
			}
		default:
		}
		return fmt.Errorf("self-check http server %s 失败: %w", addr, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http健康检查状态码异常: %d", resp.StatusCode)
	}
	return nil
}

func newGin(zapLog *zap.Logger, conf *config, tracingConf *frameconfig.Tracing) (*gin.Engine, *config, error) {
	if err := common.Validate(conf); err != nil {
		return nil, nil, fmt.Errorf("http配置验证不通过: %w", err)
	}
	if len(conf.Addr) < 1 {
		return nil, nil, errors.New("http配置字段addr没有配置值")
	}
	if conf.IsProd {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.MaxMultipartMemory = httpMaxMultipartMemory
	maxBodyBytes := conf.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = httpDefaultMaxBodyBytes
	}
	router.Use(bodyLimitMiddleware(maxBodyBytes))
	router.Use(ginzap.Ginzap(zapLog, time.RFC3339Nano, false))
	router.Use(recoveryWithZap(zapLog, true))
	if tracingConf != nil && tracingConf.Enable {
		router.Use(otelgin.Middleware(tracingConf.ServiceName))
	}
	if conf.IsMetrics {
		m := ginmetrics.GetMonitor()
		m.SetMetricPath(conf.MetricsPath)
		m.SetSlowTime(1) // 超过1秒的响应都是慢请求
		m.SetDuration([]float64{0.1, 0.3, 1.2, 5, 10})
		m.Use(router)
	}
	router.NoRoute(noRoute)
	router.NoMethod(noMethodHandler)
	// 健康检查
	router.GET("/health_check", func(ctx *gin.Context) {
		common.Wrap(ctx).Success("ok")
	})
	// 文档
	if conf.IsDoc {
		router.GET(conf.DocPath, ginSwagger.WrapHandler(swaggerfiles.Handler))
	}
	if conf.IsPprof {
		if err := validatePprofCredentials(conf.PprofUsername, conf.PprofPassword); err != nil {
			return nil, nil, err
		}
		pprofRouter := router.Group("", gin.BasicAuth(gin.Accounts{
			conf.PprofUsername: conf.PprofPassword,
		}))
		if len(conf.PprofPrefix) > 0 {
			pprof.Register(pprofRouter, conf.PprofPrefix)
		} else {
			pprof.Register(pprofRouter)
		}
	}
	return router, conf, nil
}

const minPprofPasswordLen = 12

// pprofWeakCredentials 列出禁用的常见弱用户名/密码;比对时统一小写,避免 Admin/SECRET 等大小写绕过。
var pprofWeakCredentials = map[string]struct{}{
	"":          {},
	"admin":     {},
	"root":      {},
	"secret":    {},
	"password":  {},
	"change_me": {},
	"changeme":  {},
}

func isWeakPprofCredential(s string) bool {
	_, bad := pprofWeakCredentials[strings.ToLower(s)]
	return bad
}

// validatePprofCredentials 在启用 pprof 时强制要求强口令,避免示例配置直接上生产。
// 规则:用户名/密码不能命中弱口令列表(大小写不敏感);密码长度必须 >= 12。
func validatePprofCredentials(username, password string) error {
	if isWeakPprofCredential(username) {
		return fmt.Errorf("pprof_username 不能为空或使用常见弱口令 (admin/root 等),当前值: %q", username)
	}
	if isWeakPprofCredential(password) {
		return errors.New("pprof_password 不能为空或使用 admin/secret/CHANGE_ME 等示例值,请在配置中替换")
	}
	if len(password) < minPprofPasswordLen {
		return fmt.Errorf("pprof_password 长度必须 >= %d,当前长度 %d", minPprofPasswordLen, len(password))
	}
	return nil
}

func noRoute(ctx *gin.Context) {
	url := ctx.Request.Method + ":" + ctx.Request.URL.Path
	logs.Error("路由不存在", zap.String("method", ctx.Request.Method), zap.String("url", url), zap.String("ip", ctx.ClientIP()))
	common.Wrap(ctx).Fail(base.CodeErrNotFound)
}

func noMethodHandler(ctx *gin.Context) {
	url := ctx.Request.Method + ":" + ctx.Request.URL.Path
	logs.Error("请求方法不允许", zap.String("method", ctx.Request.Method), zap.String("url", url), zap.String("ip", ctx.ClientIP()))
	ctx.AbortWithStatus(http.StatusMethodNotAllowed)
}

// bodyLimitMiddleware 通过 http.MaxBytesReader 限制单个请求体最大字节数,
// 防止超大 body 占用内存或写盘空间。0 或负值表示不启用限制。
func bodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes > 0 && c.Request != nil && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

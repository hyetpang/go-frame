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
	"github.com/hyetpang/go-frame/pkgs/base"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/penglongli/gin-metrics/ginmetrics"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(zapLog *zap.Logger, lc fx.Lifecycle, conf *config) (gin.IRouter, error) {
	router, conf, err := newGin(zapLog, conf)
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

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	default:
		if router != nil && !isProd {
			for _, r := range router.Routes() {
				logs.Info("注册的路由=>", zap.String("method", r.Method), zap.String("url", r.Path), zap.String("handler", r.Handler))
			}
		}
		logs.Info("HTTP服务器启动成功", zap.String("监听地址", lis.Addr().String()))
		return nil
	}
}

func newGin(zapLog *zap.Logger, conf *config) (*gin.Engine, *config, error) {
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
	router.Use(ginzap.Ginzap(zapLog, time.RFC3339Nano, false))
	router.Use(recoveryWithZap(zapLog, true))
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

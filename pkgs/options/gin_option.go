package options

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/HyetPang/go-frame/pkgs/base"
	"github.com/HyetPang/go-frame/pkgs/common"
	"github.com/HyetPang/go-frame/pkgs/logadapter"
	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/HyetPang/go-frame/pkgs/wrapper"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func WithHttp() Option {
	return func(o *Options) {
		o.FxOptions = append(o.FxOptions, fx.Provide(newHttp))
	}
}

func newHttp(lc fx.Lifecycle) gin.IRouter {
	if viper.GetString("server.run_mode") != common.DevMode {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(ginzap.Ginzap(zap.L(), time.RFC3339, true))
	router.Use(gin.RecoveryWithWriter(logadapter.NewGinRecoveryZapLog()))
	router.NoMethod(noMethod)
	router.NoRoute(noRoute)
	// 健康检查
	router.GET("/health_check", func(ctx *gin.Context) {
		wrapper.Wrap(ctx).Success("ok")
	})

	lis, err := net.Listen("tcp4", viper.GetString("server.addr"))
	if err != nil {
		common.Panic(err)
	}

	srv := &http.Server{
		Handler: router,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			errC := make(chan error, 1)
			go func() {
				if err := srv.Serve(lis); err != nil && err != http.ErrServerClosed {
					errC <- err
				}
			}()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errC:
				return err
			case <-time.After(time.Second):
				for _, r := range router.Routes() {
					logs.Info("注册HTTP路由", zap.String("handler", r.Handler), zap.String("method", r.Method), zap.String("path", r.Path))
				}
				logs.Info("HTTP服务器启动成功", zap.String("监听地址", lis.Addr().String()))
				return nil
			}
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return router
}

func noMethod(ctx *gin.Context) {
	logs.Warn("noMethod:路由不存在", zap.String("url", ctx.Request.Method+":"+ctx.Request.URL.Path))
	wrapper.Wrap(ctx).Fail(base.CodeErrNotFound)
}

func noRoute(ctx *gin.Context) {
	logs.Warn("noRoute:路由不存在", zap.String("url", ctx.Request.Method+":"+ctx.Request.URL.Path))
	wrapper.Wrap(ctx).Fail(base.CodeErrNotFound)
}

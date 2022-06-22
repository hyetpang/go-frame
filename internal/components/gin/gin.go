/*
 * @Date: 2022-04-30 16:15:16
 * @LastEditTime: 2022-05-13 10:06:10
 * @FilePath: /go-frame/internal/components/gin/gin.go
 */
package gin

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/HyetPang/go-frame/internal/adapter/log"
	"github.com/HyetPang/go-frame/pkgs/base"
	"github.com/HyetPang/go-frame/pkgs/common"
	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/HyetPang/go-frame/pkgs/wrapper"
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(zapLog *zap.Logger, lc fx.Lifecycle) gin.IRouter {
	conf := new(config)
	err := viper.UnmarshalKey("http", &conf)
	if err != nil {
		logs.Fatal("http配置Unmarshal到对象出错", zap.Error(err))
	}
	router := gin.New()
	if conf.IsProd {
		gin.SetMode(gin.ReleaseMode)
	}
	router.Use(ginzap.Ginzap(zapLog, time.RFC3339, true))
	router.Use(gin.RecoveryWithWriter(log.NewGinRecoveryZapLog()))
	router.NoRoute(noMethod)
	router.NoMethod(noMethod)
	// 健康检查
	router.GET("/health_check", func(ctx *gin.Context) {
		wrapper.Wrap(ctx).Success("ok")
	})
	// 文档
	if conf.IsDoc {
		router.GET(conf.DocPrefix+"/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	}
	if conf.IsPprof {
		// TODO 加权限
		pprof.Register(router)
	}
	lis, err := net.Listen("tcp4", conf.Addr)
	if err != nil {
		logs.Error("http服务地址监听出错", zap.Error(err), zap.String("监听的地址", conf.Addr))
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
					logs.Info("注册的路由=>", zap.String("method", r.Method), zap.String("url", r.Path), zap.String("handler", r.Handler))
				}
				logs.Info("HTTP服务器启动成功", zap.String("监听地址", lis.Addr().String()))
				return nil
			}
		},
		OnStop: func(ctx context.Context) error {
			err := srv.Shutdown(ctx)
			if err != context.Canceled {
				return err
			}
			return nil
		},
	})
	return router
}

func noMethod(ctx *gin.Context) {
	logs.Warn("noMethod:路由不存在", zap.String("url", ctx.Request.Method+":"+ctx.Request.URL.Path))
	wrapper.Wrap(ctx).Fail(base.CodeErrNotFound)
}

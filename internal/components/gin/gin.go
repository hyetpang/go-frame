/*
 * @Date: 2022-04-30 16:15:16
 * @LastEditTime: 2022-05-01 22:55:12
 * @FilePath: \go-frame\internal\components\gin\gin.go
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
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(zapLog *zap.Logger, lc fx.Lifecycle) gin.IRouter {
	if viper.GetString("server.run_mode") != common.DevMode {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(ginzap.Ginzap(zapLog, time.RFC3339, true))
	router.Use(gin.RecoveryWithWriter(log.NewGinRecoveryZapLog()))
	router.NoRoute(noMethod)
	router.NoMethod(noMethod)
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

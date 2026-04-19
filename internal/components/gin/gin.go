package gin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/hyetpang/go-frame/pkgs/base"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/jpillora/overseer"
	"github.com/penglongli/gin-metrics/ginmetrics"
	"github.com/spf13/viper"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(zapLog *zap.Logger, lc fx.Lifecycle) (gin.IRouter, error) {
	router, conf, err := newGin(zapLog)
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
			case <-time.After(time.Second):
				if !conf.IsProd {
					for _, r := range router.Routes() {
						logs.Info("注册的路由=>", zap.String("method", r.Method), zap.String("url", r.Path), zap.String("handler", r.Handler))
					}
				}
				logs.Info("HTTP服务器启动成功", zap.String("监听地址", lis.Addr().String()))
				return nil
			}
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

func NewWithGraceRestart(zapLog *zap.Logger, state overseer.State, lc fx.Lifecycle) (gin.IRouter, error) {
	if state.Listener == nil {
		return nil, errors.New("网络监听器对象(overseer.State.Listener)没有初始化")
	}
	router, _, err := newGin(zapLog)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			errC := make(chan error, 1)
			go func() {
				if err := router.RunListener(state.Listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					errC <- err
				}
			}()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errC:
				return err
			case <-time.After(time.Second):
				return nil
			}
		},
		OnStop: func(ctx context.Context) error {
			logs.Info("http服务器已关闭...")
			return nil
		},
	})
	return router, nil
}

func newGin(zapLog *zap.Logger) (*gin.Engine, *config, error) {
	conf := new(config)
	if err := viper.UnmarshalKey("http", conf); err != nil {
		return nil, nil, fmt.Errorf("http配置Unmarshal到对象出错: %w", err)
	}
	common.MustValidate(conf)
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
	router.NoRoute(noMethod)
	router.NoMethod(noMethod)
	// 健康检查
	router.GET("/health_check", func(ctx *gin.Context) {
		common.Wrap(ctx).Success("ok")
	})
	// 文档
	if conf.IsDoc {
		router.GET(conf.DocPath, ginSwagger.WrapHandler(swaggerfiles.Handler))
	}
	if conf.IsPprof {
		// TODO 加权限
		if len(conf.PprofPrefix) > 0 {
			pprof.Register(router, conf.PprofPrefix)
		} else {
			pprof.Register(router)
		}
	}
	return router, conf, nil
}

func noMethod(ctx *gin.Context) {
	url := ctx.Request.Method + ":" + ctx.Request.URL.Path
	logs.Error("接口不存在", zap.String("method", ctx.Request.Method), zap.String("url", url), zap.String("ip", ctx.ClientIP()))
	common.Wrap(ctx).Fail(base.CodeErrNotFound)
}

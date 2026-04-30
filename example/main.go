package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hyetpang/go-frame/pkgs/app"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/hyetpang/go-frame/pkgs/options"
	"github.com/robfig/cron/v3"
)

func main() {
	app.Run(
		options.WithConfigFile("./conf/app.toml"), // 配置文件路径,默认是./conf/app.toml
		options.WithHttp(),                        // 对应http配置
		options.WithMysql(),                       // 对应mysql配置
		// options.WithRedis(),     // 对应redis配置
		options.WithTasks(), // 这是定时任务,项目中用到定时任务可加这个
		// options.WithLogNotice(), // 对应配置文件log_notice段配置
		options.WithGRPCClient(),
		options.WithGRPCServer(),
		options.WithEtcd(),
		options.WithInvokes(registerTasks, registerRouter),
	)
}

func registerTasks(c *cron.Cron) {
	_, _ = c.AddFunc("0/2 * * * * ?", func() {
		logs.Debug("zap   =====================")
		fmt.Println("=================end")
		//_ = api.CacheExpiresGoogleToken()
	})
}

func registerRouter(r gin.IRouter) {
	r.GET("/ping", func(ctx *gin.Context) {
		common.Wrap(ctx).Success("pong")
	})
}

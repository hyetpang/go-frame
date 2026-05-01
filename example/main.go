// Package main 是 go-frame 的官方 example,演示框架对外暴露的"完整能力面"。
//
// 本 main 同时启用 HTTP/MySQL/gRPC/Etcd/Tasks/Tracing/LogNotice/Kafka 等多个组件,
// 目的是让初次接入的开发者一眼看到所有 With* option 怎么搭配。生产工程通常只用其中
// 几项(例如纯生产者服务只需要 WithKafkaSyncProducer + WithGRPCClient),按需裁剪即可,
// 框架对未启用的组件零依赖、零成本。
//
// 多环境配置:本 main 在启动时读取环境变量 APP_ENV 推导配置文件路径:
//
//	APP_ENV=dev  -> ./conf/app.dev.toml
//	APP_ENV=prod -> ./conf/app.prod.toml
//	未设置       -> ./conf/app.toml(向后兼容默认路径)
//
// 配置层 internal/config 已实现 LoadWithEnv 的 base+overlay 合并能力,业务方
// 在自定义启动流程时可以直接调用,本 example 通过 WithConfigFile 选择主文件。
package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hyetpang/go-frame/pkgs/app"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/hyetpang/go-frame/pkgs/options"
	"github.com/robfig/cron/v3"
)

func main() {
	app.Run(
		// 配置文件:按 APP_ENV 选择,fallback 到 ./conf/app.toml
		options.WithConfigFile(resolveConfigFile()),

		// HTTP 服务(swagger/pprof/metrics 由配置开关控制)
		options.WithHttp(),

		// 持久化与缓存
		options.WithMysql(),
		// options.WithRedis(), // 用到 redis 时打开

		// gRPC 服务发现走 etcd:WithEtcd 必须在 WithGRPCServer/Client 之前
		options.WithEtcd(),
		options.WithGRPCServer(),
		options.WithGRPCClient(),

		// 定时任务:cron.New 已使用 WithSeconds,表达式按 6 段(秒 分 时 日 月 周)
		options.WithTasks(),

		// OpenTelemetry 分布式追踪:tracing.enable=false 时使用 noop TracerProvider,
		// 零成本回退,所以推荐生产也调用本 option,改配置文件控制是否上报。
		options.WithTracing(),

		// 错误日志通知链路:logs.Error 触发后会按 log_notice 段配置发往企业微信/邮件/飞书/telegram,
		// 同一错误在 limit_window_seconds 内聚合避免刷屏。
		options.WithLogNotice(),

		// Kafka 拆分版:本 example 演示"纯同步生产者"场景,只建必要的连接资源。
		// 其他组合:
		//   - 自管 Producer/Consumer:options.WithKafkaClient()
		//   - 异步生产:options.WithKafkaAsyncProducer()
		//   - 纯消费:options.WithKafkaConsumer()
		//   - 全部:options.WithKafka()(已 Deprecated,仅向后兼容)
		options.WithKafkaSyncProducer(),

		// 业务路由与定时任务注册
		options.WithInvokes(registerTasks, registerRouter),
	)
}

// resolveConfigFile 按 APP_ENV 推导配置文件路径,环境未设置时回落到默认 app.toml。
// 这与 internal/config.LoadWithEnv 的 base+overlay 思路一致,但 example 这里
// 直接选择主文件,更接近常见的"按环境部署不同 image"做法。
func resolveConfigFile() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		return "./conf/app.toml"
	}
	return fmt.Sprintf("./conf/app.%s.toml", env)
}

// registerTasks 演示如何注册 cron 任务。
// 注意 1:框架的 cron 已启用 WithSeconds(),表达式必须是 6 段(秒 分 时 日 月 周)。
// 注意 2:AddFunc 的 error 必须显式处理,robfig/cron/v3 不支持 quartz 的 "?" 占位符,
//
//	早期版本里把 err 丢给 _ 会导致表达式解析失败时静默不跑。
func registerTasks(c *cron.Cron) {
	if _, err := c.AddFunc("*/2 * * * * *", func() {
		logs.Debug("cron tick: 每 2 秒触发一次")
	}); err != nil {
		logs.Fatal("注册定时任务失败")
	}
}

// registerRouter 演示最小的 gin 路由注册;响应统一通过 common.Wrap 包装。
func registerRouter(r gin.IRouter) {
	r.GET("/ping", func(ctx *gin.Context) {
		common.Wrap(ctx).Success("pong")
	})
}

package main

import (
	"github.com/HyetPang/go-frame/pkgs/app"
	"github.com/HyetPang/go-frame/pkgs/options"
)

func main() {
	app.Run(
		options.WithHttp(nil),   // 对应http配置,这里的传参是使用的swagger文档，详情请参考gin接入swagger教程
		options.WithMysql(),     // 对应mysql配置
		options.WithRedis(),     // 对应redis配置
		options.WithTasks(),     // 这是定时任务,项目中用到定时任务可加这个
		options.WithLogNotice(), // 对应配置文件log_notice段配置
	)
}

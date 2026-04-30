# go-frame

`go-frame` 是一个基于 `fx`、`gin`、`zap`、`gorm`、`go-redis`、`grpc` 的 Go 服务脚手架。框架负责统一启动、配置加载、日志、HTTP、gRPC、常用组件注入和优雅停机，业务侧通过 `options.WithXxx()` 按需启用组件。

## 快速开始

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hyetpang/go-frame/pkgs/app"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/options"
)

func register(router gin.IRouter) {
	router.GET("/ping", func(ctx *gin.Context) {
		common.Wrap(ctx).Success("pong")
	})
}

func main() {
	app.Run(
		options.WithConfigFile("./conf/app.dev.toml"),
		options.WithHttp(),
		options.WithInvokes(register),
	)
}
```

运行：

```bash
go run ./example
```

## 配置文件

示例配置在 `example/conf`：

- `app.dev.toml`：开发环境配置，默认开启 swagger、pprof、metrics，便于调试。
- `app.prod.toml`：生产环境安全默认值，默认关闭 swagger、pprof，保留 metrics。
- `app.toml`：兼容旧默认路径，内容保持开发环境风格。

生产环境建议显式指定配置文件：

```go
app.Run(
	options.WithConfigFile("./conf/app.prod.toml"),
	options.WithHttp(),
)
```

配置只在启动层读取一次并解析成强类型对象，组件通过 `fx` 接收自己的配置 section，不直接读取全局 `viper`。

## 常用组件

HTTP：

```go
app.Run(options.WithHttp())
```

MySQL：

```go
app.Run(options.WithMysql())
```

多库配置使用 `[[mysql]]`，单库配置可以继续使用 `[mysql]`。单库注入会优先选择 `name = "default"`。

Redis：

```go
app.Run(options.WithRedis())
```

gRPC：

```go
app.Run(options.WithGRPCServer())
app.Run(options.WithGRPCClient())
```

使用 etcd 服务发现：

```go
app.Run(
	options.WithEtcd(),
	options.WithGRPCServer(options.GrpcOptionEtcd()),
)
```

日志通知：

```go
app.Run(options.WithLogNotice())
```

同一错误会按 `filename + line + msg` 在配置窗口内聚合，避免高频刷屏。

## HTTP 约定

默认响应结构：

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

框架预留错误码：

- `0`：成功
- `1`：系统错误
- `2`：参数无效
- `3`：未找到

业务错误码建议从 `100` 开始。

## 安全建议

- 生产环境关闭 `is_doc` 和 `is_pprof`。
- 如必须开启 pprof，必须配置 `pprof_username` 和 `pprof_password`，并建议放在内网或网关鉴权之后。
- metrics 可以保留开启，但建议只暴露给 Prometheus 所在网络。
- `log_notice.notice`、数据库密码、Redis 密码不要提交真实值。

## 开发工具

```bash
go install github.com/swaggo/swag/cmd/swag@latest
go install github.com/dkorunic/betteralign/cmd/betteralign@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install github.com/hyetpang/go-code-gen@latest
```

`protoc` 需要从 Protocol Buffers 官方 release 下载对应平台版本。

## 验证命令

```bash
go test ./...
go vet ./...
gofmt -w $(find . -name '*.go' -not -path './vendor/*')
```

## 依赖说明

gRPC middleware 已收敛到 `go-grpc-middleware/v2` 和 `providers/prometheus`。旧 Redis 客户端版本主要来自 `go-redsync/redsync/v4` 的测试依赖链，当前业务代码直接使用的是 `github.com/redis/go-redis/v9`。

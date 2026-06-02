package nats

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hyetpang/go-frame/internal/lifecycle"
	"github.com/hyetpang/go-frame/pkgs/common"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// parseAddrs 解析逗号分隔的 NATS 地址串,对每个地址做 TrimSpace 并过滤空串,
// 避免 "a, b" 这类带空格的配置产生非法地址。
func parseAddrs(addr string) []string {
	parts := strings.Split(addr, ",")
	addrs := make([]string, 0, len(parts))
	for _, p := range parts {
		if a := strings.TrimSpace(p); a != "" {
			addrs = append(addrs, a)
		}
	}
	return addrs
}

// NewConn 创建 *nats.Conn,作为核心 NATS 与 JetStream 的共享底座。
// Username 留空走无认证连接;运行期断连/重连/异步错误经回调桥接到 zap。
// 与 sarama 不同,NATS 不替换全局 Logger,而是通过连接事件回调记日志。
func NewConn(lc fx.Lifecycle, zapLog *zap.Logger, conf *config) (*nats.Conn, error) {
	if err := common.Validate(conf); err != nil {
		return nil, fmt.Errorf("nats配置验证不通过: %w", err)
	}
	// closedCh 在连接 ClosedHandler 触发时关闭,供 OnStop 同步等待 Drain 真正完成。
	closedCh := make(chan struct{})
	opts := []nats.Option{
		nats.MaxReconnects(conf.MaxReconnects),
		nats.ReconnectWait(time.Duration(conf.ReconnectWaitSec) * time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			zapLog.Warn("nats连接断开", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			zapLog.Info("nats重连成功", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ErrorHandler(func(_ *nats.Conn, sub *nats.Subscription, err error) {
			subject := ""
			if sub != nil {
				subject = sub.Subject
			}
			zapLog.Error("nats异步错误", zap.String("subject", subject), zap.Error(err))
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			close(closedCh)
		}),
	}
	if conf.Name != "" {
		opts = append(opts, nats.Name(conf.Name))
	}
	if conf.Username != "" {
		opts = append(opts, nats.UserInfo(conf.Username, conf.Password))
	}
	nc, err := nats.Connect(strings.Join(parseAddrs(conf.Addr), ","), opts...)
	if err != nil {
		return nil, fmt.Errorf("连接nats出错 addr=%s: %w", conf.Addr, err)
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// Drain 处理完未决订阅/发布后触发 ClosedHandler;<-closedCh 同步等待其真正完成。
			// CloseWithContext 在 goroutine 中跑此函数并与 fx Stop 的 ctx deadline 竞速,
			// 超时则放弃等待,不拖死整个 Stop 流程。
			closer := func() error {
				// 连接若已被 SDK 关闭(如 MaxReconnects 耗尽),Drain 返回 ErrConnectionClosed,
				// 此时连接已关闭、ClosedHandler 已触发,视为正常,不当作关闭错误上报。
				if e := nc.Drain(); e != nil && !errors.Is(e, nats.ErrConnectionClosed) {
					return e
				}
				<-closedCh
				return nil
			}
			if e := lifecycle.CloseWithContext(ctx, "nats", closer); e != nil {
				logs.Error("关闭nats出错", zap.Error(e))
				return e
			}
			return nil
		},
	})
	return nc, nil
}

// NewJetStream 基于共享 *nats.Conn 创建 JetStream 上下文。
// 它是 *nats.Conn 的轻量包装,不额外建连,随连接关闭自然失效,无需独立生命周期 hook。
// 不在此创建 Stream/Consumer —— 与 kafka 不建 topic 一致,声明属于业务职责。
func NewJetStream(nc *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("创建nats jetstream出错: %w", err)
	}
	return js, nil
}

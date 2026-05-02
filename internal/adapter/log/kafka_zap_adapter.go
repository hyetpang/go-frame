package log

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// kafkaLog 将 sarama 的 StdLogger 接口适配到 zap。
//
// sarama 的 Logger 接口本身不分级,所有内部消息(metadata refresh / leader change /
// 连接错误 / SASL 失败等)都走同一个 Print/Printf。旧实现一律打到 Debug 级别,
// 生产环境(Info 级别)看不到任何 Kafka 内部线索 — 出问题排查只能靠重启加 Debug。
//
// 这里在 Print 入口做关键字识别,把含 error/fail/disconnect 等关键词的消息上调到
// Warn 级别,普通运行日志保持 Debug。让生产 Info 级别能感知 Kafka 异常,同时不刷屏。
type kafkaLog struct {
	*zap.Logger
}

func NewKafkaLog(log *zap.Logger) kafkaLog {
	return kafkaLog{
		Logger: log,
	}
}

func (cl kafkaLog) Printf(msg string, format ...any) {
	cl.Print(fmt.Sprintf(msg, format...))
}

func (cl kafkaLog) Print(v ...any) {
	msg := fmt.Sprint(v...)
	if isKafkaWarnMessage(msg) {
		cl.Logger.Warn("kafka", zap.String("message", msg))
		return
	}
	cl.Logger.Debug("kafka", zap.String("message", msg))
}

func (cl kafkaLog) Println(v ...any) {
	cl.Print(v...)
}

// kafkaWarnKeywords 命中即把 sarama 内部消息上调到 Warn 级别。
// 列表保持小而精,避免误把 metadata refresh 等正常消息上调。
var kafkaWarnKeywords = []string{
	"error",
	"fail",
	"disconnect",
	"closed network connection",
	"refused",
	"timed out",
	"timeout",
	"unauthorized",
	"sasl",
}

func isKafkaWarnMessage(msg string) bool {
	lower := strings.ToLower(msg)
	for _, kw := range kafkaWarnKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

package log

import (
	"fmt"

	"go.uber.org/zap"
)

type kafkaLog struct {
	*zap.Logger
}

func NewKafkaLog(log *zap.Logger) kafkaLog {
	return kafkaLog{
		Logger: log,
	}
}

func (cl kafkaLog) Printf(msg string, format ...interface{}) {
	cl.Print(fmt.Sprintf(msg, format...))
}

func (cl kafkaLog) Print(v ...interface{}) {
	cl.Logger.Debug("kafka-log", zap.Any("kafka", v))
}

func (cl kafkaLog) Println(v ...interface{}) {
	cl.Print(v...)
}

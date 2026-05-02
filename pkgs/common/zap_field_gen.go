package common

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GenZapNanoId 生成一个 nano_id 字段,常用于请求级日志的关联追踪。
func GenZapNanoId() zapcore.Field {
	return zap.String("nano_id", GenID())
}

// ObjectMarshaler 把 func(zapcore.ObjectEncoder) 适配为 zapcore.ObjectMarshalerFunc,
// 便于 zap.Object("k", common.ObjectMarshaler(func(oe){...})) 的内联用法。
//
// 注意: 数组字段请直接使用 zap 官方 API(zap.Strings/zap.Ints/zap.Int64s 等),
// 性能更好且符合 zap 习惯。本包不再提供 ZapStringArray/ZapIntArray 包装。
func ObjectMarshaler(f func(zapcore.ObjectEncoder)) zapcore.ObjectMarshalerFunc {
	return func(oe zapcore.ObjectEncoder) error {
		f(oe)
		return nil
	}
}

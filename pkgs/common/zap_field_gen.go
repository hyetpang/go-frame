package common

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func GenZapNanoId() zapcore.Field {
	return zap.String("nano_id", GenID())
}

func ZapStringArray(key string, data []string) zapcore.Field {
	return zap.Array(key, stringArrayMarshaler(data))
}

func ZapIntArray(key string, data []int) zapcore.Field {
	return zap.Array(key, intArrayMarshaler(data))
}

func stringArrayMarshaler(stringArray []string) zapcore.ArrayMarshalerFunc {
	return func(ae zapcore.ArrayEncoder) error {
		for _, v := range stringArray {
			ae.AppendString(v)
		}
		return nil
	}
}

func intArrayMarshaler(intArray []int) zapcore.ArrayMarshalerFunc {
	return func(ae zapcore.ArrayEncoder) error {
		for _, v := range intArray {
			ae.AppendInt(v)
		}
		return nil
	}
}

func ObjectArrayMarshaler(f zapcore.ArrayMarshalerFunc) zapcore.ArrayMarshalerFunc {
	return f
}

func ObjectMarshaler(f func(zapcore.ObjectEncoder)) zapcore.ObjectMarshalerFunc {
	return func(oe zapcore.ObjectEncoder) error {
		f(oe)
		return nil
	}
}

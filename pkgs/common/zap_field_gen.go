package common

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func GenZapNanoId() zapcore.Field {
	return zap.String("nanoId", GenNanoIdString())
}

func GenNanoIdString() string {
	nanoId, err := GenNanoID()
	if err != nil {
		logs.Error("zap nanoid生成出错", zap.Error(err))
		nanoId = strconv.Itoa(int(time.Now().Unix() + rand.Int63n(50)))
	}
	return nanoId
}

func ZapStringArray(key string, data []string) zapcore.Field {
	return zap.Array(key, StringArrayMarshaler(data))
}

func ZapIntArray(key string, data []int) zapcore.Field {
	return zap.Array(key, IntArrayMarshaler(data))
}

func StringArrayMarshaler(stringArray []string) zapcore.ArrayMarshalerFunc {
	return func(ae zapcore.ArrayEncoder) error {
		for _, v := range stringArray {
			ae.AppendString(v)
		}
		return nil
	}
}

func IntArrayMarshaler(intArray []int) zapcore.ArrayMarshalerFunc {
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

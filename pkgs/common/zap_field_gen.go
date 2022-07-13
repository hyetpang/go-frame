package common

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/HyetPang/go-frame/pkgs/logs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func GenZapNanoId() zapcore.Field {
	nanoId, err := GenNanoID()
	if err != nil {
		logs.Error("zap nanoid生成出错", zap.Error(err))
		nanoId = strconv.Itoa(int(time.Now().Unix() + rand.Int63n(50)))
	}
	return zap.String("nanoId", nanoId)
}

func ZapStringArray(key string, data []string) zapcore.Field {
	return zap.Array(key, StringArrayMarshaler(data))
}

func ZapIntArray(key string, data []int) zapcore.Field {
	return zap.Array(key, IntArrayMarshaler(data))
}

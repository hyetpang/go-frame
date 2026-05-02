package common

import (
	"errors"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/hyetpang/go-frame/pkgs/logs"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.uber.org/zap"
)

var (
	alphaLower  string = "abcdefghijklmnopqrstuvwxyz"
	alphaUpper  string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	number      string = "1234567890"
	alphaNumber        = alphaLower + alphaUpper + number
)

// fallbackCounter 在 GenID 走 nanoid 失败回退路径时与 UnixNano + pid 一起拼成 ID,
// 把"同进程同纳秒生成两次"会碰撞的窗口收敛掉,避免 etcd lease key 等需要唯一性的场景误共用。
var fallbackCounter uint64

func GenNanoID() (string, error) {
	return gonanoid.Generate(alphaNumber, 10)
}

// MustGenID 生成 nanoid,失败直接返回 error。
// 适用于唯一性硬约束的场景(etcd 注册 key、分布式锁 value 等),
// 让调用方决定是 fail-fast 还是降级。
func MustGenID() (string, error) {
	return GenNanoID()
}

// GenID 生成 nanoid,失败时回退到 UnixNano + pid + 原子计数器拼成的字符串。
// 仅适用于"对唯一性宽容"的场景(如日志请求 id);硬约束唯一性请用 MustGenID。
//
// 旧实现失败时仅返回 UnixNano,在两个副本同纳秒注册 etcd endpoint 时可能碰撞;
// 现在追加 pid + counter 把碰撞概率降到接近零,即使 nanoid 极端故障也能保住基本唯一性。
func GenID() string {
	nanoId, err := GenNanoID()
	if err != nil {
		logs.Error("nanoid生成出错,回退到 UnixNano+pid+counter", zap.Error(err))
		seq := atomic.AddUint64(&fallbackCounter, 1)
		return strconv.FormatInt(time.Now().UnixNano(), 10) +
			"-" + strconv.Itoa(os.Getpid()) +
			"-" + strconv.FormatUint(seq, 10)
	}
	return nanoId
}

// size 表示生成的id长度, tryCount表示尝试次数,isValid验证生成的id是否有效
func TryGenNanoIDFromAlphaNumber(size, tryCount int, isValid func(id string) (bool, error)) (string, error) {
	// tryCount <= 0 统一当作 1 处理，保证至少尝试一次
	if tryCount <= 0 {
		tryCount = 1
	}
	for i := 0; i < tryCount; i++ {
		id, err := gonanoid.Generate(alphaNumber, size)
		if err != nil {
			return "", err
		}
		ok, err := isValid(id)
		if err != nil {
			return "", err
		}
		if ok {
			return id, nil
		}
	}
	return "", errors.New("生成唯一id超过最大尝试次数:" + strconv.Itoa(tryCount))
}

package common

import (
	"errors"
	"math/rand"
	"strconv"
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

func GenNanoID() (string, error) {
	return gonanoid.Generate(alphaNumber, 10)
}

func GenID() string {
	nanoId, err := GenNanoID()
	if err != nil {
		logs.Error("zap nanoid生成出错", zap.Error(err))
		unixNano := time.Now().Unix()
		r := rand.New(rand.NewSource(unixNano)).Int63n((50))
		nanoId = strconv.Itoa(int(unixNano + r))
	}
	return nanoId
}

// size 表示生成的id长度, tryCount表示尝试次数,isValid验证生成的id是否有效
func TryGenNanoIDFromAlphaNumber(size, tryCount int, isValid func(id string) (bool, error)) (string, error) {
	if tryCount < 0 {
		panic("尝试次数不能小于0")
	}
	if tryCount == 0 {
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

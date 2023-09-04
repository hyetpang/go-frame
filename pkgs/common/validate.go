package common

import (
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/zap"
)

// 数据验证，不通过直接panic
func MustValidate(a any) {
	if err := validator.New().Struct(a); err != nil {
		logs.Error("结构体参数验证不通过", zap.Error(err), zap.Any("struct", a))
		log.Fatalf("结构体参数验证不通过,err:%s,struct:%+v\n", err.Error(), a)
	}
}

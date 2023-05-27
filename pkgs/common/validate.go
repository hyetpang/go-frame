package common

import (
	"log"

	"github.com/go-playground/validator/v10"
)

// 数据验证，不通过直接panic
func MustValidate(a any) {
	if err := validator.New().Struct(a); err != nil {
		log.Fatalf("配置出错,缺少配置数据,err:%s,struct:%v", err.Error(), a)
	}
}

package common

import (
	"log"

	"github.com/go-playground/validator/v10"
)

// 数据验证，不通过直接panic
func MustValidate(a any) {
	if err := validator.New().Struct(a); err != nil {
		log.Fatalf("结构体参数验证不通过,err:%s,struct:%+v\n", err.Error(), a)
	}
}

package common

import (
	"log"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validate     *validator.Validate
	validateOnce sync.Once
)

// 数据验证，不通过直接panic
func MustValidate(a any) {
	if err := Validate(a); err != nil {
		log.Fatalf("结构体参数验证不通过,err:%s,struct:%+v\n", err.Error(), a)
	}
}

func Validate(a any) error {
	validateOnce.Do(func() { validate = validator.New() })
	return validate.Struct(a)
}

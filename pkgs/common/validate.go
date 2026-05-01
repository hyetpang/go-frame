package common

import (
	"fmt"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validate     *validator.Validate
	validateOnce sync.Once
)

// MustValidate 数据验证，不通过则 panic，由调用方（如 fx）决定是否致命，
// 避免 log.Fatalf 调用 os.Exit(1) 绕过 fx Stop 导致数据库连接/Kafka producer 不正常关闭。
func MustValidate(a any) {
	if err := Validate(a); err != nil {
		panic(fmt.Errorf("结构体参数验证不通过: %w, struct: %+v", err, a))
	}
}

func Validate(a any) error {
	validateOnce.Do(func() { validate = validator.New() })
	return validate.Struct(a)
}

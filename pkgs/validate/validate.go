package validate

import (
	"log"

	"github.com/go-playground/validator/v10"
)

// 数据验证，不通过直接panic
func Must(a any) {
	if err := validator.New().Struct(a); err != nil {
		log.Fatalf("配置出错,缺少配置数据:%s", err.Error())
		panic(err)
	}
}

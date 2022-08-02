package validate

import (
	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// 数据验证，不通过直接panic
func Must(a any) {
	if err := validator.New().Struct(a); err != nil {
		logs.Error("配置出错,缺少配置数据!", zap.Error(err))
		panic(err)
	}
}

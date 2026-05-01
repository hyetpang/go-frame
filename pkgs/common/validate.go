package common

import (
	"fmt"
	"sync"

	"github.com/go-playground/validator/v10"
)

// validator.Validate 内部对自定义校验注册不是并发安全的:
// 注册路径(RegisterValidation)与读路径(Struct)若并发执行可能数据竞争,
// 因此用 RWMutex 保护扩展路径,确保业务可在请求路径外随时注册自定义校验。
var (
	validate     *validator.Validate
	validateOnce sync.Once
	validateMu   sync.RWMutex
)

func ensureValidator() *validator.Validate {
	validateOnce.Do(func() { validate = validator.New() })
	return validate
}

// MustValidate 数据验证，不通过则 panic，由调用方（如 fx）决定是否致命，
// 避免 log.Fatalf 调用 os.Exit(1) 绕过 fx Stop 导致数据库连接/Kafka producer 不正常关闭。
func MustValidate(a any) {
	if err := Validate(a); err != nil {
		panic(fmt.Errorf("结构体参数验证不通过: %w, struct: %+v", err, a))
	}
}

func Validate(a any) error {
	v := ensureValidator()
	validateMu.RLock()
	defer validateMu.RUnlock()
	return v.Struct(a)
}

// RegisterValidation 注册自定义校验 tag。建议在启动期(fx Provide/Invoke)调用,
// 与请求路径上的 Validate 互斥执行,避免内部 map 写竞争。
func RegisterValidation(tag string, fn validator.Func) error {
	v := ensureValidator()
	validateMu.Lock()
	defer validateMu.Unlock()
	return v.RegisterValidation(tag, fn)
}

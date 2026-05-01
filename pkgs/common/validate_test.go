package common

import (
	"testing"
)

// TestMustValidatePanicsOnInvalidStruct 验证 MustValidate 对不合法结构体触发 panic，
// 而非调用 os.Exit(1) 绕过 fx Stop 流程。
func TestMustValidatePanicsOnInvalidStruct(t *testing.T) {
	type reqStruct struct {
		Name string `validate:"required"`
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("期望 MustValidate 在验证不通过时触发 panic，但实际上没有")
		}
	}()

	// Name 为空，验证必然不通过，应触发 panic
	MustValidate(reqStruct{Name: ""})
}

// TestMustValidatePassesOnValidStruct 验证 MustValidate 对合法结构体不触发 panic。
func TestMustValidatePassesOnValidStruct(t *testing.T) {
	type reqStruct struct {
		Name string `validate:"required"`
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("期望 MustValidate 在验证通过时不触发 panic，但实际 panic: %v", r)
		}
	}()

	MustValidate(reqStruct{Name: "valid"})
}

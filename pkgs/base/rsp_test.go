package base

import (
	"errors"
	"testing"
)

func TestCodeErrImpl_FormatMsgDoesNotMutateOriginal(t *testing.T) {
	orig := NewCodeErr(42, "hello %s %d")
	out := orig.FormatMsg("world", 7)

	if got := out.GetMsg(); got != "hello world 7" {
		t.Errorf("formatted msg = %q, want %q", got, "hello world 7")
	}
	if orig.GetMsg() != "hello %s %d" {
		t.Errorf("original mutated: %q", orig.GetMsg())
	}
	if out == orig {
		t.Error("FormatMsg should return a new instance")
	}
}

func TestCodeErrImpl_Error(t *testing.T) {
	ce := NewCodeErr(7, "boom")
	if got := ce.Error(); got != "7:boom" {
		t.Errorf("Error() = %q, want 7:boom", got)
	}
}

func TestCodeErrImpl_IsSuccess(t *testing.T) {
	if !NewCodeErr(0, "ok").IsSuccess() {
		t.Fatal("code 0 should be success")
	}
	if NewCodeErr(1, "no").IsSuccess() {
		t.Fatal("code 1 should not be success")
	}
}

func TestGetCodeI(t *testing.T) {
	if GetCodeI(nil) != nil {
		t.Fatal("nil err should return nil")
	}
	custom := NewCodeErr(99, "custom")
	if GetCodeI(custom) != custom {
		t.Fatal("CodeErrI should be returned as-is")
	}
	if GetCodeI(errors.New("plain")) != CodeErrSystem {
		t.Fatal("plain error should map to CodeErrSystem")
	}
}

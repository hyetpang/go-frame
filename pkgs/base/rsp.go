package base

import (
	"fmt"
	"strconv"
)

type CodeErrI interface {
	GetMsg() string
	FormatMsg(...any) CodeErrI
	GetCode() uint
	IsSuccess() bool
	error
}
type CodeErrImpl struct {
	Data any    `json:"data"`
	Msg  string `json:"msg"`
	FMsg string `json:"-"`
	Code uint   `json:"code"`
}

func NewCodeErr(code uint, msg string) CodeErrI {
	return &CodeErrImpl{Code: code, Msg: msg, Data: struct{}{}}
}
func (ce *CodeErrImpl) GetCode() uint {
	return ce.Code
}
func (ce *CodeErrImpl) GetMsg() string {
	if len(ce.FMsg) > 0 {
		return ce.FMsg
	}
	return ce.Msg
}
func (ce *CodeErrImpl) IsSuccess() bool {
	return ce.Code == 0
}
func (ce *CodeErrImpl) Error() string {
	return strconv.Itoa(int(ce.GetCode())) + ":" + ce.GetMsg()
}
func (ce *CodeErrImpl) FormatMsg(args ...any) CodeErrI {
	clone := *ce
	clone.FMsg = fmt.Sprintf(ce.Msg, args...)
	return &clone
}
func toCodeI(err error) (CodeErrI, bool) {
	if err == nil {
		return nil, true
	}
	var errInterface any = err
	codeE, ok := errInterface.(CodeErrI)
	return codeE, ok
}
func GetCodeI(err error) CodeErrI {
	if err == nil {
		return nil
	}
	codeE, ok := toCodeI(err)
	if ok {
		return codeE
	}
	return CodeErrSystem
}

// 响应结构体
type ResultRsp struct {
	Data any    `json:"data"`
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

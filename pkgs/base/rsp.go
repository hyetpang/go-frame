/*
 * @Date: 2022-05-05 15:26:12
 * @LastEditTime: 2022-05-17 11:12:02
 * @FilePath: /ultrasdk.center.go/projects/ultrasdk/go-frame/pkgs/base/rsp.go
 * @Author: guangming.zhang hyetpang@yeah.net
 * @LastEditors: guangming.zhang hyetpang@yeah.net
 * @Description: 基本数据
 *
 * Copyright (c) 2022 by hero, All Rights Reserved.
 */
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
	Code uint   `json:"code"`
	Msg  string `json:"msg"`
	err  error  `json:"-"`
	Data any    `json:"data"`
}

func NewCodeErr(code uint, msg string) CodeErrI {
	return &CodeErrImpl{Code: code, Msg: msg, Data: struct{}{}}
}

func (ce *CodeErrImpl) GetCode() uint {
	return ce.Code
}

func (ce *CodeErrImpl) GetMsg() string {
	return ce.Msg
}

func (ce *CodeErrImpl) IsSuccess() bool {
	return ce.Code == 0
}

func (ce *CodeErrImpl) Error() string {
	err := ce.err
	if err != nil {
		return err.Error()
	}
	return strconv.Itoa(int(ce.GetCode())) + ":" + ce.GetMsg()
}

func (ce *CodeErrImpl) FormatMsg(args ...any) CodeErrI {
	ce.Msg = fmt.Sprintf(ce.Msg, args)
	return ce
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

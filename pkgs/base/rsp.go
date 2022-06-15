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

import "fmt"

type CodeErrI interface {
	GetMsg() string
	FormatMsg(...any) CodeErrI
	GetCode() uint
	IsSuccess() bool
	error
}

type codeErrImpl struct {
	Code uint   `json:"code"`
	Msg  string `json:"msg"`
	err  error  `json:"-"`
}

func NewCodeErr(code uint, msg string) CodeErrI {
	return &codeErrImpl{Code: code, Msg: msg}
}

func (ce *codeErrImpl) GetCode() uint {
	return ce.Code
}

func (ce *codeErrImpl) GetMsg() string {
	return ce.Msg
}

func (ce *codeErrImpl) IsSuccess() bool {
	return ce.Code == 0
}

func (ce *codeErrImpl) Error() string {
	err := ce.err
	if err != nil {
		return err.Error()
	}
	return ""
}

func (ce *codeErrImpl) FormatMsg(args ...any) CodeErrI {
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

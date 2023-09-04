/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-05-01 22:29:19
 * @FilePath: \go-frame\pkgs\common\utils.go
 */
package common

import (
	"unsafe"
)

func Panic(err error) {
	if err != nil {
		panic(err)
	}
}

// string转byte
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// bytes转string
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

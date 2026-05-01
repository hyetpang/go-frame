package common

import "unsafe"

// StringToBytes 零拷贝将 string 转为 []byte。
// 警告：返回的切片共享底层内存，禁止修改或 append（会触发 panic 或未定义行为）。
func StringToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// BytesToString 零拷贝将 []byte 转为 string。
// 警告：调用方在转换后必须保证入参 []byte 不再被修改。
func BytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

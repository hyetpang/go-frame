/*
 * @Date: 2022-04-30 10:34:56
 * @LastEditTime: 2022-05-01 22:29:19
 * @FilePath: \go-frame\pkgs\common\utils.go
 */
package common

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
	"regexp"
	"strconv"
	"time"
	"unsafe"

	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"
)

func Panic(err error) {
	if err != nil {
		panic(err)
	}
}

func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
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
func BytesString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func Now() int64 {
	return time.Now().Unix()
}

// IsInArray找到给定的ele是否在arr中
func IsInArray[E comparable](arr []E, ele E) bool {
	return slices.Contains(arr, ele)
}

func CheckEmail(email string) bool {
	matched, _ := regexp.MatchString(`\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`, email)
	return matched
}

func Rand(num int) string {
	result := ""
	temp := 0
	for i := 0; i < num; i++ {
		rand.Seed(time.Now().UnixNano())
		temp = rand.Intn(10)
		result += strconv.Itoa(temp)
	}
	return result
}

func StringArrayMarshaler(stringArray []string) zapcore.ArrayMarshalerFunc {
	var ignoreURLArrayMarshaler zapcore.ArrayMarshalerFunc = func(ae zapcore.ArrayEncoder) error {
		for _, v := range stringArray {
			ae.AppendString(v)
		}
		return nil
	}
	return ignoreURLArrayMarshaler
}

func IntArrayMarshaler(intArray []int) zapcore.ArrayMarshalerFunc {
	var ignoreURLArrayMarshaler zapcore.ArrayMarshalerFunc = func(ae zapcore.ArrayEncoder) error {
		for _, v := range intArray {
			ae.AppendInt(v)
		}
		return nil
	}
	return ignoreURLArrayMarshaler
}

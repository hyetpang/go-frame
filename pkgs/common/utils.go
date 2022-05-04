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
func String2Byte(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
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

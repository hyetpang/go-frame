package common

import (
	"fmt"

	"golang.org/x/sync/singleflight"
)

var dsf singleflight.Group

// 缓存不存在
var CacheNotExist = fmt.Errorf("cache is not exist")

type (
	DBGetterFunc    func() (any, error)
	CacheGetterFunc func() (any, bool, error) // 返回的 bool 值表示是否缓存存在值,true=>存在,false=>不存在
)

// SingleGet 执行singleFlight模式
func SingleGet(key string, cacheGetter CacheGetterFunc, dbGetter DBGetterFunc) (any, error) {
	result, ok, err := cacheGetter()
	if err != nil { // 缓存获取数据失败
		return nil, fmt.Errorf("get from cache err:%w", err)
	}
	if !ok { // 缓存不存在
		// 从数据库里获取
		result, err, _ = dsf.Do(key, dbGetter)
		if err != nil {
			return nil, fmt.Errorf("get from db err: %w", err)
		}
	}
	return result, nil
}

package common

import (
	"fmt"

	"golang.org/x/sync/singleflight"
)

var dsf singleflight.Group

// 缓存不存在
var CacheNotExist = fmt.Errorf("cache not exist")

type GetterFunc func() (any, error)

// SingleGet 执行singleFlight模式
func SingleGet(key string, cacheGetter, dbGetter GetterFunc) (any, error) {
	result, err := cacheGetter()
	if err == CacheNotExist { // 缓存不存在
		// 从数据库里获取
		result, err, _ = dsf.Do(key, dbGetter)
		if err != nil {
			return nil, fmt.Errorf("get from db err: %v", err)
		}

	} else if err != nil { // 获取数据失败
		return nil, fmt.Errorf("get from cache err: %v", err)
	}
	return result, nil
}

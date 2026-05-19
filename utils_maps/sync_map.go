package utils_maps

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"sync"
	"time"
)

/**

使用 LocalSyncMap（ sync.Map） 实现一个本地缓存，用于存储 key-value 数据

1，提供一个初始化的方法 Init()，用于初始化 LocalSyncMap
2，Init() 会启用一个 goroutine 定时清理过期数据（10分钟执行一次），获取当前时间，如果当前时间大于 LocalSyncMapValue.ExpireTime，则删除该数据
3，提供一个 Set(key string, value interface{}, expireTime int64) 方法，用于设置数据，expireTime 为过期时间，单位为秒
	3.1，value需要先进行深拷贝处理，避免内存泄露
	3.2，如果 key 已经存在，则更新数据
	3.3，如果 key 不存在，则创建数据
4，提供一个 Get(key string) (interface{}, bool) 方法，用于获取数据
	4.1，如果 key 不存在，则返回 false
	4.2，如果 key 存在，则返回 true 和数据
5，提供一个 SafeGet(key string) (interface{}, bool) 方法，用于获取数据
	5.1，如果 key 不存在，则返回 false
	5.2，如果 key 存在，拿到的数据需要进行深拷贝处理，避免内存泄露，然后返回 true 和深拷贝后的数据
6，提供一个 Del(key string) 方法，用于删除数据
	5.1，如果 key 不存在，则返回 false
	5.2，如果 key 存在，则返回 true

备注：深拷贝的方式，最差的方式是使用 json.Marshal 和 json.Unmarshal；建议使用一个性能更好的方式！

**/

type LocalSyncMap struct {
	m        sync.Map
	stopChan chan struct{}
	once     sync.Once
	stopOnce sync.Once
}

type LocalSyncMapValue struct {
	Value      interface{}
	ExpireTime int64
}

// Init 初始化 LocalSyncMap，启动定时清理 goroutine。使用前须调用；与 Stop 成对使用。
func (lsm *LocalSyncMap) Init() {
	lsm.once.Do(func() {
		lsm.stopChan = make(chan struct{})
		go lsm.cleanupExpired()
	})
}

// Stop 停止定时清理 goroutine；可安全多次调用。
func (lsm *LocalSyncMap) Stop() {
	lsm.stopOnce.Do(func() {
		if lsm.stopChan != nil {
			close(lsm.stopChan)
		}
	})
}

// cleanupExpired 定时清理过期数据，每10分钟执行一次
func (lsm *LocalSyncMap) cleanupExpired() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now().Unix()
			lsm.m.Range(func(key, value interface{}) bool {
				if v, ok := value.(*LocalSyncMapValue); ok {
					if now > v.ExpireTime {
						lsm.m.Delete(key)
					}
				}
				return true
			})
		case <-lsm.stopChan:
			return
		}
	}
}

// deepCopy 使用 gob 进行深拷贝；按值类型创建目标，避免 interface{} 解码失败。
func deepCopy(src interface{}) (interface{}, error) {
	if src == nil {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return nil, err
	}

	dst := reflect.New(reflect.TypeOf(src)).Interface()
	if err := gob.NewDecoder(&buf).Decode(dst); err != nil {
		return nil, err
	}
	return reflect.ValueOf(dst).Elem().Interface(), nil
}

// Set 设置数据，expireTime 为过期时间（秒）
func (lsm *LocalSyncMap) Set(key string, value interface{}, expireTime int64) error {
	// 深拷贝 value，避免内存泄露
	copiedValue, err := deepCopy(value)
	if err != nil {
		return err
	}

	// 计算过期时间戳
	expireTimestamp := time.Now().Unix() + expireTime

	// 存储数据
	lsm.m.Store(key, &LocalSyncMapValue{
		Value:      copiedValue,
		ExpireTime: expireTimestamp,
	})

	return nil
}

// Get 获取数据，不进行深拷贝
func (lsm *LocalSyncMap) Get(key string) (interface{}, bool) {
	value, ok := lsm.m.Load(key)
	if !ok {
		return nil, false
	}

	v, ok := value.(*LocalSyncMapValue)
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if time.Now().Unix() > v.ExpireTime {
		lsm.m.Delete(key)
		return nil, false
	}

	return v.Value, true
}

// SafeGet 获取数据，返回深拷贝后的数据
func (lsm *LocalSyncMap) SafeGet(key string) (interface{}, bool) {
	value, ok := lsm.m.Load(key)
	if !ok {
		return nil, false
	}

	v, ok := value.(*LocalSyncMapValue)
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if time.Now().Unix() > v.ExpireTime {
		lsm.m.Delete(key)
		return nil, false
	}

	// 深拷贝数据，避免内存泄露
	copiedValue, err := deepCopy(v.Value)
	if err != nil {
		return nil, false
	}

	return copiedValue, true
}

// Del 删除数据
func (lsm *LocalSyncMap) Del(key string) bool {
	_, ok := lsm.m.Load(key)
	if !ok {
		return false
	}

	lsm.m.Delete(key)
	return true
}

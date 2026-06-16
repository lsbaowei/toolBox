# utils_maps

`utils_maps` 当前提供基于 `sync.Map` 的本地缓存 `LocalSyncMap`，支持过期时间、定时清理和深拷贝读取，适合进程内短期缓存。

## 功能概览

- 并发安全的 key-value 存储。
- 每条数据支持过期时间。
- 后台 goroutine 每 10 分钟清理过期数据。
- `Set` 写入时深拷贝 value，减少外部引用影响。
- `SafeGet` 读取时返回深拷贝，避免调用方修改缓存内部对象。

## 安装与导入

```go
import "github.com/lsbaowei/toolBox/utils_maps"
```

## 基本用法

```go
var cache utils_maps.LocalSyncMap
cache.Init()
defer cache.Stop()

err := cache.Set("user:1", map[string]string{
    "name": "alice",
}, 60)
if err != nil {
    // handle error
}

v, ok := cache.SafeGet("user:1")
if ok {
    _ = v
}

deleted := cache.Del("user:1")
_ = deleted
```

## API

### `Init`

`Init()` 初始化缓存，并启动后台清理 goroutine。使用 `LocalSyncMap` 前应先调用。

### `Stop`

`Stop()` 停止后台清理 goroutine。该方法可安全多次调用，建议与 `Init` 成对使用。

### `Set`

`Set(key string, value interface{}, expireTime int64) error` 写入缓存。

- `expireTime` 单位为秒。
- 过期时间为 `time.Now().Unix() + expireTime`。
- 写入前会使用 `gob` 深拷贝 value。

### `Get`

`Get(key string) (interface{}, bool)` 获取缓存数据。

- 不进行深拷贝，性能更好。
- 如果数据已过期，会删除并返回 `false`。
- 返回值可能仍指向缓存内部对象，调用方不应修改。

### `SafeGet`

`SafeGet(key string) (interface{}, bool)` 获取缓存数据，并返回深拷贝后的值。

- 更适合调用方可能修改返回对象的场景。
- 深拷贝失败时返回 `false`。

### `Del`

`Del(key string) bool` 删除缓存数据，返回是否存在并删除成功。

## 深拷贝限制

当前深拷贝基于 `encoding/gob`：

- value 类型需要能被 `gob` 编码。
- 包含 channel、function、不可导出复杂字段的对象可能无法拷贝。
- 对性能敏感的大对象，应评估 `SafeGet` 和 `Set` 的拷贝成本。

## 注意事项

- `LocalSyncMap` 是进程内缓存，不适合多实例共享状态。
- 清理 goroutine 每 10 分钟执行一次；过期数据也会在 `Get` / `SafeGet` 时被惰性删除。
- `expireTime <= 0` 会让数据立即或很快过期。

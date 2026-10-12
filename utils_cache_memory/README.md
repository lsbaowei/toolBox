## 特殊内存缓存

`utils_cache_memory` 提供两类零值可用、并发安全、存储互不共享的进程内缓存：

- `Cache`：连续 `[]byte`，适合把 JSON 等字节直接写进 HTTP 响应。
- `StructCache[T]`：按结构体类型保存值，适合业务对象。定义见 `struct_cache.go`。

## 字节缓存

所有 `Cache` 实例共享一个全局 `sync.Map`，每个条目使用一个连续的 `[]byte` 保存：

- 前 8 字节：以 Big Endian 编码的 Unix 秒级绝对过期时间。
- 剩余字节：调用方写入的业务数据。

`Set` 会复制一次输入数据，避免调用方复用切片时修改缓存内容；`Get` 直接返回缓存条目的数据子切片，不进行额外复制。

## 使用示例

```go
package main

import (
	"fmt"
	"time"

	cachememory "github.com/lsbaowei/toolBox/utils_cache_memory"
)

func main() {
	var cache cachememory.Cache
	expireAt := time.Now().Add(time.Minute).Unix()

	if err := cache.Set("key", []byte("value"), expireAt); err != nil {
		panic(err)
	}

	gotExpireAt, value, ok := cache.Get("key")
	if ok {
		fmt.Printf("expireAt=%d value=%s\n", gotExpireAt, value)
	}
}
```

## 过期时间语义

- `expireAt > 0`：Unix 秒级绝对过期时间；当前时间大于或等于该值时视为过期。
- `expireAt == 0`：永不过期。
- `expireAt < 0`：`Set` 返回 `ErrInvalidExpiration`，原有条目不变。

`Get` 遇到过期条目时返回对应的过期时间和业务数据，同时令 `ok == false`，方便调用方判断或降级使用；该操作不会自动删除条目。缓存不会启动后台 goroutine，过期数据由调用方通过 `Delete` 或 `DeleteExpired(now)` 主动清理。过期只影响读取结果，条目仍占用内存。缓存没有条数或字节上限，持续写入新 key 又不定期清理时，进程内存会一直增长。

## API

- `Set`：写入业务数据和绝对过期时间。
- `Get`：读取过期时间、业务数据和命中状态。
- `Delete`：按 key 删除条目。
- `LoadAndDelete`：原子取出并删除有效条目。
- `Range`：遍历当前可观察到的有效条目，回调返回 `false` 时停止。
- `DeleteExpired`：按指定 Unix 秒删除过期条目并返回实际删除数量。

key 必须是 Go 的可比较类型，要求与 `sync.Map` 一致。`Range` 继承 `sync.Map.Range` 的弱一致性，不提供快照视图。

## 内存与并发约束

- 所有 `Cache` 实例共享数据，不提供实例级命名空间隔离。
- `DeleteExpired` 与并发写入采用弱一致性语义，清理过程中相同 key 的并发更新可能被删除；调用方应根据业务容忍度安排清理。
- 缓存仅存在于当前进程内，进程退出后数据不会保留。

## 零复制与只读约束

`Get`、`Range` 和 `LoadAndDelete` 返回的 `value` 是缓存条目去掉前 8 字节后的子切片，与缓存中的数据共享底层数组。读取过程不会复制业务数据，因此命中 `Get` 可以做到零额外内存分配。

调用方 **禁止修改返回的 `value`**。直接修改会改变缓存中的共享数据，并可能与其他并发读取产生 data race：

```go
_, value, ok := cache.Get("key")
if ok {
	value[0] = 'X' // 错误：修改了缓存底层数据
}
```

只读消费可以安全地并发执行，例如直接通过 Gin 返回响应：

```go
expireAt, value, ok := cache.Get("key")
if ok {
	c.Data(http.StatusOK, "application/json; charset=utf-8", value)
	return
}

// ok == false 时 value 仍可能包含已过期数据，可根据 expireAt 决定是否降级使用。
_ = expireAt
```

`Set` 会复制调用方传入的切片，所以 `Set` 返回后可以修改或复用原始切片。`Get` 返回后，即使相同 key 被 `Set` 覆盖或 `Delete` 删除，已经取得的 `value` 仍然有效，Go GC 会在所有引用释放后回收其底层数组。

如果调用方确实需要修改返回数据，必须先复制：

```go
_, value, ok := cache.Get("key")
if ok {
	mutable := append([]byte(nil), value...)
	mutable[0] = 'X' // 正确：仅修改副本
}
```

## 结构体缓存

`StructCache[T]` 与上面的 `Cache` 使用不同的 `sync.Map`。同一种 `T` 的全部实例共享一张表；`StructCache[User]` 和 `StructCache[Order]` 即使 key 相同也不会互相覆盖，也不会读写字节缓存。

过期时间语义与 `Cache` 相同：正数是 Unix 秒级绝对时间，`0` 表示永不过期，负数让 `Set` 返回 `ErrInvalidExpiration`。清理仍由调用方通过 `Delete` 或 `DeleteExpired(now)` 主动完成，没有后台 goroutine。过期条目仍占用内存；同一种 `T` 的 key 持续增加又不清理时，这张表会一直变大。

```go
package main

import (
	"fmt"
	"time"

	cachememory "github.com/lsbaowei/toolBox/utils_cache_memory"
)

type User struct {
	ID   int
	Name string
}

func main() {
	var users cachememory.StructCache[User]
	expireAt := time.Now().Add(time.Minute).Unix()
	if err := users.Set("user:1", User{ID: 1, Name: "alice"}, expireAt); err != nil {
		panic(err)
	}

	gotExpireAt, user, ok := users.Get("user:1")
	if ok {
		fmt.Printf("expireAt=%d id=%d\n", gotExpireAt, user.ID)
	}
}
```

`Set` 按值复制结构体，`Get` 返回的也是副本。修改副本中的值字段不会写回缓存。结构体里的切片、`map` 和指针仍与缓存共享底层数据，调用方不要修改这些引用指向的内容。
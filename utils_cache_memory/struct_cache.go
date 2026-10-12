package utils_cache_memory

import "sync"

// StructCache 按结构体类型保存进程内数据，与 Cache 的字节存储互不影响。
//
// 零值可用。同一种 T 的全部实例共享一张 sync.Map；不同 T 各自独立，
// 相同 key 不会互相覆盖。key 必须是可比较类型，要求与 sync.Map 相同。
//
// Set 按值复制 T。Get、Range 和 LoadAndDelete 返回的也是值副本，修改副本里的
// 值字段不会写回缓存。T 中的切片、map 和指针仍与缓存共享底层数据。
type StructCache[T any] struct{}

type structEntry[T any] struct {
	expireAt int64
	value    T
}

// typeKey 是某种 T 在注册表里的键。不同 T 的空结构体类型不同，因此不必调用 reflect.TypeOf。
type typeKey[T any] struct{}

// structRegistry 按 T 保存各自的 sync.Map，避免不同结构体类型共用一张表。
var structRegistry sync.Map

func structEntries[T any]() *sync.Map {
	key := typeKey[T]{}
	if raw, ok := structRegistry.Load(key); ok {
		return raw.(*sync.Map)
	}
	actual, _ := structRegistry.LoadOrStore(key, new(sync.Map))
	return actual.(*sync.Map)
}

// Set 保存 value 及其 Unix 秒级绝对过期时间 expireAt。
//
// expireAt 为 0 时条目永不过期，为负数时返回 ErrInvalidExpiration。
// value 按值复制；其中的切片、map 和指针仍与调用方共享底层数据。
func (StructCache[T]) Set(key any, value T, expireAt int64) error {
	if expireAt < 0 {
		return ErrInvalidExpiration
	}
	structEntries[T]().Store(key, structEntry[T]{expireAt: expireAt, value: value})
	return nil
}

// Get 返回 key 对应的过期时间、结构体副本和命中状态。
//
// 过期或不存在的条目返回 ok=false；过期时仍返回 expireAt 和 value 的副本，
// 且不会自动删除。
func (StructCache[T]) Get(key any) (expireAt int64, value T, ok bool) {
	entry, ok := loadStructEntry[T](key)
	if !ok {
		return 0, value, false
	}
	if isExpired(entry.expireAt, nowUnix()) {
		return entry.expireAt, entry.value, false
	}
	return entry.expireAt, entry.value, true
}

// Delete 删除当前 T 对应存储中的 key。
func (StructCache[T]) Delete(key any) {
	structEntries[T]().Delete(key)
}

// LoadAndDelete 原子地取出并删除 key 对应的有效条目。
//
// 返回的 value 是结构体副本。条目已过期时会被删除，并返回 ok=false。
func (StructCache[T]) LoadAndDelete(key any) (expireAt int64, value T, ok bool) {
	raw, ok := structEntries[T]().LoadAndDelete(key)
	if !ok {
		return 0, value, false
	}
	entry, ok := raw.(structEntry[T])
	if !ok || isExpired(entry.expireAt, nowUnix()) {
		return 0, value, false
	}
	return entry.expireAt, entry.value, true
}

// Range 依次调用 fn 访问当前 T 可观察到的有效条目。
//
// fn 返回 false 时停止遍历。Range 继承 sync.Map.Range 的弱一致性语义。
// 回调收到的 value 是结构体副本。
func (StructCache[T]) Range(fn func(key any, expireAt int64, value T) bool) {
	if fn == nil {
		return
	}

	now := nowUnix()
	structEntries[T]().Range(func(key, raw any) bool {
		entry, ok := raw.(structEntry[T])
		if !ok || isExpired(entry.expireAt, now) {
			return true
		}
		return fn(key, entry.expireAt, entry.value)
	})
}

// DeleteExpired 删除当前 T 的存储中，过期时间小于或等于 now 的条目，并返回发起删除的数量。
//
// 过期时间为 0 的条目不会被删除。该操作采用 sync.Map.Range 的弱一致性语义，
// 不保证保留清理期间相同 key 的并发更新。其他类型的 StructCache 和字节 Cache 不受影响。
func (StructCache[T]) DeleteExpired(now int64) int {
	deleted := 0
	entries := structEntries[T]()
	entries.Range(func(key, raw any) bool {
		entry, ok := raw.(structEntry[T])
		if ok && isExpired(entry.expireAt, now) {
			entries.Delete(key)
			deleted++
		}
		return true
	})
	return deleted
}

func loadStructEntry[T any](key any) (structEntry[T], bool) {
	raw, ok := structEntries[T]().Load(key)
	if !ok {
		return structEntry[T]{}, false
	}
	entry, ok := raw.(structEntry[T])
	return entry, ok
}

// Package utils_cache_memory 提供进程内全局共享的并发字节缓存。
package utils_cache_memory

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"
)

const expirationHeaderSize = 8

var (
	// ErrInvalidExpiration 表示过期时间不能为负数。
	ErrInvalidExpiration = errors.New("expiration must not be negative")

	entries sync.Map
	nowUnix = func() int64 {
		return time.Now().Unix()
	}
)

// Cache 是零值可用的内存缓存。
//
// 所有 Cache 实例共享同一个进程级存储。key 必须是可比较类型，要求与
// sync.Map 相同。
type Cache struct{}

// Set 保存 value 及其 Unix 秒级绝对过期时间 expireAt。
//
// expireAt 为 0 时条目永不过期，为负数时返回 ErrInvalidExpiration。
// Set 会复制 value，调用方可在返回后安全地修改或复用原切片。
func (Cache) Set(key any, value []byte, expireAt int64) error {
	if expireAt < 0 {
		return ErrInvalidExpiration
	}

	entry := encodeEntry(value, expireAt)
	entries.Store(key, entry)
	return nil
}

// Get 返回 key 对应的过期时间、业务数据和命中状态。
//
// 过期或不存在的条目返回 ok=false；过期时仍返回 expireAt 和 value，且不会
// 自动删除。返回的 value 与缓存条目共享底层数组，调用方必须将其视为只读；
// 如需修改，应先自行复制。
func (Cache) Get(key any) (expireAt int64, value []byte, ok bool) {
	raw, ok := entries.Load(key)
	if !ok {
		return 0, nil, false
	}

	entry, ok := raw.([]byte)
	if !ok {
		return 0, nil, false
	}
	expireAt, value, ok = decodeEntry(entry)
	if !ok {
		return 0, nil, false
	}
	if isExpired(expireAt, nowUnix()) {
		return expireAt, value, false
	}
	return expireAt, value, true
}

// Delete 删除 key 对应的条目。
func (Cache) Delete(key any) {
	entries.Delete(key)
}

// LoadAndDelete 原子地取出并删除 key 对应的有效条目。
//
// 返回的 value 与被删除条目共享底层数组，调用方必须将其视为只读。
func (Cache) LoadAndDelete(key any) (expireAt int64, value []byte, ok bool) {
	raw, ok := entries.LoadAndDelete(key)
	if !ok {
		return 0, nil, false
	}

	entry, ok := raw.([]byte)
	if !ok {
		return 0, nil, false
	}
	expireAt, value, ok = decodeEntry(entry)
	if !ok || isExpired(expireAt, nowUnix()) {
		return 0, nil, false
	}
	return expireAt, value, true
}

// Range 依次调用 fn 访问当前可观察到的有效条目。
//
// fn 返回 false 时停止遍历。Range 继承 sync.Map.Range 的弱一致性语义；
// value 与缓存条目共享底层数组，调用方必须将其视为只读。
func (Cache) Range(fn func(key any, expireAt int64, value []byte) bool) {
	if fn == nil {
		return
	}

	now := nowUnix()
	entries.Range(func(key, raw any) bool {
		entry, ok := raw.([]byte)
		if !ok {
			return true
		}
		expireAt, value, ok := decodeEntry(entry)
		if !ok || isExpired(expireAt, now) {
			return true
		}
		return fn(key, expireAt, value)
	})
}

// DeleteExpired 删除过期时间小于或等于 now 的条目，并返回发起删除的数量。
//
// 过期时间为 0 的条目不会被删除。该操作采用 sync.Map.Range 的弱一致性
// 语义，不保证保留清理期间相同 key 的并发更新。
func (Cache) DeleteExpired(now int64) int {
	deleted := 0
	entries.Range(func(key, raw any) bool {
		entry, ok := raw.([]byte)
		if !ok {
			return true
		}
		expireAt, _, ok := decodeEntry(entry)
		if ok && isExpired(expireAt, now) {
			entries.Delete(key)
			deleted++
		}
		return true
	})
	return deleted
}

func encodeEntry(value []byte, expireAt int64) []byte {
	entry := make([]byte, expirationHeaderSize+len(value))
	binary.BigEndian.PutUint64(entry[:expirationHeaderSize], uint64(expireAt))
	copy(entry[expirationHeaderSize:], value)
	return entry
}

func decodeEntry(entry []byte) (expireAt int64, value []byte, ok bool) {
	if len(entry) < expirationHeaderSize {
		return 0, nil, false
	}
	expireAt = int64(binary.BigEndian.Uint64(entry[:expirationHeaderSize]))
	return expireAt, entry[expirationHeaderSize:], true
}

func isExpired(expireAt, now int64) bool {
	return expireAt > 0 && expireAt <= now
}

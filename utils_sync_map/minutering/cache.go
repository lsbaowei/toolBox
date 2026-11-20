// Package minutering 实现「双 sync.Map + 60 槽环形时间分桶」缓存。
//
// 桶的时间跨度等于清理间隔。过期 key 放入「过期时间槽的下一格」，
// 清理指针无阻塞时也按同一跨度走格，扫到该桶时 key 已经过期。
package minutering

import (
	"fmt"
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// BucketCount 固定物理 Bucket 数量。
	BucketCount = 60
	// DefaultKeyLockShards 默认 KEY 分片锁数量。
	DefaultKeyLockShards = 256
)

var hashSeed = maphash.MakeSeed()

// Config 可选配置。零值字段在 New 时填入默认值。
type Config struct {
	KeyLockShards int
	// CleanupTick 清理推进间隔，也是每个桶覆盖的时间跨度。默认 1 分钟。
	CleanupTick time.Duration
	// SkipStartupSweep 为 true 时跳过启动全量扫描（仅测试用）。
	SkipStartupSweep bool
	// DisableCleanup 为 true 时不启动后台清理（仅测试用）。
	DisableCleanup bool
	// Now 可选时钟；测试可注入。
	Now func() int64
}

// Entry 是 m1 中保存的不可变业务条目。
type Entry struct {
	Value      any
	ExpireAt   int64 // UnixNano
	BucketSlot int
}

type bucket struct {
	items sync.Map // key -> int64(expireAt)
}

// MinuteRing 是 60 槽环形时间分桶缓存。桶跨度等于 CleanupTick。
type MinuteRing struct {
	m1 sync.Map // key -> *Entry

	buckets  [BucketCount]bucket
	keyLocks []sync.Mutex

	cleanupTick      time.Duration
	skipStartupSweep bool

	// nextSlot 是下一次要清理的桶编号，按顺序 0→1→…→59→0 推进。
	// 无阻塞时它和时间槽对齐；停顿后仍只清下一格，不跳到当前时间槽。
	nextSlot int

	stopCh    chan struct{}
	doneCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once

	nowFn atomic.Value // stores func() int64
}

// New 创建 MinuteRing 并启动每分钟推进一个 Bucket 的清理协程。
func New(cfg Config) *MinuteRing {
	if cfg.KeyLockShards <= 0 {
		cfg.KeyLockShards = DefaultKeyLockShards
	}
	if cfg.CleanupTick <= 0 {
		cfg.CleanupTick = time.Minute
	}
	now := cfg.Now
	if now == nil {
		now = func() int64 { return time.Now().UnixNano() }
	}

	m := &MinuteRing{
		keyLocks:         make([]sync.Mutex, cfg.KeyLockShards),
		cleanupTick:      cfg.CleanupTick,
		skipStartupSweep: cfg.SkipStartupSweep,
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
	}
	m.nowFn.Store(now)
	if !cfg.DisableCleanup {
		m.startOnce.Do(func() {
			go m.cleanupLoop()
		})
	} else {
		close(m.doneCh)
	}
	return m
}

// Close 停止后台清理协程，可安全多次调用。
func (m *MinuteRing) Close() {
	m.stopOnce.Do(func() {
		select {
		case <-m.doneCh:
		default:
			close(m.stopCh)
			<-m.doneCh
		}
	})
}

func (m *MinuteRing) nowNano() int64 {
	return m.nowFn.Load().(func() int64)()
}

func (m *MinuteRing) setNowFn(fn func() int64) {
	m.nowFn.Store(fn)
}

// Store 写入业务数据与过期索引。ttl <= 0 时等价于 Delete。
func (m *MinuteRing) Store(key, value any, ttl time.Duration) {
	if ttl <= 0 {
		m.Delete(key)
		return
	}

	lock := m.getKeyLock(key)
	lock.Lock()
	defer lock.Unlock()

	now := m.nowNano()
	expireAt := now + int64(ttl)

	// 桶跨度就是清理间隔。key 放进过期时间槽的下一格：
	// 指针对齐走到这一格时，ExpireAt 已经过去，这一拍就能删掉。
	current := m.slotID(now)
	expire := m.slotID(expireAt)
	due := expire + 1
	if due < current+1 {
		due = current + 1
	}
	slot := bucketSlot(due)

	entry := &Entry{
		Value:      value,
		ExpireAt:   expireAt,
		BucketSlot: slot,
	}

	m.m1.Store(key, entry)
	m.buckets[slot].items.Store(key, expireAt)
}

// Load 只访问 m1；异步过期语义下可能读到已过期数据。
func (m *MinuteRing) Load(key any) (any, bool) {
	raw, ok := m.m1.Load(key)
	if !ok {
		return nil, false
	}
	entry, ok := raw.(*Entry)
	if !ok || entry == nil {
		return nil, false
	}
	return entry.Value, true
}

// Delete 删除业务数据，并尝试清理当前 Entry 对应的索引。
func (m *MinuteRing) Delete(key any) {
	lock := m.getKeyLock(key)
	lock.Lock()
	defer lock.Unlock()

	raw, ok := m.m1.Load(key)
	if !ok {
		return
	}
	entry, ok := raw.(*Entry)
	if !ok || entry == nil {
		m.m1.Delete(key)
		return
	}

	compareAndDeleteMap(&m.m1, key, entry)
	compareAndDeleteMap(&m.buckets[entry.BucketSlot].items, key, entry.ExpireAt)
}

func (m *MinuteRing) getKeyLock(key any) *sync.Mutex {
	h := hashKey(key)
	return &m.keyLocks[h%uint64(len(m.keyLocks))]
}

// slotID 用清理间隔把 Unix 纳秒换成绝对时间槽编号。
func (m *MinuteRing) slotID(timestamp int64) int64 {
	return timestamp / int64(m.cleanupTick)
}

func bucketSlot(minute int64) int {
	slot := int(minute % BucketCount)
	if slot < 0 {
		slot += BucketCount
	}
	return slot
}

func hashKey(key any) uint64 {
	switch v := key.(type) {
	case string:
		var h maphash.Hash
		h.SetSeed(hashSeed)
		_, _ = h.WriteString(v)
		return h.Sum64()
	case int:
		return uint64(v)
	case int8:
		return uint64(v)
	case int16:
		return uint64(v)
	case int32:
		return uint64(v)
	case int64:
		return uint64(v)
	case uint:
		return uint64(v)
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return v
	default:
		var h maphash.Hash
		h.SetSeed(hashSeed)
		_, _ = fmt.Fprintf(&h, "%T:%v", key, key)
		return h.Sum64()
	}
}

func compareAndDeleteMap(sm *sync.Map, key, old any) bool {
	cur, ok := sm.Load(key)
	if !ok {
		return false
	}
	if !valueEqual(cur, old) {
		return false
	}
	sm.Delete(key)
	return true
}

func valueEqual(a, b any) bool {
	switch av := a.(type) {
	case *Entry:
		bv, ok := b.(*Entry)
		return ok && av == bv
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	default:
		return a == b
	}
}

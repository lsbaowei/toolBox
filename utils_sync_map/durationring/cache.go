// Package durationring 实现「固定 Bucket 数量 + 可配置时间粒度」的双 sync.Map 缓存。
//
// 设计要点（对应 README_task_1）：
//   - Bucket 数量默认 60，时间粒度（BucketDuration）可配置，默认 6 分钟；
//   - 索引保存完整绝对时间槽编号，不能仅用 % N 判断轮次；
//   - 后台按 CleanupInterval（默认 2 小时）扫描到期索引；
//   - Entry 不可变；业务 Load 只访问 m1。
package durationring

import (
	"fmt"
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultBucketCount 默认物理 Bucket 数量。
	DefaultBucketCount = 60
	// DefaultBucketDuration 默认单个 Bucket 覆盖的时间范围。
	DefaultBucketDuration = 6 * time.Minute
	// DefaultCleanupInterval 默认后台清理周期。
	DefaultCleanupInterval = 2 * time.Hour
	// DefaultKeyLockShards 默认 KEY 分片锁数量。
	DefaultKeyLockShards = 256
)

var hashSeed = maphash.MakeSeed()

// Config 为 DurationRing 的可调参数。零值字段在 New 时填入默认值。
type Config struct {
	BucketCount     int
	BucketDuration  time.Duration
	CleanupInterval time.Duration
	KeyLockShards   int
	// Now 可选时钟；测试可注入。
	Now func() int64
	// DisableCleanup 为 true 时不启动后台清理（仅测试用）。
	DisableCleanup bool
}

// Entry 是 m1 中保存的不可变业务条目。
type Entry struct {
	Value        any
	ExpireAt     int64 // UnixNano 绝对过期时间
	AbsoluteSlot int64 // expireAt / BucketDuration，完整时间槽编号
	BucketSlot   int   // AbsoluteSlot % BucketCount，物理槽位
}

// indexMeta 是 m2 Bucket 中保存的过期索引。
type indexMeta struct {
	expireAt     int64
	absoluteSlot int64
}

type bucket struct {
	items sync.Map // key -> indexMeta
}

// DurationRing 是可配置时间粒度的环形时间分桶缓存。
type DurationRing struct {
	m1 sync.Map // key -> *Entry

	buckets         []bucket
	keyLocks        []sync.Mutex
	bucketCount     int
	bucketDuration  time.Duration
	cleanupInterval time.Duration

	stopCh    chan struct{}
	doneCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once

	nowFn atomic.Value // stores func() int64
}

// New 按配置创建 DurationRing，并启动后台清理协程。
func New(cfg Config) *DurationRing {
	if cfg.BucketCount <= 0 {
		cfg.BucketCount = DefaultBucketCount
	}
	if cfg.BucketDuration <= 0 {
		cfg.BucketDuration = DefaultBucketDuration
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = DefaultCleanupInterval
	}
	if cfg.KeyLockShards <= 0 {
		cfg.KeyLockShards = DefaultKeyLockShards
	}
	now := cfg.Now
	if now == nil {
		now = func() int64 { return time.Now().UnixNano() }
	}

	d := &DurationRing{
		buckets:         make([]bucket, cfg.BucketCount),
		keyLocks:        make([]sync.Mutex, cfg.KeyLockShards),
		bucketCount:     cfg.BucketCount,
		bucketDuration:  cfg.BucketDuration,
		cleanupInterval: cfg.CleanupInterval,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}
	d.nowFn.Store(now)
	if !cfg.DisableCleanup {
		d.startOnce.Do(func() {
			go d.cleanupLoop()
		})
	} else {
		close(d.doneCh)
	}
	return d
}

// Close 停止后台清理协程，可安全多次调用。
func (d *DurationRing) Close() {
	d.stopOnce.Do(func() {
		select {
		case <-d.doneCh:
			// 未启动清理或已结束。
		default:
			close(d.stopCh)
			<-d.doneCh
		}
	})
}

func (d *DurationRing) nowNano() int64 {
	return d.nowFn.Load().(func() int64)()
}

// setNowFn 仅供测试注入时钟。
func (d *DurationRing) setNowFn(fn func() int64) {
	d.nowFn.Store(fn)
}

// Store 写入业务数据与过期索引。ttl <= 0 时等价于 Delete。
func (d *DurationRing) Store(key, value any, ttl time.Duration) {
	if ttl <= 0 {
		d.Delete(key)
		return
	}

	lock := d.getKeyLock(key)
	lock.Lock()
	defer lock.Unlock()

	now := d.nowNano()
	expireAt := now + int64(ttl)
	slotDur := int64(d.bucketDuration)

	currentSlot := now / slotDur
	expireSlot := expireAt / slotDur
	// 最早可检查槽至少是下一槽，避免落入正在/已扫描的当前槽而漏检。
	dueSlot := expireSlot
	if dueSlot < currentSlot+1 {
		dueSlot = currentSlot + 1
	}
	physical := int(dueSlot % int64(d.bucketCount))
	if physical < 0 {
		physical += d.bucketCount
	}

	entry := &Entry{
		Value:        value,
		ExpireAt:     expireAt,
		AbsoluteSlot: dueSlot,
		BucketSlot:   physical,
	}

	d.m1.Store(key, entry)
	d.buckets[physical].items.Store(key, indexMeta{
		expireAt:     expireAt,
		absoluteSlot: dueSlot,
	})
}

// Load 只访问 m1，不检查过期索引；异步过期语义下可能读到已过期数据。
func (d *DurationRing) Load(key any) (any, bool) {
	raw, ok := d.m1.Load(key)
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
func (d *DurationRing) Delete(key any) {
	lock := d.getKeyLock(key)
	lock.Lock()
	defer lock.Unlock()

	raw, ok := d.m1.Load(key)
	if !ok {
		return
	}
	entry, ok := raw.(*Entry)
	if !ok || entry == nil {
		d.m1.Delete(key)
		return
	}

	compareAndDeleteMap(&d.m1, key, entry)
	meta := indexMeta{expireAt: entry.ExpireAt, absoluteSlot: entry.AbsoluteSlot}
	compareAndDeleteMap(&d.buckets[entry.BucketSlot].items, key, meta)
}

func (d *DurationRing) getKeyLock(key any) *sync.Mutex {
	h := hashKey(key)
	return &d.keyLocks[h%uint64(len(d.keyLocks))]
}

func (d *DurationRing) absoluteSlot(ts int64) int64 {
	return ts / int64(d.bucketDuration)
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

// compareAndDeleteMap 在持有 KEY 分片锁的前提下，按值相等删除条目。
// 兼容 go1.19（无同步路径替代 sync.Map.CompareAndDelete）。
func compareAndDeleteMap(m *sync.Map, key, old any) bool {
	cur, ok := m.Load(key)
	if !ok {
		return false
	}
	if !valueEqual(cur, old) {
		return false
	}
	m.Delete(key)
	return true
}

func valueEqual(a, b any) bool {
	switch av := a.(type) {
	case *Entry:
		bv, ok := b.(*Entry)
		return ok && av == bv
	case indexMeta:
		bv, ok := b.(indexMeta)
		return ok && av == bv
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	default:
		return a == b
	}
}

package minutering

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestRing(t *testing.T) *MinuteRing {
	t.Helper()
	m := New(Config{
		KeyLockShards:    32,
		DisableCleanup:   true,
		SkipStartupSweep: true,
	})
	t.Cleanup(m.Close)
	return m
}

func TestMinuteRingStoreLoadDelete(t *testing.T) {
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	m.setNowFn(func() int64 { return base })

	m.Store("a", "v1", 10*time.Minute)
	got, ok := m.Load("a")
	if !ok || got != "v1" {
		t.Fatalf("Load got=%v ok=%v", got, ok)
	}

	m.Delete("a")
	if _, ok := m.Load("a"); ok {
		t.Fatal("expected missing after Delete")
	}
}

func TestMinuteRingTTLZeroDeletes(t *testing.T) {
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	m.setNowFn(func() int64 { return base })

	m.Store("a", 1, time.Minute)
	m.Store("a", 2, -1)
	if _, ok := m.Load("a"); ok {
		t.Fatal("ttl<=0 should delete")
	}
}

func TestMinuteRingExpireLandsInFollowingSlot(t *testing.T) {
	// 12:00:00 写入，59 分 30 秒后过期。过期时刻落在第 59 格，key 放进下一格（第 0 号桶）。
	// 对齐的指针在 13:00 扫到 0 号桶，此时已经过期。
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	m.setNowFn(func() int64 { return base })

	m.Store("k", "v", 59*time.Minute+30*time.Second)
	raw, ok := m.m1.Load("k")
	if !ok {
		t.Fatal("missing entry")
	}
	entry := raw.(*Entry)
	if entry.BucketSlot != 0 {
		t.Fatalf("BucketSlot=%d want 0", entry.BucketSlot)
	}

	// 59 分整扫到 0 号桶时，key 还要再过 30 秒才到期，不能删。
	var now atomic.Int64
	now.Store(base + int64(59*time.Minute))
	m.setNowFn(func() int64 { return now.Load() })
	m.cleanupBucket(0)
	if _, ok := m.Load("k"); !ok {
		t.Fatal("key is not expired yet")
	}

	// 对齐的下一拍是 13:00，扫 0 号桶时已经过期。
	now.Store(base + int64(60*time.Minute))
	m.cleanupBucket(0)
	if _, ok := m.Load("k"); ok {
		t.Fatal("expected key removed on the aligned following slot")
	}
}

func TestMinuteRingDueMinuteAvoidsCurrentSlot(t *testing.T) {
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 30, 0, time.UTC).UnixNano()
	m.setNowFn(func() int64 { return base })

	m.Store("soon", "v", 10*time.Second)
	raw, ok := m.m1.Load("soon")
	if !ok {
		t.Fatal("missing entry")
	}
	entry := raw.(*Entry)
	cur := m.slotID(base)
	wantSlot := bucketSlot(cur + 1)
	if entry.BucketSlot != wantSlot {
		t.Fatalf("BucketSlot=%d want %d", entry.BucketSlot, wantSlot)
	}
}

func TestMinuteRingStaleIndexDoesNotDeleteNewEntry(t *testing.T) {
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	m.setNowFn(func() int64 { return base })

	m.Store("k", "old", 5*time.Minute)
	raw1, _ := m.m1.Load("k")
	old := raw1.(*Entry)

	m.Store("k", "new", 65*time.Minute)
	got, ok := m.Load("k")
	if !ok || got != "new" {
		t.Fatalf("update failed: got=%v ok=%v", got, ok)
	}

	m.cleanupBucket(old.BucketSlot)
	got, ok = m.Load("k")
	if !ok || got != "new" {
		t.Fatalf("stale index cleanup deleted new data: got=%v ok=%v", got, ok)
	}
}

func TestMinuteRingCleanupExpires(t *testing.T) {
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	var now atomic.Int64
	now.Store(base)
	m.setNowFn(func() int64 { return now.Load() })

	m.Store("k", "v", 2*time.Minute)
	raw, _ := m.m1.Load("k")
	slot := raw.(*Entry).BucketSlot

	now.Store(base + int64(3*time.Minute))
	m.cleanupBucket(slot)
	if _, ok := m.Load("k"); ok {
		t.Fatal("expected expired key removed")
	}
}

func TestMinuteRingTTLOverFullCycle(t *testing.T) {
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	var now atomic.Int64
	now.Store(base)
	m.setNowFn(func() int64 { return now.Load() })

	m.Store("long", "v", 90*time.Minute)
	raw, _ := m.m1.Load("long")
	entry := raw.(*Entry)

	m.cleanupBucket(entry.BucketSlot)
	if _, ok := m.Load("long"); !ok {
		t.Fatal("long TTL should remain after early scan")
	}

	now.Store(base + int64(91*time.Minute))
	m.cleanupBucket(entry.BucketSlot)
	if _, ok := m.Load("long"); ok {
		t.Fatal("expected long TTL cleaned after expire")
	}
}

func TestMinuteRingSequentialAfterStall(t *testing.T) {
	// 清理按顺序推进：9 号桶清完，下一个清 10 号桶，
	// 不跳到当前分钟对应的桶。
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	m.setNowFn(func() int64 { return base })

	// 手动推进 9 个桶（0..8），nextSlot 应为 9。
	for i := 0; i < 9; i++ {
		m.cleanupBucket(m.nextSlot)
		m.nextSlot = (m.nextSlot + 1) % BucketCount
	}
	if m.nextSlot != 9 {
		t.Fatalf("nextSlot=%d want 9", m.nextSlot)
	}

	// 模拟停顿：时间跳到很久以后，但下一个清的仍然是 10 号桶。
	m.cleanupBucket(m.nextSlot) // 清 9 号
	m.nextSlot = (m.nextSlot + 1) % BucketCount
	if m.nextSlot != 10 {
		t.Fatalf("nextSlot=%d want 10", m.nextSlot)
	}
}

func TestMinuteRingNoBurstAfterStall(t *testing.T) {
	// 停顿很久后恢复，只清一个桶，过期 key 多等一轮自然回收。
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	var now atomic.Int64
	now.Store(base)
	m.setNowFn(func() int64 { return now.Load() })

	m.Store("k", "v", 2*time.Minute)
	raw, _ := m.m1.Load("k")
	slot := raw.(*Entry).BucketSlot

	// 时间跳到很久以后，模拟停顿后恢复。
	now.Store(base + int64(190*time.Minute))

	// 只清一个桶，不补扫。
	m.cleanupBucket(m.nextSlot)
	m.nextSlot = (m.nextSlot + 1) % BucketCount

	// 手动扫描 "k" 所在的桶确认能正确回收。
	now.Store(base + int64(3*time.Minute))
	m.cleanupBucket(slot)
	if _, ok := m.Load("k"); ok {
		t.Fatal("expected expired key cleaned when its bucket is scanned")
	}
}

func TestMinuteRingConcurrentStoreCleanup(t *testing.T) {
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	var now atomic.Int64
	now.Store(base)
	m.setNowFn(func() int64 { return now.Load() })

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				m.Store(i, n, time.Minute)
				_, _ = m.Load(i)
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < BucketCount; i++ {
			m.cleanupBucket(i)
		}
	}()

	wg.Wait()
	now.Store(base + int64(2*time.Minute))
	for i := 0; i < BucketCount; i++ {
		m.cleanupBucket(i)
	}
}

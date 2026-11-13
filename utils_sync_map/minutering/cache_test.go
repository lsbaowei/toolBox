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
	cur := minuteID(base)
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

func TestMinuteRingCleanupCatchUp(t *testing.T) {
	m := newTestRing(t)
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano()
	var now atomic.Int64
	now.Store(base)
	m.setNowFn(func() int64 { return now.Load() })

	m.mu.Lock()
	m.lastCleanedMinute = minuteID(base)
	m.mu.Unlock()

	m.Store("k", "v", 2*time.Minute)
	raw, _ := m.m1.Load("k")
	slot := raw.(*Entry).BucketSlot

	now.Store(base + int64(5*time.Minute))
	m.cleanupTickOnce()

	if _, ok := m.Load("k"); ok {
		m.cleanupBucket(slot)
		if _, ok := m.Load("k"); ok {
			t.Fatal("expected key cleaned after catch-up")
		}
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

package durationring

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestRing(t *testing.T, cfg Config) *DurationRing {
	t.Helper()
	cfg.DisableCleanup = true
	if cfg.BucketDuration == 0 {
		cfg.BucketDuration = time.Minute
	}
	if cfg.BucketCount == 0 {
		cfg.BucketCount = 60
	}
	d := New(cfg)
	t.Cleanup(d.Close)
	return d
}

func TestDurationRingStoreLoadDelete(t *testing.T) {
	d := newTestRing(t, Config{})
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	d.setNowFn(func() int64 { return base })

	d.Store("a", "v1", 10*time.Minute)
	got, ok := d.Load("a")
	if !ok || got != "v1" {
		t.Fatalf("Load got=%v ok=%v", got, ok)
	}

	d.Store("a", "v2", 10*time.Minute)
	got, ok = d.Load("a")
	if !ok || got != "v2" {
		t.Fatalf("after update Load got=%v ok=%v", got, ok)
	}

	d.Delete("a")
	if _, ok := d.Load("a"); ok {
		t.Fatal("expected missing after Delete")
	}
}

func TestDurationRingTTLZeroDeletes(t *testing.T) {
	d := newTestRing(t, Config{})
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	d.setNowFn(func() int64 { return base })

	d.Store("a", 1, time.Minute)
	d.Store("a", 2, 0)
	if _, ok := d.Load("a"); ok {
		t.Fatal("ttl<=0 should delete")
	}
}

func TestDurationRingAbsoluteSlotSurvivesPhysicalReuse(t *testing.T) {
	d := newTestRing(t, Config{
		BucketCount:    60,
		BucketDuration: time.Minute,
	})
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	var now atomic.Int64
	now.Store(base)
	d.setNowFn(func() int64 { return now.Load() })

	d.Store("far", "keep", 90*time.Minute)

	raw, _ := d.m1.Load("far")
	entry := raw.(*Entry)
	d.cleanupBucket(entry.BucketSlot)
	if _, ok := d.Load("far"); !ok {
		t.Fatal("long TTL must survive early physical bucket scan")
	}

	now.Store(base + int64(91*time.Minute))
	d.cleanupBucket(entry.BucketSlot)
	if _, ok := d.Load("far"); ok {
		t.Fatal("expected expired long TTL to be cleaned")
	}
}

func TestDurationRingStaleIndexDoesNotDeleteNewEntry(t *testing.T) {
	d := newTestRing(t, Config{
		BucketCount:    60,
		BucketDuration: time.Minute,
	})
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	d.setNowFn(func() int64 { return base })

	d.Store("k", "old", 5*time.Minute)
	raw1, _ := d.m1.Load("k")
	old := raw1.(*Entry)

	d.Store("k", "new", 65*time.Minute)

	d.cleanupBucket(old.BucketSlot)
	got, ok := d.Load("k")
	if !ok || got != "new" {
		t.Fatalf("stale cleanup corrupted data: got=%v ok=%v", got, ok)
	}
}

func TestDurationRingCleanupExpires(t *testing.T) {
	d := newTestRing(t, Config{
		BucketCount:    60,
		BucketDuration: time.Minute,
	})
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	var now atomic.Int64
	now.Store(base)
	d.setNowFn(func() int64 { return now.Load() })

	d.Store("k", "v", 2*time.Minute)
	raw, _ := d.m1.Load("k")
	slot := raw.(*Entry).BucketSlot

	now.Store(base + int64(3*time.Minute))
	d.cleanupBucket(slot)
	if _, ok := d.Load("k"); ok {
		t.Fatal("expected cleanup to remove expired key")
	}
}

func TestDurationRingConcurrentStoreLoad(t *testing.T) {
	d := newTestRing(t, Config{KeyLockShards: 32})
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	d.setNowFn(func() int64 { return base })

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := i % 8
			for n := 0; n < 200; n++ {
				d.Store(key, n, time.Minute)
				_, _ = d.Load(key)
				if n%17 == 0 {
					d.Delete(key)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestDurationRingDefaultConfig(t *testing.T) {
	d := New(Config{DisableCleanup: true})
	t.Cleanup(d.Close)
	if d.bucketCount != DefaultBucketCount {
		t.Fatalf("bucketCount=%d", d.bucketCount)
	}
	if d.bucketDuration != DefaultBucketDuration {
		t.Fatalf("bucketDuration=%v", d.bucketDuration)
	}
}

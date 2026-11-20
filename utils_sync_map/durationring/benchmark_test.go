package durationring

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// 预填充 key 数量。
const benchKeys = 10000

func fillDurationRing(d *DurationRing) {
	for i := 0; i < benchKeys; i++ {
		d.Store(fmt.Sprintf("key-%d", i), i, time.Hour)
	}
}

func fillSyncMap(m *sync.Map) {
	for i := 0; i < benchKeys; i++ {
		m.Store(fmt.Sprintf("key-%d", i), i)
	}
}

// ----------------------------------------------------------------
// 读取：单协程
// ----------------------------------------------------------------

func BenchmarkSyncMapLoad(b *testing.B) {
	var m sync.Map
	fillSyncMap(&m)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%benchKeys)
		v, ok := m.Load(key)
		if !ok {
			b.Fatal("miss")
		}
		_ = v
	}
}

func BenchmarkDurationRingLoad(b *testing.B) {
	d := New(Config{DisableCleanup: true})
	defer d.Close()
	fillDurationRing(d)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%benchKeys)
		v, ok := d.Load(key)
		if !ok {
			b.Fatal("miss")
		}
		_ = v
	}
}

// ----------------------------------------------------------------
// 读取：并发
// ----------------------------------------------------------------

func BenchmarkSyncMapLoadParallel(b *testing.B) {
	var m sync.Map
	fillSyncMap(&m)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			key := fmt.Sprintf("key-%d", i%benchKeys)
			v, ok := m.Load(key)
			if !ok {
				b.Fatal("miss")
			}
			_ = v
		}
	})
}

func BenchmarkDurationRingLoadParallel(b *testing.B) {
	d := New(Config{DisableCleanup: true})
	defer d.Close()
	fillDurationRing(d)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			key := fmt.Sprintf("key-%d", i%benchKeys)
			v, ok := d.Load(key)
			if !ok {
				b.Fatal("miss")
			}
			_ = v
		}
	})
}

// ----------------------------------------------------------------
// 读取：并发 + 后台清理同时运行（验证清理不拖慢 Load）
// ----------------------------------------------------------------

func BenchmarkDurationRingLoadWithCleanup(b *testing.B) {
	d := New(Config{
		DisableCleanup:   false,
		CleanupInterval:   50 * time.Millisecond,
		BucketDuration:   time.Minute,
	})
	defer d.Close()
	fillDurationRing(d)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			key := fmt.Sprintf("key-%d", i%benchKeys)
			v, ok := d.Load(key)
			if !ok {
				b.Fatal("miss")
			}
			_ = v
		}
	})
}

// ----------------------------------------------------------------
// 写入：单协程
// ----------------------------------------------------------------

func BenchmarkSyncMapStore(b *testing.B) {
	var m sync.Map
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store(fmt.Sprintf("key-%d", i%benchKeys), i)
	}
}

func BenchmarkDurationRingStore(b *testing.B) {
	d := New(Config{DisableCleanup: true})
	defer d.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Store(fmt.Sprintf("key-%d", i%benchKeys), i, time.Hour)
	}
}

// ----------------------------------------------------------------
// 写入：并发
// ----------------------------------------------------------------

func BenchmarkSyncMapStoreParallel(b *testing.B) {
	var m sync.Map
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			m.Store(fmt.Sprintf("key-%d", i%benchKeys), i)
		}
	})
}

func BenchmarkDurationRingStoreParallel(b *testing.B) {
	d := New(Config{DisableCleanup: true, KeyLockShards: 256})
	defer d.Close()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			d.Store(fmt.Sprintf("key-%d", i%benchKeys), i, time.Hour)
		}
	})
}

// ----------------------------------------------------------------
// 混合读写：80% Load + 20% Store
// ----------------------------------------------------------------

func BenchmarkSyncMapMixedParallel(b *testing.B) {
	var m sync.Map
	fillSyncMap(&m)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			if i%5 == 0 {
				m.Store(fmt.Sprintf("key-%d", i%benchKeys), i)
			} else {
				key := fmt.Sprintf("key-%d", i%benchKeys)
				v, ok := m.Load(key)
				if !ok {
					b.Fatal("miss")
				}
				_ = v
			}
		}
	})
}

func BenchmarkDurationRingMixedParallel(b *testing.B) {
	d := New(Config{DisableCleanup: true, KeyLockShards: 256})
	defer d.Close()
	fillDurationRing(d)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			if i%5 == 0 {
				d.Store(fmt.Sprintf("key-%d", i%benchKeys), i, time.Hour)
			} else {
				key := fmt.Sprintf("key-%d", i%benchKeys)
				v, ok := d.Load(key)
				if !ok {
					b.Fatal("miss")
				}
				_ = v
			}
		}
	})
}

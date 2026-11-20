package minutering

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

const benchKeys = 10000

func fillMinuteRing(m *MinuteRing) {
	for i := 0; i < benchKeys; i++ {
		m.Store(fmt.Sprintf("key-%d", i), i, time.Hour)
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

func BenchmarkMinuteRingLoad(b *testing.B) {
	m := New(Config{DisableCleanup: true})
	defer m.Close()
	fillMinuteRing(m)
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

func BenchmarkMinuteRingLoadParallel(b *testing.B) {
	m := New(Config{DisableCleanup: true})
	defer m.Close()
	fillMinuteRing(m)
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

// ----------------------------------------------------------------
// 读取：并发 + 后台清理同时运行
// ----------------------------------------------------------------

func BenchmarkMinuteRingLoadWithCleanup(b *testing.B) {
	m := New(Config{
		DisableCleanup: false,
		CleanupTick:    50 * time.Millisecond,
	})
	defer m.Close()
	fillMinuteRing(m)

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

// ----------------------------------------------------------------
// 写入
// ----------------------------------------------------------------

func BenchmarkSyncMapStore(b *testing.B) {
	var m sync.Map
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store(fmt.Sprintf("key-%d", i%benchKeys), i)
	}
}

func BenchmarkMinuteRingStore(b *testing.B) {
	m := New(Config{DisableCleanup: true})
	defer m.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store(fmt.Sprintf("key-%d", i%benchKeys), i, time.Hour)
	}
}

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

func BenchmarkMinuteRingStoreParallel(b *testing.B) {
	m := New(Config{DisableCleanup: true, KeyLockShards: 256})
	defer m.Close()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			m.Store(fmt.Sprintf("key-%d", i%benchKeys), i, time.Hour)
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

func BenchmarkMinuteRingMixedParallel(b *testing.B) {
	m := New(Config{DisableCleanup: true, KeyLockShards: 256})
	defer m.Close()
	fillMinuteRing(m)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			if i%5 == 0 {
				m.Store(fmt.Sprintf("key-%d", i%benchKeys), i, time.Hour)
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

package utils_cache_version

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

var benchmarkBusinessIDs = makeBenchmarkBusinessIDs(4096)

func BenchmarkManagerSelectVersionStableParallel(b *testing.B) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	m := New(Config{ReleaseDuration: 3 * time.Hour})
	if _, err := m.SelectVersion(SelectOptions{Version: "v1", Now: start}); err != nil {
		b.Fatal(err)
	}

	var seq uint64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := atomic.AddUint64(&seq, 1)
			_, err := m.SelectVersion(SelectOptions{
				BusinessID: benchmarkBusinessIDs[int(n)%len(benchmarkBusinessIDs)],
				Version:    "v1",
				Now:        start,
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkManagerSelectVersionActiveReleaseParallel(b *testing.B) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	m := New(Config{ReleaseDuration: 3 * time.Hour})
	if _, err := m.SelectVersion(SelectOptions{Version: "v1", Now: start}); err != nil {
		b.Fatal(err)
	}
	if _, err := m.SelectVersion(SelectOptions{BusinessID: "warmup", Version: "v2", Now: start}); err != nil {
		b.Fatal(err)
	}

	now := start.Add(90 * time.Minute)
	var seq uint64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := atomic.AddUint64(&seq, 1)
			_, err := m.SelectVersion(SelectOptions{
				BusinessID: benchmarkBusinessIDs[int(n)%len(benchmarkBusinessIDs)],
				Version:    "v2",
				Now:        now,
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkManagerSelectVersionTargetUpdateParallel(b *testing.B) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	m := New(Config{ReleaseDuration: 3 * time.Hour})
	if _, err := m.SelectVersion(SelectOptions{Version: "v1", Now: start}); err != nil {
		b.Fatal(err)
	}
	if _, err := m.SelectVersion(SelectOptions{BusinessID: "warmup", Version: "v2", Now: start}); err != nil {
		b.Fatal(err)
	}

	now := start.Add(90 * time.Minute)
	versions := []string{"v2", "v3", "v4", "v5"}
	var seq uint64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := atomic.AddUint64(&seq, 1)
			_, err := m.SelectVersion(SelectOptions{
				BusinessID: benchmarkBusinessIDs[int(n)%len(benchmarkBusinessIDs)],
				Version:    versions[int(n)%len(versions)],
				Now:        now,
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func makeBenchmarkBusinessIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("biz-%d", i)
	}
	return ids
}

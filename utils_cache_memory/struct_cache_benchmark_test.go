package utils_cache_memory

import "testing"

func BenchmarkStructCacheSet(b *testing.B) {
	resetStructEntries[structRecord]()
	b.Cleanup(func() { resetStructEntries[structRecord]() })

	var cache StructCache[structRecord]
	value := structRecord{ID: 1, Name: "benchmark"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cache.Set(i, value, 0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStructCacheGetHit(b *testing.B) {
	resetStructEntries[structRecord]()
	b.Cleanup(func() { resetStructEntries[structRecord]() })

	var cache StructCache[structRecord]
	if err := cache.Set("hit", structRecord{ID: 1, Name: "benchmark"}, 0); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, ok := cache.Get("hit"); !ok {
			b.Fatal("unexpected cache miss")
		}
	}
}

package utils_cache_memory

import "testing"

func BenchmarkCacheSet(b *testing.B) {
	resetEntries()
	b.Cleanup(resetEntries)

	var cache Cache
	value := []byte("benchmark-value")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cache.Set(i, value, 0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheGetHit(b *testing.B) {
	resetEntries()
	b.Cleanup(resetEntries)

	var cache Cache
	if err := cache.Set("hit", []byte("benchmark-value"), 0); err != nil {
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

func BenchmarkCacheGetExpired(b *testing.B) {
	originalNow := nowUnix
	nowUnix = func() int64 {
		return 100
	}
	b.Cleanup(func() {
		nowUnix = originalNow
		resetEntries()
	})

	var cache Cache
	entries.Store("expired", encodeEntry([]byte("benchmark-value"), 100))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, ok := cache.Get("expired"); ok {
			b.Fatal("unexpected cache hit")
		}
	}
}

package utils_cache_memory

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestCacheZeroValueAndGlobalSharing(t *testing.T) {
	resetEntries()
	t.Cleanup(resetEntries)
	setTestNow(t, 100)

	var first Cache
	var second Cache
	if err := first.Set("shared", []byte("value"), 200); err != nil {
		t.Fatal(err)
	}

	expireAt, value, ok := second.Get("shared")
	if !ok || expireAt != 200 || string(value) != "value" {
		t.Fatalf("Get() = (%d, %q, %v)", expireAt, value, ok)
	}
}

func TestSetEntryLayoutAndInputOwnership(t *testing.T) {
	resetEntries()
	t.Cleanup(resetEntries)
	setTestNow(t, 100)

	var cache Cache
	input := []byte("value")
	if err := cache.Set("layout", input, 200); err != nil {
		t.Fatal(err)
	}
	input[0] = 'X'

	raw, ok := entries.Load("layout")
	if !ok {
		t.Fatal("entry was not stored")
	}
	entry := raw.([]byte)
	if len(entry) != expirationHeaderSize+len("value") {
		t.Fatalf("entry length = %d", len(entry))
	}
	if got := int64(binary.BigEndian.Uint64(entry[:expirationHeaderSize])); got != 200 {
		t.Fatalf("encoded expiration = %d", got)
	}
	if got := string(entry[expirationHeaderSize:]); got != "value" {
		t.Fatalf("encoded value = %q", got)
	}

	_, value, ok := cache.Get("layout")
	if !ok {
		t.Fatal("Get() missed stored entry")
	}
	if len(value) > 0 && &value[0] != &entry[expirationHeaderSize] {
		t.Fatal("Get() copied the stored value")
	}
}

func TestSetEmptyValueAndInvalidExpiration(t *testing.T) {
	resetEntries()
	t.Cleanup(resetEntries)
	setTestNow(t, 100)

	var cache Cache
	if err := cache.Set("empty", nil, 200); err != nil {
		t.Fatal(err)
	}
	expireAt, value, ok := cache.Get("empty")
	if !ok || expireAt != 200 || len(value) != 0 {
		t.Fatalf("Get(empty) = (%d, %v, %v)", expireAt, value, ok)
	}

	if err := cache.Set("unchanged", []byte("old"), 200); err != nil {
		t.Fatal(err)
	}
	err := cache.Set("unchanged", []byte("new"), -1)
	if !errors.Is(err, ErrInvalidExpiration) {
		t.Fatalf("Set() error = %v", err)
	}
	_, value, ok = cache.Get("unchanged")
	if !ok || string(value) != "old" {
		t.Fatalf("invalid Set changed entry: value=%q ok=%v", value, ok)
	}
}

func TestGetExpirationSemantics(t *testing.T) {
	resetEntries()
	t.Cleanup(resetEntries)
	setTestNow(t, 100)

	var cache Cache
	mustSet(t, cache, "valid", "valid", 101)
	mustSet(t, cache, "boundary", "expired", 100)
	mustSet(t, cache, "forever", "forever", 0)

	if _, value, ok := cache.Get("valid"); !ok || string(value) != "valid" {
		t.Fatalf("valid Get() = (%q, %v)", value, ok)
	}
	if expireAt, value, ok := cache.Get("boundary"); ok || expireAt != 100 || string(value) != "expired" {
		t.Fatalf("expired Get() = (%d, %q, %v)", expireAt, value, ok)
	}
	if _, ok := entries.Load("boundary"); !ok {
		t.Fatal("Get() should not remove expired entry")
	}
	if expireAt, value, ok := cache.Get("forever"); !ok || expireAt != 0 || string(value) != "forever" {
		t.Fatalf("forever Get() = (%d, %q, %v)", expireAt, value, ok)
	}
	if _, _, ok := cache.Get("missing"); ok {
		t.Fatal("missing entry should not be found")
	}
}

func TestDeleteAndLoadAndDelete(t *testing.T) {
	resetEntries()
	t.Cleanup(resetEntries)
	setTestNow(t, 100)

	var cache Cache
	mustSet(t, cache, "delete", "value", 200)
	cache.Delete("delete")
	if _, _, ok := cache.Get("delete"); ok {
		t.Fatal("deleted entry should not be found")
	}

	mustSet(t, cache, "take", "value", 200)
	expireAt, value, ok := cache.LoadAndDelete("take")
	if !ok || expireAt != 200 || string(value) != "value" {
		t.Fatalf("LoadAndDelete() = (%d, %q, %v)", expireAt, value, ok)
	}
	if _, _, ok := cache.Get("take"); ok {
		t.Fatal("taken entry should not remain")
	}

	mustSet(t, cache, "expired", "value", 100)
	if _, _, ok := cache.LoadAndDelete("expired"); ok {
		t.Fatal("expired entry should not be returned")
	}
}

func TestRangeFiltersExpiredAndStops(t *testing.T) {
	resetEntries()
	t.Cleanup(resetEntries)
	setTestNow(t, 100)

	var cache Cache
	mustSet(t, cache, "valid-1", "one", 200)
	mustSet(t, cache, "valid-2", "two", 0)
	mustSet(t, cache, "expired", "old", 100)

	seen := make(map[any]string)
	cache.Range(func(key any, _ int64, value []byte) bool {
		seen[key] = string(value)
		return true
	})
	if len(seen) != 2 || seen["valid-1"] != "one" || seen["valid-2"] != "two" {
		t.Fatalf("Range() entries = %#v", seen)
	}
	if _, exists := seen["expired"]; exists {
		t.Fatal("Range() exposed expired entry")
	}

	calls := 0
	cache.Range(func(any, int64, []byte) bool {
		calls++
		return false
	})
	if calls != 1 {
		t.Fatalf("stopped Range() calls = %d", calls)
	}
}

func TestDeleteExpired(t *testing.T) {
	resetEntries()
	t.Cleanup(resetEntries)
	setTestNow(t, 100)

	var cache Cache
	mustSet(t, cache, "expired-1", "one", 99)
	mustSet(t, cache, "expired-2", "two", 100)
	mustSet(t, cache, "valid", "three", 101)
	mustSet(t, cache, "forever", "four", 0)

	if deleted := cache.DeleteExpired(100); deleted != 2 {
		t.Fatalf("DeleteExpired() = %d", deleted)
	}
	if _, _, ok := cache.Get("expired-1"); ok {
		t.Fatal("expired-1 should be deleted")
	}
	if _, _, ok := cache.Get("expired-2"); ok {
		t.Fatal("expired-2 should be deleted")
	}
	if _, _, ok := cache.Get("valid"); !ok {
		t.Fatal("valid entry should remain")
	}
	if _, _, ok := cache.Get("forever"); !ok {
		t.Fatal("forever entry should remain")
	}
}

func TestConcurrentAccess(t *testing.T) {
	resetEntries()
	t.Cleanup(resetEntries)
	setTestNow(t, 100)

	var cache Cache
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent-%d", i)
			if err := cache.Set(key, []byte("value"), 200); err != nil {
				t.Errorf("Set() error = %v", err)
				return
			}
			if _, value, ok := cache.Get(key); !ok || string(value) != "value" {
				t.Errorf("Get(%q) = (%q, %v)", key, value, ok)
			}
			cache.Delete(key)
		}(i)
	}
	wg.Wait()
}

func mustSet(t *testing.T, cache Cache, key, value string, expireAt int64) {
	t.Helper()
	if err := cache.Set(key, []byte(value), expireAt); err != nil {
		t.Fatal(err)
	}
}

func setTestNow(t *testing.T, now int64) {
	t.Helper()
	old := nowUnix
	nowUnix = func() int64 {
		return now
	}
	t.Cleanup(func() {
		nowUnix = old
	})
}

func resetEntries() {
	entries.Range(func(key, _ any) bool {
		entries.Delete(key)
		return true
	})
}

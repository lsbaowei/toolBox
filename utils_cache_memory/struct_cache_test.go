package utils_cache_memory

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

type structRecord struct {
	ID   int
	Name string
	Tags []string
}

type otherRecord struct {
	ID int
}

func TestStructCacheZeroValueAndTypeIsolation(t *testing.T) {
	resetStructEntries[structRecord]()
	resetStructEntries[otherRecord]()
	t.Cleanup(func() {
		resetStructEntries[structRecord]()
		resetStructEntries[otherRecord]()
	})
	setTestNow(t, 100)

	var first StructCache[structRecord]
	var second StructCache[structRecord]
	var other StructCache[otherRecord]
	if err := first.Set("shared", structRecord{ID: 1, Name: "alice"}, 200); err != nil {
		t.Fatal(err)
	}
	if err := other.Set("shared", otherRecord{ID: 2}, 200); err != nil {
		t.Fatal(err)
	}

	expireAt, value, ok := second.Get("shared")
	if !ok || expireAt != 200 || value.ID != 1 || value.Name != "alice" {
		t.Fatalf("Get() = (%d, %+v, %v)", expireAt, value, ok)
	}
	expireAt, otherValue, ok := other.Get("shared")
	if !ok || expireAt != 200 || otherValue.ID != 2 {
		t.Fatalf("other Get() = (%d, %+v, %v)", expireAt, otherValue, ok)
	}

	first.Delete("shared")
	if _, _, ok := second.Get("shared"); ok {
		t.Fatal("deleted struct entry should not remain")
	}
	if _, _, ok := other.Get("shared"); !ok {
		t.Fatal("deleting one struct type removed another type")
	}
}

func TestStructCacheSeparatedFromByteCache(t *testing.T) {
	resetEntries()
	resetStructEntries[structRecord]()
	t.Cleanup(func() {
		resetEntries()
		resetStructEntries[structRecord]()
	})
	setTestNow(t, 100)

	var bytes Cache
	var objects StructCache[structRecord]
	if err := bytes.Set("shared", []byte("json"), 200); err != nil {
		t.Fatal(err)
	}
	if err := objects.Set("shared", structRecord{ID: 7, Name: "obj"}, 200); err != nil {
		t.Fatal(err)
	}

	if _, value, ok := bytes.Get("shared"); !ok || string(value) != "json" {
		t.Fatalf("byte Get() = (%q, %v)", value, ok)
	}
	if _, value, ok := objects.Get("shared"); !ok || value.ID != 7 || value.Name != "obj" {
		t.Fatalf("struct Get() = (%+v, %v)", value, ok)
	}

	if err := bytes.Set("expired-bytes", []byte("keep"), 50); err != nil {
		t.Fatal(err)
	}
	bytes.Delete("shared")
	if _, _, ok := objects.Get("shared"); !ok {
		t.Fatal("deleting byte cache removed struct entry")
	}
	if deleted := objects.DeleteExpired(100); deleted != 0 {
		t.Fatalf("DeleteExpired() = %d", deleted)
	}
	if expireAt, value, ok := bytes.Get("expired-bytes"); ok || expireAt != 50 || string(value) != "keep" {
		t.Fatalf("struct cleanup changed byte cache: (%d, %q, %v)", expireAt, value, ok)
	}
}

func TestStructCacheCopiesValueFields(t *testing.T) {
	resetStructEntries[structRecord]()
	t.Cleanup(func() { resetStructEntries[structRecord]() })
	setTestNow(t, 100)

	var cache StructCache[structRecord]
	input := structRecord{ID: 1, Name: "alice", Tags: []string{"red"}}
	if err := cache.Set("user", input, 200); err != nil {
		t.Fatal(err)
	}
	input.ID = 9
	input.Name = "changed"
	input.Tags[0] = "blue"

	_, got, ok := cache.Get("user")
	if !ok || got.ID != 1 || got.Name != "alice" || len(got.Tags) != 1 || got.Tags[0] != "blue" {
		t.Fatalf("Get() = (%+v, %v)", got, ok)
	}

	got.ID = 3
	got.Name = "local"
	_, again, ok := cache.Get("user")
	if !ok || again.ID != 1 || again.Name != "alice" {
		t.Fatalf("returned struct shares value fields: %+v", again)
	}
	got.Tags[0] = "green"
	_, third, ok := cache.Get("user")
	if !ok || third.Tags[0] != "green" {
		t.Fatalf("reference field = %+v, ok=%v", third.Tags, ok)
	}
}

func TestStructCacheInvalidExpiration(t *testing.T) {
	resetStructEntries[structRecord]()
	t.Cleanup(func() { resetStructEntries[structRecord]() })
	setTestNow(t, 100)

	var cache StructCache[structRecord]
	if err := cache.Set("keep", structRecord{ID: 1}, 200); err != nil {
		t.Fatal(err)
	}
	err := cache.Set("keep", structRecord{ID: 2}, -1)
	if !errors.Is(err, ErrInvalidExpiration) {
		t.Fatalf("Set() error = %v", err)
	}
	if _, value, ok := cache.Get("keep"); !ok || value.ID != 1 {
		t.Fatalf("invalid Set changed entry: %+v ok=%v", value, ok)
	}
}

func TestStructCacheExpirationSemantics(t *testing.T) {
	resetStructEntries[structRecord]()
	t.Cleanup(func() { resetStructEntries[structRecord]() })
	setTestNow(t, 100)

	var cache StructCache[structRecord]
	mustSetStruct(t, cache, "valid", structRecord{ID: 1}, 101)
	mustSetStruct(t, cache, "boundary", structRecord{ID: 2}, 100)
	mustSetStruct(t, cache, "forever", structRecord{ID: 3}, 0)

	if _, value, ok := cache.Get("valid"); !ok || value.ID != 1 {
		t.Fatalf("valid Get() = (%+v, %v)", value, ok)
	}
	if expireAt, value, ok := cache.Get("boundary"); ok || expireAt != 100 || value.ID != 2 {
		t.Fatalf("expired Get() = (%d, %+v, %v)", expireAt, value, ok)
	}
	if _, ok := structEntries[structRecord]().Load("boundary"); !ok {
		t.Fatal("Get() should not remove expired entry")
	}
	if expireAt, value, ok := cache.Get("forever"); !ok || expireAt != 0 || value.ID != 3 {
		t.Fatalf("forever Get() = (%d, %+v, %v)", expireAt, value, ok)
	}
	if _, _, ok := cache.Get("missing"); ok {
		t.Fatal("missing entry should not be found")
	}
}

func TestStructCacheDeleteAndLoadAndDelete(t *testing.T) {
	resetStructEntries[structRecord]()
	t.Cleanup(func() { resetStructEntries[structRecord]() })
	setTestNow(t, 100)

	var cache StructCache[structRecord]
	mustSetStruct(t, cache, "delete", structRecord{ID: 1}, 200)
	cache.Delete("delete")
	if _, _, ok := cache.Get("delete"); ok {
		t.Fatal("deleted entry should not be found")
	}

	mustSetStruct(t, cache, "take", structRecord{ID: 2}, 200)
	expireAt, value, ok := cache.LoadAndDelete("take")
	if !ok || expireAt != 200 || value.ID != 2 {
		t.Fatalf("LoadAndDelete() = (%d, %+v, %v)", expireAt, value, ok)
	}
	if _, _, ok := cache.Get("take"); ok {
		t.Fatal("taken entry should not remain")
	}

	mustSetStruct(t, cache, "expired", structRecord{ID: 3}, 100)
	if _, _, ok := cache.LoadAndDelete("expired"); ok {
		t.Fatal("expired entry should not be returned")
	}
	if _, ok := structEntries[structRecord]().Load("expired"); ok {
		t.Fatal("expired LoadAndDelete should remove the entry")
	}
}

func TestStructCacheRangeFiltersExpiredAndStops(t *testing.T) {
	resetStructEntries[structRecord]()
	t.Cleanup(func() { resetStructEntries[structRecord]() })
	setTestNow(t, 100)

	var cache StructCache[structRecord]
	mustSetStruct(t, cache, "valid-1", structRecord{ID: 1}, 200)
	mustSetStruct(t, cache, "valid-2", structRecord{ID: 2}, 0)
	mustSetStruct(t, cache, "expired", structRecord{ID: 3}, 100)

	seen := make(map[any]int)
	cache.Range(func(key any, _ int64, value structRecord) bool {
		seen[key] = value.ID
		return true
	})
	if len(seen) != 2 || seen["valid-1"] != 1 || seen["valid-2"] != 2 {
		t.Fatalf("Range() entries = %#v", seen)
	}
	if _, exists := seen["expired"]; exists {
		t.Fatal("Range() exposed expired entry")
	}

	calls := 0
	cache.Range(func(any, int64, structRecord) bool {
		calls++
		return false
	})
	if calls != 1 {
		t.Fatalf("stopped Range() calls = %d", calls)
	}
	cache.Range(nil)
}

func TestStructCacheDeleteExpired(t *testing.T) {
	resetStructEntries[structRecord]()
	resetStructEntries[otherRecord]()
	t.Cleanup(func() {
		resetStructEntries[structRecord]()
		resetStructEntries[otherRecord]()
	})
	setTestNow(t, 100)

	var cache StructCache[structRecord]
	var other StructCache[otherRecord]
	mustSetStruct(t, cache, "expired-1", structRecord{ID: 1}, 99)
	mustSetStruct(t, cache, "expired-2", structRecord{ID: 2}, 100)
	mustSetStruct(t, cache, "valid", structRecord{ID: 3}, 101)
	mustSetStruct(t, cache, "forever", structRecord{ID: 4}, 0)
	mustSetStruct(t, other, "expired-1", otherRecord{ID: 9}, 99)

	if deleted := cache.DeleteExpired(100); deleted != 2 {
		t.Fatalf("DeleteExpired() = %d", deleted)
	}
	if _, _, ok := cache.Get("expired-1"); ok {
		t.Fatal("expired-1 should be deleted")
	}
	if _, _, ok := cache.Get("expired-2"); ok {
		t.Fatal("expired-2 should be deleted")
	}
	if _, value, ok := cache.Get("valid"); !ok || value.ID != 3 {
		t.Fatal("valid entry should remain")
	}
	if _, value, ok := cache.Get("forever"); !ok || value.ID != 4 {
		t.Fatal("forever entry should remain")
	}
	raw, ok := structEntries[otherRecord]().Load("expired-1")
	entry, isEntry := raw.(structEntry[otherRecord])
	if !ok || !isEntry || entry.value.ID != 9 {
		t.Fatal("DeleteExpired should not remove another struct type")
	}
}

func TestStructCacheConcurrentAccess(t *testing.T) {
	resetStructEntries[structRecord]()
	t.Cleanup(func() { resetStructEntries[structRecord]() })
	setTestNow(t, 100)

	var cache StructCache[structRecord]
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent-%d", i)
			if err := cache.Set(key, structRecord{ID: i}, 200); err != nil {
				t.Errorf("Set() error = %v", err)
				return
			}
			if _, value, ok := cache.Get(key); !ok || value.ID != i {
				t.Errorf("Get(%q) = (%+v, %v)", key, value, ok)
			}
			cache.Delete(key)
		}(i)
	}
	wg.Wait()
}

func mustSetStruct[T any](t *testing.T, cache StructCache[T], key any, value T, expireAt int64) {
	t.Helper()
	if err := cache.Set(key, value, expireAt); err != nil {
		t.Fatal(err)
	}
}

func resetStructEntries[T any]() {
	entries := structEntries[T]()
	entries.Range(func(key, _ any) bool {
		entries.Delete(key)
		return true
	})
}

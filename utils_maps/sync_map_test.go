package utils_maps

import "testing"

func TestLocalSyncMapSetGetDel(t *testing.T) {
	var m LocalSyncMap
	m.Init()
	defer m.Stop()

	if err := m.Set("k", "hello", 60); err != nil {
		t.Fatal(err)
	}

	v, ok := m.Get("k")
	if !ok || v != "hello" {
		t.Fatalf("got %v ok=%v", v, ok)
	}

	sv, ok := m.SafeGet("k")
	if !ok || sv != "hello" {
		t.Fatalf("SafeGet got %v ok=%v", sv, ok)
	}

	if !m.Del("k") {
		t.Fatal("Del should return true")
	}
	if _, ok := m.Get("k"); ok {
		t.Fatal("key should be gone")
	}
}

func TestLocalSyncMapExpire(t *testing.T) {
	var m LocalSyncMap
	m.Init()
	defer m.Stop()

	_ = m.Set("k", "v", -1)
	if _, ok := m.Get("k"); ok {
		t.Fatal("expected expired")
	}
}

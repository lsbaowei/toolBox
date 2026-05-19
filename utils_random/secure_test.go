package utils_random

import "testing"

func TestSecureIntn(t *testing.T) {
	if got, err := SecureIntn(0); err != nil || got != 0 {
		t.Fatalf("max 0: got %d err %v", got, err)
	}

	const max int64 = 50
	for i := 0; i < 10; i++ {
		got, err := SecureIntn(max)
		if err != nil {
			t.Fatal(err)
		}
		if got < 0 || got >= max {
			t.Fatalf("out of range: %d", got)
		}
	}
}

func TestSecureInt64(t *testing.T) {
	n, err := SecureInt64()
	if err != nil {
		t.Fatal(err)
	}
	if n < 0 {
		t.Fatalf("negative: %d", n)
	}
}

func TestIntWithSafetyDelegates(t *testing.T) {
	a, err := IntWithSafety()
	if err != nil {
		t.Fatal(err)
	}
	b, err := SecureInt64()
	if err != nil {
		t.Fatal(err)
	}
	if a < 0 || b < 0 {
		t.Fatal("expected non-negative")
	}
}

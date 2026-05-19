package utils_random

import "testing"

func TestIntWithSafety(t *testing.T) {
	n, err := IntWithSafety()
	if err != nil {
		t.Fatal(err)
	}
	if n < 0 {
		t.Fatalf("negative: %d", n)
	}
}

func TestRandUtilIntn(t *testing.T) {
	ru := NewWithSeed(1)
	if got := ru.Intn(0); got != 0 {
		t.Fatalf("Intn(0) = %d", got)
	}
	if got := ru.Intn(-1); got != 0 {
		t.Fatalf("Intn(-1) = %d", got)
	}
	got := ru.Intn(10)
	if got < 0 || got >= 10 {
		t.Fatalf("Intn(10) = %d", got)
	}
}

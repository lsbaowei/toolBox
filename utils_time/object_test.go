package utils_time

import (
	"testing"
	"time"
)

func mustTime(y int, m time.Month, d, hh, mm, ss int, nsec int, loc *time.Location) time.Time {
	return time.Date(y, m, d, hh, mm, ss, nsec, loc)
}

func TestNew(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	base := mustTime(2024, 6, 15, 14, 30, 0, 0, loc)

	d := New(&base)
	if !d.Time().Equal(base) {
		t.Fatalf("got %v want %v", d.Time(), base)
	}

	nilBase := New(nil)
	if nilBase == nil {
		t.Fatal("expected non-nil")
	}
	now := time.Now()
	if nilBase.Time().Sub(now) > time.Second || now.Sub(nilBase.Time()) > time.Second {
		t.Fatalf("New(nil) too far from now: %v", nilBase.Time())
	}
}

func TestFormatAndArithmetic(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	base := mustTime(2024, 6, 15, 14, 30, 45, 0, loc)
	d := New(&base)

	if got := d.FormatRFC3339(); got != base.Format(time.RFC3339) {
		t.Fatalf("FormatRFC3339: %s", got)
	}
	if got := d.FormatDate(); got != "2024-06-15" {
		t.Fatalf("FormatDate: %s", got)
	}
	if got := d.FormatDateTime(); got != "2024-06-15 14:30:45" {
		t.Fatalf("FormatDateTime: %s", got)
	}

	after := d.Add(2 * time.Hour)
	want := base.Add(2 * time.Hour)
	if !after.Time().Equal(want) {
		t.Fatalf("Add: got %v want %v", after.Time(), want)
	}

	nextMonth := d.AddDate(0, 1, 0)
	wantMonth := base.AddDate(0, 1, 0)
	if !nextMonth.Time().Equal(wantMonth) {
		t.Fatalf("AddDate: got %v want %v", nextMonth.Time(), wantMonth)
	}

	// 不可变
	if d.Time().Equal(after.Time()) {
		t.Fatal("receiver should be unchanged after Add")
	}
}

func TestDayBoundaries(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	base := mustTime(2024, 6, 15, 14, 30, 0, 0, loc)
	d := New(&base)

	start := d.StartOfDay().Time()
	wantStart := mustTime(2024, 6, 15, 0, 0, 0, 0, loc)
	if !start.Equal(wantStart) {
		t.Fatalf("StartOfDay: got %v want %v", start, wantStart)
	}

	end := d.EndOfDay().Time()
	wantEnd := mustTime(2024, 6, 15, 23, 59, 59, 999999999, loc)
	if !end.Equal(wantEnd) {
		t.Fatalf("EndOfDay: got %v want %v", end, wantEnd)
	}
}

func TestWeekBoundaries(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	// 2024-06-12 周三
	base := mustTime(2024, 6, 12, 10, 0, 0, 0, loc)
	d := New(&base)

	start := d.StartOfWeek().Time()
	wantStart := mustTime(2024, 6, 10, 0, 0, 0, 0, loc)
	if !start.Equal(wantStart) {
		t.Fatalf("StartOfWeek: got %v want %v", start, wantStart)
	}

	end := d.EndOfWeek().Time()
	wantEnd := mustTime(2024, 6, 16, 23, 59, 59, 0, loc)
	if !end.Equal(wantEnd) {
		t.Fatalf("EndOfWeek: got %v want %v", end, wantEnd)
	}
}

func TestMonthBoundaries(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	base := mustTime(2024, 6, 15, 12, 0, 0, 0, loc)
	d := New(&base)

	start := d.StartOfMonth().Time()
	wantStart := mustTime(2024, 6, 1, 0, 0, 0, 0, loc)
	if !start.Equal(wantStart) {
		t.Fatalf("StartOfMonth: got %v want %v", start, wantStart)
	}

	end := d.EndOfMonth().Time()
	wantEnd := mustTime(2024, 6, 30, 23, 59, 59, 0, loc)
	if !end.Equal(wantEnd) {
		t.Fatalf("EndOfMonth: got %v want %v", end, wantEnd)
	}

	// 2023-02 非闰年
	feb := New(ptr(mustTime(2023, 2, 10, 8, 0, 0, 0, loc)))
	febEnd := feb.EndOfMonth().Time()
	wantFebEnd := mustTime(2023, 2, 28, 23, 59, 59, 0, loc)
	if !febEnd.Equal(wantFebEnd) {
		t.Fatalf("EndOfMonth Feb: got %v want %v", febEnd, wantFebEnd)
	}
}

func ptr(t time.Time) *time.Time {
	return &t
}

func TestRemainingSeconds(t *testing.T) {
	loc := time.UTC
	base := mustTime(1970, 1, 1, 0, 33, 20, 0, loc) // Unix 2000
	other := mustTime(1970, 1, 1, 0, 25, 0, 0, loc) // Unix 1500
	d := New(&base)

	if got := d.RemainingSeconds(&other); got != 500 {
		t.Fatalf("got %d want 500", got)
	}

	later := mustTime(1970, 1, 1, 0, 33, 40, 0, loc) // Unix 2020
	if got := d.RemainingSeconds(&later); got != -20 {
		t.Fatalf("got %d want -20", got)
	}
}

func TestRandom(t *testing.T) {
	d := New(nil)

	if got := d.Random(0); got != 0 {
		t.Fatalf("max 0: got %d", got)
	}
	if got := d.Random(-1); got != 0 {
		t.Fatalf("max negative: got %d", got)
	}

	const max int64 = 100
	for i := 0; i < 20; i++ {
		got := d.Random(max)
		if got < 0 || got >= max {
			t.Fatalf("out of range: got %d max %d", got, max)
		}
	}
}

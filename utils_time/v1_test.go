package utils_time

import (
	"testing"
	"time"
)

func TestParseUnixTimestamp(t *testing.T) {
	sec := int64(1716883200)
	ms := int64(1716883200000)

	tests := []struct {
		name    string
		ts      int64
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			name: "seconds",
			ts:   sec,
			check: func(t *testing.T, got time.Time) {
				want := time.Unix(sec, 0)
				if !got.Equal(want) {
					t.Fatalf("got %v want %v", got, want)
				}
			},
		},
		{
			name: "milliseconds at threshold",
			ts:   secThreshold,
			check: func(t *testing.T, got time.Time) {
				want := time.UnixMilli(secThreshold)
				if !got.Equal(want) {
					t.Fatalf("got %v want %v", got, want)
				}
			},
		},
		{
			name: "seconds below threshold",
			ts:   secThreshold - 1,
			check: func(t *testing.T, got time.Time) {
				want := time.Unix(secThreshold-1, 0)
				if !got.Equal(want) {
					t.Fatalf("got %v want %v", got, want)
				}
			},
		},
		{
			name: "milliseconds below upper bound",
			ts:   msThreshold - 1,
			check: func(t *testing.T, got time.Time) {
				want := time.UnixMilli(msThreshold - 1)
				if !got.Equal(want) {
					t.Fatalf("got %v want %v", got, want)
				}
			},
		},
		{
			name:    "zero",
			ts:      0,
			wantErr: true,
		},
		{
			name:    "negative",
			ts:      -1,
			wantErr: true,
		},
		{
			name:    "at ms upper bound",
			ts:      msThreshold,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUnixTimestamp(tt.ts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, got)
		})
	}

	// 毫秒路径应得到与秒路径相同的瞬间
	gotSec, err := ParseUnixTimestamp(sec)
	if err != nil {
		t.Fatal(err)
	}
	gotMs, err := ParseUnixTimestamp(ms)
	if err != nil {
		t.Fatal(err)
	}
	if !gotSec.Equal(gotMs) {
		t.Fatalf("sec %v ms %v should be same instant", gotSec, gotMs)
	}
}

func TestParseTimeUTC(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "unix seconds",
			input: "1716883200",
			want:  time.Unix(1716883200, 0).UTC(),
		},
		{
			name:  "unix milliseconds",
			input: "1716883200000",
			want:  time.Unix(1716883200, 0).UTC(),
		},
		{
			name:  "date only UTC wall",
			input: "2024-05-28",
			want:  time.Date(2024, 5, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "datetime UTC wall",
			input: "2024-05-28 12:34:56",
			want:  time.Date(2024, 5, 28, 12, 34, 56, 0, time.UTC),
		},
		{
			name:  "RFC3339 Z",
			input: "2025-01-02T15:04:05Z",
			want:  time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name:  "RFC3339 offset",
			input: "2025-01-02T15:04:05+08:00",
			want:  time.Date(2025, 1, 2, 7, 4, 5, 0, time.UTC),
		},
		{
			name:    "invalid timestamp zero",
			input:   "0",
			wantErr: true,
		},
		{
			name:    "garbage",
			input:   "not-a-date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTimeUTC(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("got %s want %s", got.Format(time.RFC3339), tt.want.Format(time.RFC3339))
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)

	tests := []struct {
		name    string
		input   string
		loc     *time.Location
		want    time.Time
		wantErr bool
	}{
		{
			name:  "wall clock datetime",
			input: "2024-06-01 15:04:05",
			loc:   loc,
			want:  time.Date(2024, 6, 1, 15, 4, 5, 0, loc),
		},
		{
			name:  "wall clock date",
			input: "2024-06-01",
			loc:   loc,
			want:  time.Date(2024, 6, 1, 0, 0, 0, 0, loc),
		},
		{
			name:  "unix in zone",
			input: "1717481045",
			loc:   loc,
			want:  time.Unix(1717481045, 0).In(loc),
		},
		{
			name:  "unix ms in zone",
			input: "1717481045000",
			loc:   loc,
			want:  time.Unix(1717481045, 0).In(loc),
		},
		{
			name:  "RFC3339 keeps offset",
			input: "2025-01-02T15:04:05+08:00",
			loc:   loc,
			want:  time.Date(2025, 1, 2, 15, 4, 5, 0, time.FixedZone("", 8*3600)),
		},
		{
			name:    "invalid zero timestamp",
			input:   "0",
			loc:     loc,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTime(tt.input, tt.loc)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("got %s want %s", got.Format(time.RFC3339), tt.want.Format(time.RFC3339))
			}
		})
	}
}

func TestParseTime_nilDefaultTZ(t *testing.T) {
	got, err := ParseTime("2024-06-01", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2024, 6, 1, 0, 0, 0, 0, time.Local)
	if got.Location().String() != want.Location().String() {
		t.Fatalf("location %s want %s", got.Location(), want.Location())
	}
	if got.Year() != 2024 || got.Month() != 6 || got.Day() != 1 {
		t.Fatalf("unexpected date: %v", got)
	}
}

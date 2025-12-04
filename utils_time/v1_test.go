package utils_time

import (
	"testing"
	"time"
)

func TestParseTimeUTC(t *testing.T) {
	//	loc := time.Local

	tests := []struct {
		input string
	}{
		{"1716883200000"},                  // Unix 毫秒
		{"1716883200"},                     // Unix 秒
		{"2024-05-28"},                     // yyyy-mm-dd
		{"2024-05-28 12:34:56"},            // yyyy-mm-dd hh:mm:ss
		{"Tue, 28 May 2024 15:04:05 GMT"},  // RFC1123
		{"2025-01-02T15:04:05Z"},           // RFC3339
		{"2025-01-02T15:04:05+08:00"},      // 东八
		{"2025-01-02T15:04:05-05:00"},      // 西五
		{"Mon Jan 02 15:04:05 -0700 2006"}, // RubyDate
		{"01/02 03:04:05PM '06 -0700"},     // Custom layout
	}

	for _, tt := range tests {
		got, err := ParseTimeUTC(tt.input)
		if err != nil {
			t.Errorf("failed to parse [%q]: %v", tt.input, err)
		} else {
			t.Logf("Parsed [%s] as: %s ----- %d", tt.input, got.Format(time.RFC3339), got.Unix())
		}
	}
}

func TestParseTime(t *testing.T) {
	cases := []string{
		"2024-06-01 15:04:05",
		"2024-06-01",
		"Mon Jan 02 15:04:05 -0700 2006",
		"Mon Jan _2 15:04:05 2006",
		"2006-01-02T15:04:05Z07:00",
		"1717481045000", // ms timestamp
		"1717481045",    // s timestamp
	}

	for _, c := range cases {
		got, err := ParseTime(c, time.Local)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", c, err)
		} else {
			t.Logf("Parsed %s as %s", c, got.Format(time.RFC3339))
		}
	}
}

package utils_time

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var layouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02",
	time.RFC3339,
	time.RFC3339Nano,
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC850,
	time.RFC822,
	time.ANSIC,
	time.UnixDate,
	time.RubyDate,
	"02 Jan 06 15:04 MST",
	"02 Jan 06 15:04 -0700",
	"01/02 03:04:05PM '06 -0700",
}

// ParseTimeUTC parses a string into time.Time in UTC timezone.
func ParseTimeUTC(input string) (time.Time, error) {

	// 尝试解析为 Unix 时间戳（毫秒）
	if ts, err := strconv.ParseInt(input, 10, 64); err == nil {
		return parseUnixTimestampV1(ts), nil
	}

	// 尝试使用各种时间格式解析
	for _, layout := range layouts {
		if t, err := time.Parse(layout, input); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("unrecognized time format")
}

// ParseTime parses a string into time.Time with auto timezone handling
func ParseTime(str string, defaultTZ *time.Location) (time.Time, error) {
	str = strings.TrimSpace(str)

	// Try unix timestamp (milliseconds)
	if ts, err := strconv.ParseInt(str, 10, 64); err == nil {
		if ts > 1e12 {
			return time.UnixMilli(ts).In(defaultTZ), nil
		}
		return time.Unix(ts, 0).In(defaultTZ), nil
	}

	// Try known layouts
	for _, layout := range layouts {
		t, err := time.Parse(layout, str)
		if err == nil {
			// if layout doesn't have zone info, set default
			if t.Location().String() == "UTC" && !hasZoneInfo(layout) {
				return t.In(defaultTZ), nil
			}
			return t, nil
		}
	}
	return time.Time{}, errors.New("unsupported time format")
}

func getLoc(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Local
	}
	return loc
}

func hasZoneInfo(layout string) bool {
	return strings.Contains(layout, "Z") || strings.Contains(layout, "-0700") || strings.Contains(layout, "MST")
}

const (
	secThreshold = 10000000000    // ~2286年
	msThreshold  = 10000000000000 // ~2286年 in ms
)

// 安全时间范围
// 传入的实际时间unix时间戳，应该在 1970-04-27 02:45:36 ~ 2286-11-20 17:46:40 UTC 之间
// 计算得到的时间是符合实际的。
// 如果传入的时间 1970-04-27 02:45:36 之前，或 2286-11-20 17:46:40 UTC 之后，无法正确分辨是秒还是毫秒

// ParseUnixTimestamp 有安全时间
func ParseUnixTimestamp(ts int64) (time.Time, error) {

	switch {
	case ts > 0 && ts < secThreshold: // 时间在 1970-01-01 00:00:00 UTC 到 2286-11-20 17:46:40 UTC
		return time.Unix(ts, 0), nil
	case ts >= secThreshold && ts < msThreshold: // 时间在 1970-01-01 00:00:00 UTC 到 2286-11-20 17:46:40 UTC
		return time.UnixMilli(ts), nil
	default:
		return time.Time{}, fmt.Errorf("invalid timestamp: %d", ts)
	}
}

// parseUnixTimestampV1 简化版本，依照 ParseUnixTimestamp
func parseUnixTimestampV1(ts int64) time.Time {
	if ts > secThreshold {
		return time.UnixMilli(ts)
	}
	return time.Unix(ts, 0)
}

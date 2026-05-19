package utils_time

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// layouts 按优先级排列；前两项无时区，其余含 Z、-0700 或 MST 等时区占位符。
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

const (
	secThreshold = 10000000000    // ~2286年，秒/毫秒分界
	msThreshold  = 10000000000000 // ~2286年（毫秒上界）
)

// ParseUnixTimestamp 在安全区间内解析 Unix 时间戳（秒或毫秒）。
// 秒：0 < ts < secThreshold；毫秒：secThreshold <= ts < msThreshold；否则返回 error。
func ParseUnixTimestamp(ts int64) (time.Time, error) {
	switch {
	case ts > 0 && ts < secThreshold:
		return time.Unix(ts, 0), nil
	case ts >= secThreshold && ts < msThreshold:
		return time.UnixMilli(ts), nil
	default:
		return time.Time{}, fmt.Errorf("invalid timestamp: %d", ts)
	}
}

// ParseTimeUTC 将字符串解析为 time.Time。
// 纯数字字符串按 ParseUnixTimestamp 规则解析后转为 UTC；
// 日期布局无时区时墙钟分量按 UTC 解释；含 Z 或 offset 的布局保留原时区信息。
func ParseTimeUTC(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, errors.New("unrecognized time format")
	}

	if t, ok, err := parseUnixFromString(input); ok {
		if err != nil {
			return time.Time{}, err
		}
		return t.UTC(), nil
	}

	t, err := parseWithLayouts(input, nil, true)
	if err != nil {
		return time.Time{}, errors.New("unrecognized time format")
	}
	return t, nil
}

// ParseTime 将字符串解析为 time.Time。
// 纯数字字符串按 ParseUnixTimestamp 解析后置于 defaultTZ（nil 时使用 Local）；
// 无时区布局将墙钟时间置于 defaultTZ；含时区信息的布局保留字符串中的时区/偏移。
func ParseTime(str string, defaultTZ *time.Location) (time.Time, error) {
	str = strings.TrimSpace(str)
	if str == "" {
		return time.Time{}, errors.New("unsupported time format")
	}

	loc := defaultTZ
	if loc == nil {
		loc = time.Local
	}

	if t, ok, err := parseUnixFromString(str); ok {
		if err != nil {
			return time.Time{}, err
		}
		return t.In(loc), nil
	}

	t, err := parseWithLayouts(str, loc, false)
	if err != nil {
		return time.Time{}, errors.New("unsupported time format")
	}
	return t, nil
}

// parseUnixFromString 在整串为十进制整数时委托 ParseUnixTimestamp。
// 第二个返回值 ok 表示已按时间戳路径处理（含失败）。
func parseUnixFromString(s string) (time.Time, bool, error) {
	if !isDecimalIntString(s) {
		return time.Time{}, false, nil
	}
	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, false, nil
	}
	t, err := ParseUnixTimestamp(ts)
	if err != nil {
		return time.Time{}, true, err
	}
	return t, true, nil
}

func isDecimalIntString(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	if len(s) > 1 && s[0] == '0' {
		return false
	}
	return true
}

func parseWithLayouts(str string, loc *time.Location, utcMode bool) (time.Time, error) {
	for _, layout := range layouts {
		t, err := time.Parse(layout, str)
		if err != nil {
			continue
		}
		if hasZoneInfo(layout) {
			return t, nil
		}
		if utcMode {
			return t, nil
		}
		return wallClockIn(t, loc), nil
	}
	return time.Time{}, errors.New("no matching layout")
}

func wallClockIn(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

func getLoc(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Local
	}
	return loc
}

func hasZoneInfo(layout string) bool {
	return strings.Contains(layout, "Z") ||
		strings.Contains(layout, "-0700") ||
		strings.Contains(layout, "MST")
}

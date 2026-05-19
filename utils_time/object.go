package utils_time

import (
	"math/rand"
	"time"
)

// DateTime 封装不可变的基础时间，变换方法均返回新实例。
type DateTime struct {
	t   time.Time
	loc *time.Location
}

// New 创建 DateTime。base 为 nil 时使用当前时间。
func New(base *time.Time) *DateTime {
	var t time.Time
	if base == nil {
		t = time.Now()
	} else {
		t = *base
	}
	loc := t.Location()
	return &DateTime{t: t, loc: loc}
}

// Time 返回基础时间。
func (d *DateTime) Time() time.Time {
	return d.t
}

// Format 按 layout 格式化基础时间。
func (d *DateTime) Format(layout string) string {
	return d.t.Format(layout)
}

// FormatRFC3339 格式化为 RFC3339。
func (d *DateTime) FormatRFC3339() string {
	return d.t.Format(time.RFC3339)
}

// FormatDate 格式化为 2006-01-02。
func (d *DateTime) FormatDate() string {
	return d.t.Format("2006-01-02")
}

// FormatDateTime 格式化为 2006-01-02 15:04:05。
func (d *DateTime) FormatDateTime() string {
	return d.t.Format("2006-01-02 15:04:05")
}

// Add 在基础时间上增加时长，返回新对象。
func (d *DateTime) Add(delta time.Duration) *DateTime {
	return &DateTime{t: d.t.Add(delta), loc: d.loc}
}

// AddDate 在基础时间上增加年月日，返回新对象。
func (d *DateTime) AddDate(years, months, days int) *DateTime {
	return &DateTime{t: d.t.AddDate(years, months, days), loc: d.loc}
}

// RemainingSeconds 返回基础时间 Unix 秒减去参考时间 Unix 秒。
// other 为 nil 时使用调用时的当前时间；结果可为负数。
func (d *DateTime) RemainingSeconds(other *time.Time) int64 {
	if other == nil {
		return d.t.Unix() - time.Now().Unix()
	}
	return d.t.Unix() - other.Unix()
}

// Random 返回 [0, max) 的伪随机整数，以当前毫秒时间戳为种子因子。
// max <= 0 时返回 0。
func (d *DateTime) Random(max int64) int64 {
	if max <= 0 {
		return 0
	}
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))
	return r.Int63n(max)
}

// StartOfDay 返回当日 00:00:00。
func (d *DateTime) StartOfDay() *DateTime {
	return d.atClock(d.t.Year(), d.t.Month(), d.t.Day(), 0, 0, 0, 0)
}

// EndOfDay 返回当日 23:59:59.999999999。
func (d *DateTime) EndOfDay() *DateTime {
	return d.atClock(d.t.Year(), d.t.Month(), d.t.Day(), 23, 59, 59, 999999999)
}

// StartOfWeek 返回所在周周一 00:00:00。
func (d *DateTime) StartOfWeek() *DateTime {
	y, m, day := d.weekMonday()
	return d.atClock(y, m, day, 0, 0, 0, 0)
}

// EndOfWeek 返回所在周周日 23:59:59。
func (d *DateTime) EndOfWeek() *DateTime {
	mon := d.StartOfWeek()
	sun := mon.t.AddDate(0, 0, 6)
	return d.atClock(sun.Year(), sun.Month(), sun.Day(), 23, 59, 59, 0)
}

// StartOfMonth 返回当月 1 日 00:00:00。
func (d *DateTime) StartOfMonth() *DateTime {
	return d.atClock(d.t.Year(), d.t.Month(), 1, 0, 0, 0, 0)
}

// EndOfMonth 返回当月最后一日 23:59:59。
func (d *DateTime) EndOfMonth() *DateTime {
	y, m, _ := d.t.Date()
	last := time.Date(y, m+1, 0, 0, 0, 0, 0, d.loc)
	return d.atClock(last.Year(), last.Month(), last.Day(), 23, 59, 59, 0)
}

func (d *DateTime) atClock(year int, month time.Month, day, hour, min, sec, nsec int) *DateTime {
	return &DateTime{
		t:   time.Date(year, month, day, hour, min, sec, nsec, d.loc),
		loc: d.loc,
	}
}

func (d *DateTime) weekMonday() (year int, month time.Month, day int) {
	wd := d.t.Weekday()
	offset := (int(wd) + 6) % 7
	mon := d.t.AddDate(0, 0, -offset)
	return mon.Date()
}

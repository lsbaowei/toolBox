# utils_time

`utils_time` 提供时间字符串解析和不可变时间对象封装，适合处理接口入参、日志时间、Unix 时间戳和常见日期边界计算。

## 功能概览

- 解析 Unix 秒/毫秒时间戳。
- 解析常见日期时间字符串，包括 `2006-01-02`、`2006-01-02 15:04:05`、`2006-01-02T15:04:05`、`RFC3339`、`RFC3339Nano`、`RFC1123`、`RFC822` 等。
- 区分 UTC 解析和默认时区解析。
- 提供不可变 `DateTime` 包装，支持格式化、加减时间、日/周/月边界计算。
- 提供简单的剩余秒数计算和非安全随机数方法。

## 安装与导入

```go
import "github.com/lsbaowei/toolBox/utils_time"
```

## 时间解析

### `ParseUnixTimestamp`

`ParseUnixTimestamp(ts int64) (time.Time, error)` 在安全区间内解析 Unix 时间戳：

- `0 < ts < 1e10`：按 Unix 秒解析。
- `1e10 <= ts < 1e13`：按 Unix 毫秒解析。
- `0`、负数、超出上界的值会返回 error。

```go
t, err := utils_time.ParseUnixTimestamp(1716883200000)
if err != nil {
    // handle error
}
_ = t
```

### `ParseTimeUTC`

`ParseTimeUTC(input string) (time.Time, error)` 将输入解析为 `time.Time`：

- 纯数字字符串按 `ParseUnixTimestamp` 规则解析后转为 UTC。
- 无时区布局按 UTC 墙钟解释。
- 含 `Z`、offset 或 `MST` 的布局保留原字符串中的时区信息。

```go
t, err := utils_time.ParseTimeUTC("2026-05-18T15:29:04.527+08:00")
if err != nil {
    // handle error
}
_ = t
```

### `ParseTime`

`ParseTime(str string, defaultTZ *time.Location) (time.Time, error)` 适合业务希望按指定默认时区解释无时区字符串的场景：

- `defaultTZ == nil` 时使用 `time.Local`。
- 含时区信息的字符串保留原时区/偏移。
- Unix 时间戳解析后会转为 `defaultTZ`。

```go
loc, _ := time.LoadLocation("Asia/Shanghai")
t, err := utils_time.ParseTime("2026-06-16 12:00:00", loc)
if err != nil {
    // handle error
}
_ = t
```

## `DateTime`

`DateTime` 是不可变时间包装。`Add`、`AddDate`、`StartOfDay` 等方法都会返回新的 `*DateTime`，不会修改原对象。

```go
base, _ := utils_time.ParseTimeUTC("2026-06-16T12:00:00Z")
d := utils_time.New(&base)

start := d.StartOfDay()
end := d.EndOfMonth()

_ = start.FormatDateTime()
_ = end.FormatRFC3339()
```

常用方法：

- `New(base *time.Time) *DateTime`
- `Time() time.Time`
- `Format(layout string) string`
- `FormatRFC3339() string`
- `FormatDate() string`
- `FormatDateTime() string`
- `Add(delta time.Duration) *DateTime`
- `AddDate(years, months, days int) *DateTime`
- `StartOfDay() *DateTime`
- `EndOfDay() *DateTime`
- `StartOfWeek() *DateTime`
- `EndOfWeek() *DateTime`
- `StartOfMonth() *DateTime`
- `EndOfMonth() *DateTime`
- `RemainingSeconds(other *time.Time) int64`
- `Random(max int64) int64`

## 注意事项

- `ParseTimeUTC` 与 `ParseTime` 对无时区字符串的解释不同，接入时应明确业务希望使用 UTC 还是本地/指定时区。
- `DateTime.Random` 是伪随机，适合抽样、抖动等非安全场景；安全随机请使用 `utils_random.SecureIntn`。
- `StartOfWeek` / `EndOfWeek` 以周一作为一周开始，周日作为一周结束。

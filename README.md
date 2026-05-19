# Toolkit for golang
    简单好用，按照功能分文件夹（包）存放

## 任务1
    基本功能：`utils_time` 将时间字符串（Unix 秒/毫秒、常见 RFC/自定义布局）解析为 `time.Time`。

    **支持形态**
    - Unix：纯十进制整数；`0 < 秒 < 1e10`，`1e10 <= 毫秒 < 1e13`（与 `ParseUnixTimestamp` 一致）
    - 布局：`2006-01-02`、`2006-01-02 15:04:05`、RFC3339/1123/822、ANSIC、UnixDate、RubyDate 等（见 `utils_time/v1.go` 中 `layouts`）

    **入口**
    - `ParseTimeUTC`：无时区布局按 UTC 墙钟；含 offset 的布局保留原偏移
    - `ParseTime(str, defaultTZ)`：无时区布局按 `defaultTZ` 墙钟（`nil` 时用 `Local`）
    - `ParseUnixTimestamp`：仅解析整数时间戳

    **BREAKING（相对旧版）**
    - 时间戳 `0`、负数、超出安全区间、非纯数字串不再被当作 Unix 成功解析
    - `ParseTime` / `ParseTimeUTC` 的 Unix 路径统一走 `ParseUnixTimestamp`（不再使用 `1e12` 或无前缀零的宽松逻辑）

## 任务2：时间对象 `DateTime`

    不可变时间包装，方法返回新 `*DateTime`。

    **构造**
    - `New(nil)` → 当前时间
    - `New(&t)` → 指定基础时间

    **基础方法**
    - `Time()` → `time.Time`
    - `Format` / `FormatRFC3339` / `FormatDate` / `FormatDateTime`
    - `Add` / `AddDate`

    **边界（沿用基础时间的 Location）**
    - `StartOfDay` / `EndOfDay`（结束为 `23:59:59.999999999`）
    - `StartOfWeek` / `EndOfWeek`（周一至周日，结束 `23:59:59`）
    - `StartOfMonth` / `EndOfMonth`（结束 `23:59:59`）

    **示例**

    ```go
    loc, _ := time.LoadLocation("Asia/Shanghai")
    t, _ := ParseTime("2024-06-15 14:30:00", loc)
    d := New(&t)
    _ = d.StartOfDay().FormatDate()           // 2024-06-15
    _ = d.EndOfDay().FormatDateTime()         // 2024-06-15 23:59:59
    _ = d.StartOfWeek().AddDate(0, 0, 7)     // 链式：下周同一时刻
    ```

## 任务2.1：`DateTime` 辅助方法

    - `RemainingSeconds(other *time.Time) int64` — `基础时间.Unix() - other.Unix()`；`other == nil` 时用当前时间；可为负
    - `Random(max int64) int64` — 以当前毫秒时间戳为种子的伪随机数，`[0, max)`；`max <= 0` 返回 `0`

    ```go
    d := New(&expireAt)
    sec := d.RemainingSeconds(nil)  // 距现在剩余秒
    n := d.Random(100)              // [0, 100)
    ```

## 任务3：代码审查与加固（已完成）

    全库审查与修复见 [CHANGELOG.md](CHANGELOG.md)。摘要：修复 `JSONDecode`、随机数上界、`OnceFullWrite` panic、`ExecCmd` 上下文取消等；demo 迁至 `cmd/csvdemo`；补充多包单元测试。

## 任务4：优化
    基于结论：
        rand.New(rand.NewSource(time.Now().UnixMilli())) 明显优于 每次 rand.Seed；
        若追求性能和并发，应 复用 *rand.Rand，安全场景用 crypto/rand。

    优化：
        现有代码，复用 *rand.Rand，提高性能
        单独提供 安全场景用 crypto/rand，封装。




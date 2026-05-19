# Design — add-utils-time-object

## Context

`utils_time` 已完成字符串解析（`utils-time-parse` spec）。任务 2 需要链式/可复用的时间对象，方法均基于单一「基础时间」，且每次变换返回新实例（不修改原对象）。

## Goals / Non-Goals

**Goals:**

- 类型 `DateTime` 内部持有 `time.Time` 与显式 `*time.Location`（取自基础时间的 Location）。
- `New(base *time.Time) *DateTime`：`base == nil` → `time.Now()`。
- 边界计算在基础时间的 Location 下完成；周一起算（ISO 周一为一周开始）。
- `EndOfDay` 为当日 `23:59:59.999999999`（含全天最后一纳秒）；`EndOfWeek` / `EndOfMonth` 为对应最后一天的 `23:59:59`。

**Non-Goals:**

- 不实现时区转换、农历、工作日历、解析字符串（继续用 `ParseTime*`）。
- 不提供可变 API 或全局单例。
- 不强制与 `encoding/json` 集成（可后续单独 change）。

## Decisions

### D1: 类型命名 `DateTime`

**选择**：`DateTime` 结构体，避免包内名称 `Time` 与标准库 `time.Time` 混淆。

**备选**：`UTime` —— 可读性略差，不采用。

### D2: 不可变 + 返回新指针

**选择**：所有变换方法返回 `*DateTime`，内部 `time.Time` 值拷贝。

**理由**：与 README「返回新的对象」一致，利于链式调用。

### D3: `New` 签名

```go
func New(base *time.Time) *DateTime
```

仅一种可选基础时间；调用方 `New(nil)` 表示当前时刻。

### D4: 格式化

- `Format(layout string) string` — 委托 `time.Time.Format`
- 快捷方法（建议）：`FormatRFC3339()`、`FormatDate()`（`2006-01-02`）、`FormatDateTime()`（`2006-01-02 15:04:05`）

### D5: 边界算法

| 方法 | 结果墙钟（在 Location 下） |
|------|---------------------------|
| `StartOfDay` | `00:00:00`，年月日与基础时间相同 |
| `EndOfDay` | `23:59:59.999999999`，年月日与基础时间相同 |
| `StartOfWeek` | 所在周周一 `00:00:00` |
| `EndOfWeek` | 所在周周日 `23:59:59` |
| `StartOfMonth` | 当月 1 日 `00:00:00` |
| `EndOfMonth` | 当月最后一日 `23:59:59` |

周一计算：`offset := (int(weekday) + 6) % 7`（`time.Monday` = 0 偏移）。

月末：`time.Date(y, m+1, 0, ...)` 取最后一天。

`EndOfDay` 实现：`time.Date(y, m, d, 23, 59, 59, 999999999, loc)`。

### D6: `Add` / `AddDate`

委托 `time.Time.Add` / `AddDate`，包装为新 `DateTime`。

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| 周边界在 DST 切换日可能「墙钟异常」 | 使用 `time.Date` 在 Location 下构造，与标准库一致 |
| 与 `time.Time` 混用 | 提供 `Time()` 显式取出 |

## Migration Plan

1. 实现 `object.go` + 测试。
2. README 任务 2 补充用法示例。
3. `openspec validate` → `/opsx:apply`。

## Open Questions

（无）

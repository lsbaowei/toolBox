# extend-datetime-helpers

## Why

README 任务 2.1 需要在已有 `DateTime` 上补充两类常用能力：与另一时刻的秒级差值（倒计时/剩余时间）以及基于当前毫秒时间戳因子的伪随机数，避免业务侧重复实现。

## What Changes

- `DateTime` 新增 `RemainingSeconds(other *time.Time) int64`：`other == nil` 时用当前时间；返回 `基础时间.Unix() - other.Unix()`（可为负）。
- `DateTime` 新增 `Random(max int64) int64`：以 `time.Now().UnixMilli()` 为种子因子，返回 `[0, max)` 的伪随机整数；`max <= 0` 时返回 `0`。
- 补充单元测试与 README 任务 2.1 说明。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `utils-time-object`：增加剩余秒与随机数方法需求。

## Impact

- **代码**：`utils_time/object.go`、`object_test.go`
- **API**：纯新增方法，无 **BREAKING**
- **依赖**：标准库 `math/rand`、`time`

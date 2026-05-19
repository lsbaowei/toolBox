# add-utils-time-object

## Why

README 任务 2 要求在 `utils_time` 提供面向业务的「时间对象」，在已有解析能力之上封装常用读写、格式化与区间边界（日/周/月），避免调用方重复编写 `time.Date`、周一起算等样板代码。

## What Changes

- 新增不可变时间包装类型（工作名 `DateTime`）及 `New(*time.Time)` 工厂：未传或 `nil` 时使用当前时间。
- 基于「基础时间」提供：`Time()`、格式化（`Format` + 常用布局快捷方法）、`Add`、`AddDate`，均返回新对象。
- 提供日/周/月边界方法：当日开始/结束、当周（周一至周日）开始/结束、当月开始/结束；`EndOfDay` 为 `23:59:59.999999999`，周/月结束仍为 `23:59:59`（见 design）。
- 补充单元测试与 README 任务 2 说明。

## Capabilities

### New Capabilities

- `utils-time-object`: 可复用的时间对象，封装基础时间、格式化与日历边界计算。

### Modified Capabilities

（无：不改变 `utils-time-parse` 的解析需求）

## Impact

- **代码**：`utils_time/` 新增 `object.go`（或等价文件）及 `object_test.go`
- **API**：纯新增导出类型与方法，无 **BREAKING**
- **依赖**：标准库 `time`；与现有 `ParseTime*` 可组合使用（先解析再 `New(&t)`）

# Tasks — add-utils-time-object

## 1. 类型与构造

- [x] 1.1 新增 `DateTime` 结构体（持有 `time.Time`）及 `New(*time.Time) *DateTime`
- [x] 1.2 实现 `Time() time.Time`

## 2. 格式化与算术

- [x] 2.1 实现 `Format(layout string) string`
- [x] 2.2 实现快捷方法：`FormatRFC3339`、`FormatDate`、`FormatDateTime`（或 design 中确定的命名）
- [x] 2.3 实现 `Add(time.Duration) *DateTime` 与 `AddDate(years, months, days int) *DateTime`

## 3. 边界方法

- [x] 3.1 实现 `StartOfDay` / `EndOfDay`（`EndOfDay` 为 `23:59:59.999999999`）
- [x] 3.2 实现 `StartOfWeek` / `EndOfWeek`（周一至周日）
- [x] 3.3 实现 `StartOfMonth` / `EndOfMonth`

## 4. 测试与文档

- [x] 4.1 为构造、格式化、Add/AddDate 编写表驱动测试
- [x] 4.2 为日/周/月边界编写测试（含跨月、周三→周一、2 月月末）
- [x] 4.3 运行 `go test ./utils_time/...`
- [x] 4.4 更新 README 任务 2：类型名、示例代码、与 `New(nil)` 行为说明
- [x] 4.5 运行 `openspec validate add-utils-time-object`

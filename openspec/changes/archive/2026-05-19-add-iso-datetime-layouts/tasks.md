# Tasks — add-iso-datetime-layouts

## 1. 实现

- [x] 1.1 新增 `2006-01-02T15:04:05` 与 `entryISODateTime`
- [x] 1.2 实现 `hasTimeZoneSuffix` 与 `candidateLayouts` 扩展（19 字符 `T`、`T` 上限 40）
- [x] 1.3 确认 `init`/`allLayouts` 包含新 layout

## 2. 测试与文档

- [x] 2.1 `TestParseTimeUTC_task5Samples`（四种样例 + ParseTime 墙钟）
- [x] 2.2 更新 README 任务 5 为已完成
- [x] 2.3 `go test ./utils_time/...`；`openspec validate add-iso-datetime-layouts`

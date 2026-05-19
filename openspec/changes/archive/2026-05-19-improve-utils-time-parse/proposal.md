# improve-utils-time-parse

## Why

`utils_time` 是工具库中把时间字符串（时间戳、多种日期格式）解析为 `time.Time` 的核心能力；当前 `ParseTimeUTC`、`ParseTime` 与 `ParseUnixTimestamp` 三套逻辑不一致，存在秒/毫秒误判、命名与行为不符、边界时间戳无校验等问题。README 任务 1 要求做一次质量审查与优化，需在保持调用方简单可用的前提下统一行为并补齐测试。

## What Changes

- 统一 Unix 时间戳（秒/毫秒）判定逻辑，优先复用带安全区间的 `ParseUnixTimestamp`，避免 `ParseTimeUTC` 与 `ParseTime` 各自实现不同阈值。
- 明确并落实 `ParseTimeUTC` 的 UTC 语义（无时区信息的布局解析结果落在 UTC；带时区的布局保留原时区信息）。
- 修正 `ParseTime` 对「无时区布局 + defaultTZ」的处理，避免依赖 `Location().String() == "UTC"` 等脆弱判断。
- 补充/调整单元测试：覆盖边界时间戳、非法输入、秒/毫秒分界、主要布局与回归用例。
- 文档化支持的输入形态与已知限制（如超出安全区间的超大时间戳）。

## Capabilities

### New Capabilities

- `utils-time-parse`: 将多种时间字符串（Unix 秒/毫秒、常见 RFC/自定义布局）解析为 `time.Time`，支持 UTC 专用与带默认时区两种入口。

### Modified Capabilities

（无既有 `openspec/specs/` 基线）

## Impact

- **代码**：`utils_time/v1.go`、`utils_time/v1_test.go`
- **API**：`ParseTimeUTC`、`ParseTime`、`ParseUnixTimestamp` 行为可能对极端时间戳输入更严格（原先静默错误解析的用例可能返回 error）——需在 design 中界定是否 **BREAKING**
- **依赖**：仅标准库 `time`、`strconv`、`strings`
- **调用方**：本仓库内其它包若依赖旧的不一致行为，需随测试一并验证

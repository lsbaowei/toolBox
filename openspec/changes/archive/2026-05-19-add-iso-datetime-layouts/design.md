# Design — add-iso-datetime-layouts

## Context

任务 5 在已有 `layoutEntry` 快路径 + 全量 fallback 基础上，仅需补齐无时区 `T` 格式并微调快路径边界。

## Decisions

### D1: 新 layout

`2006-01-02T15:04:05`，`hasZone: false`，插入 `layoutStrings` 于 `RFC3339` 之前。

### D2: 快路径

- `n==19 && s[10]=='T' && !hasTimeZoneSuffix(s)` → 仅 `entryISODateTime`
- `hasTimeZoneSuffix`：`Z` 后缀或 `+`/`-` 出现在第 10 字符之后
- `T` 分支：`n` 从 20～**40**（原 35），含 `.` 时先 `RFC3339Nano`

### D3: 语义

- `ParseTimeUTC`：`ParseInLocation(..., UTC)` 墙钟
- `ParseTime`：`ParseInLocation(..., defaultTZ)` 墙钟

### D4: 测试

`TestParseTimeUTC_task5Samples` 覆盖 README 四种样例 + `ParseTime` 墙钟一例。

## Non-Goals

- 不新增除任务 5 列表外的格式。

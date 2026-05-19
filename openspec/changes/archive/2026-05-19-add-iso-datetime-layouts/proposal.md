# add-iso-datetime-layouts

## Why

README 任务 5 要求 `parseWithLayouts` 明确支持常见 ISO 类时间字符串。当前实测：

| 样例 | 当前 |
|------|------|
| `2025-11-19T11:33:19.920584349+08:00` | 可解析（`RFC3339Nano`） |
| `2026-05-18T15:29:04.527+08:00` | 可解析（`RFC3339Nano`） |
| `2025-01-02T15:04:05+08:00` | 可解析（`RFC3339`） |
| `2025-01-02T15:04:05` | **失败**（缺无时区 `T` 布局） |

本 change 补齐缺失项，并为 `T` 形态快路径放宽长度上限（35→40），减少长纳秒串绕路。

## What Changes

- 新增 layout `2006-01-02T15:04:05`（无时区）及快路径 `len==19` + `T` 分流。
- `candidateLayouts` 中 `T` 形态上限调整为 40。
- 四种 README 样例单元测试；更新 README 任务 5。

## Capabilities

### Modified Capabilities

- `utils-time-parse`：扩展 ISO 日期时间解析能力。

## Impact

- **代码**：`utils_time/v1.go`、`v1_test.go`、`README.md`
- **BREAKING**：无

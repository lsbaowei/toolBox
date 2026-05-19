# Design — improve-utils-time-parse

## Context

`utils_time/v1.go` 提供三个相关入口：

| 函数 | 用途 |
|------|------|
| `ParseUnixTimestamp` | 带 `secThreshold` / `msThreshold` 安全区间的秒/毫秒解析 |
| `ParseTimeUTC` | 文档称 UTC；整数走 `parseUnixTimestampV1`（无校验），字符串走 `layouts` + `time.Parse` |
| `ParseTime` | 整数用 `1e12` 分界；布局解析后对无时区结果 `.In(defaultTZ)` |

现有问题摘要：

1. **三套阈值**：`1e12`、`secThreshold`（1e10）、`msThreshold`（1e13）不一致，同一数字在不同函数下可能得到不同时刻。
2. **`parseUnixTimestampV1` 无错误返回**：任意可解析整数都成功，包括 0、负数、超大值。
3. **`ParseTimeUTC` 命名**：布局解析用 `time.Parse` 得到 UTC 是符合 Go 语义的；但应对调用方说明「带 `Z`/`±offset` 的布局保留偏移」。
4. **`ParseTime` 时区逻辑**：`t.Location().String() == "UTC" && !hasZoneInfo(layout)` 在部分布局下不可靠；应改为根据 layout 是否含时区占位符决定是否在 `defaultTZ` 下解释「墙钟时间」。
5. **测试**：`TestParseTime` 含 `"Mon Jan _2 15:04:05 2006"`（非法 reference time），应删除或修正；缺少表驱动断言与边界用例。

## Goals / Non-Goals

**Goals:**

- 单一内部函数处理 `int64` 时间戳（委托 `ParseUnixTimestamp`），`ParseTimeUTC` / `ParseTime` 共用。
- 行为与 README 任务一致：常见字符串形态可解析；秒/毫秒在安全区间内判定正确。
- 测试可回归：表驱动 + 期望 `time.Time` 或 RFC3339 字符串对比。

**Non-Goals:**

- 不支持浮点 Unix、纳秒时间戳、自然语言日期。
- 不新增大量小众 locale 格式（除非现有测试已覆盖且易维护）。
- 不改变包名或导出函数签名（除非发现必须 **BREAKING** 的 bug，见 Risks）。

## Decisions

### D1: Unix 解析统一走 `ParseUnixTimestamp`

**选择**：`ParseTimeUTC` 与 `ParseTime` 在识别为纯整数字符串后，调用 `ParseUnixTimestamp`；UTC 入口对结果 `.UTC()`，带时区入口对结果 `.In(defaultTZ)`。

**理由**：单一真相源，与注释中的「安全时间范围」一致。

**备选**：保留 `parseUnixTimestampV1` 仅用于 UTC —— 拒绝，会继续产生分歧。

### D2: 纯数字字符串的识别

**选择**：`strings.TrimSpace` 后，若 `strconv.ParseInt` 成功且整串无多余字符，视为时间戳；否则走布局列表。

**理由**：与现行为一致，避免 `"2024-05-28"` 被误判。

### D3: 无时区布局的时区语义

**选择**：

- `ParseTimeUTC`：对不含时区占位符的 layout，`time.Parse` 后在 UTC 下保持墙钟分量（Go 默认即为 UTC location）。
- `ParseTime`：对不含时区占位符的 layout，解析后将年月日时分秒视为 `defaultTZ` 的墙钟时间（`time.Date` 重建或 `t.In(defaultTZ)` 且文档说明不用于已含 offset 的字符串）。

**理由**：符合「UTC 解析」与「本地/业务时区默认」的常见预期。

**实现提示**：复用/增强 `hasZoneInfo(layout)`，或维护「无时区 layout」白名单（当前 `layouts` 前两项）。

### D4: 破坏性变更范围

**选择**：对超出 `ParseUnixTimestamp` 安全区间或非法的时间戳，**返回 error**（不再用 `parseUnixTimestampV1` 静默成功）。在 proposal 中标记为潜在 **BREAKING**，若仓库无外部消费者可接受。

**备选**：保留宽松行为并仅加注释 —— 拒绝，与「安全时间」设计目标冲突。

### D5: 测试策略

**选择**：表驱动 `tests []struct{ in string; want time.Time; wantErr bool }`；对 UTC 与 Local 各一组；增加 `ParseUnixTimestamp` 边界表（`secThreshold-1`, `secThreshold`, `msThreshold-1`, 非法值）。

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| 原先依赖超大/负数时间戳的调用方失败 | 在 README 或包注释写明支持区间；变更说明 **BREAKING** |
| `ParseTime` 时区行为微调导致墙钟偏移 | 用固定 `time.FixedZone` 的测试用例锁定行为 |
| `layouts` 顺序影响歧义字符串 | 保持现有顺序，仅在有冲突时加注释；新格式追加到末尾 |

## Migration Plan

1. 实现内部 `parseUnixFromString` → `ParseUnixTimestamp`。
2. 调整 `ParseTimeUTC` / `ParseTime`，删除 `parseUnixTimestampV1`（或改为薄包装并标记 unexported 测试）。
3. 更新/扩充 `v1_test.go`，`go test ./utils_time/...`。
4. `openspec validate improve-utils-time-parse` 通过后，使用 `/opsx:apply` 或手工按 `tasks.md` 实施。

## Open Questions

- 仓库外是否有服务依赖 `ParseTimeUTC` 对 `0` 或超大整数仍返回成功？若无，可放心收紧。
- `ParseTime` 的 `defaultTZ == nil` 是否应 panic 或默认 `time.Local`？（当前未校验，可在 tasks 中明确。）

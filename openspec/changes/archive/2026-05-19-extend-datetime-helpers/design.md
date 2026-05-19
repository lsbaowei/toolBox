# Design — extend-datetime-helpers

## Context

`DateTime` 已提供构造、格式化、算术与日历边界。任务 2.1 增加两个与「当前/对比时刻」相关的辅助方法，均不修改接收者状态。

## Goals / Non-Goals

**Goals:**

- `RemainingSeconds` 语义清晰、可测；`nil` 表示与「现在」比较。
- `Random` 使用进程内 `math/rand`，种子来自调用时的 `UnixMilli()`，满足「毫秒因子」描述。
- 行为在 README 与 spec 中写清边界（`max <= 0`、负数剩余秒）。

**Non-Goals:**

- 不提供密码学安全随机（不用 `crypto/rand`）。
- 不将随机状态持久化到 `DateTime` 实例（每次 `Random` 独立播种）。
- 不改变既有边界/格式化方法。

## Decisions

### D1: `RemainingSeconds(other *time.Time) int64`

```go
func (d *DateTime) RemainingSeconds(other *time.Time) int64 {
    ref := time.Now()
    if other != nil {
        ref = *other
    }
    return d.t.Unix() - ref.Unix()
}
```

- 使用秒级 `Unix()`，与 README「unix秒」一致。
- 若 `other` 晚于基础时间，结果为负数（不截断为 0）。

### D2: `Random(max int64) int64`

```go
func (d *DateTime) Random(max int64) int64 {
    if max <= 0 {
        return 0
    }
    r := rand.New(rand.NewSource(time.Now().UnixMilli()))
    return r.Int63n(max)
}
```

- 接收者 `d` 不参与随机计算（README 仅要求当前毫秒因子）；保留为方法以便 `d.Random(n)` 链式风格。
- `max` 上限使用 `Int63n`，要求 `max` 落在 `int64` 正数范围内；超大 `max` 与 `Int63n` 约束一致。

### D3: 命名

- `RemainingSeconds` — 直译「剩余秒」
- `Random` — 简短；若与标准库混淆，包内调用为 `dt.Random(max)`

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| 每次 `Random` 新建 `rand.Rand`，高频调用略慢 | 任务规模小，可接受；后续可抽包级源 |
| 同一毫秒内多次调用可能相同种子 | 文档说明非加密、非强唯一 |
| `RemainingSeconds` 依赖系统时钟 | 与 `New(nil)` 行为一致 |

## Migration Plan

1. 实现方法 + 测试。
2. 更新 README 2.1。
3. `openspec validate` → `/opsx:apply`。

## Open Questions

（无）

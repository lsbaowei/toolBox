# Design — optimize-rand-reuse

## Context

任务 3 已修复 `IntWithSafety` 上界与 `RandUtil.Intn` 边界。任务 4 聚焦性能与 API 分层：伪随机 vs 密码学安全。

## Goals / Non-Goals

**Goals:**

- `DateTime.Random` 与 `utils_random.IntV2` 复用长期存活的 `*rand.Rand`。
- 提供清晰的 **secure** API：`SecureIntn(max)`、`SecureInt64()`（包装现有逻辑）。
- README 任务 4 补充用法对比示例。

**Non-Goals:**

- 不替换 `RandUtil`（已是正确模式）。
- 不把 `DateTime.Random` 改为 crypto（业务抖动/抽奖用伪随机即可）。
- 不删除 `Int()` Deprecated 函数。

## Decisions

### D1: 包级共享伪随机源（`utils_time`）

```go
var (
    pseudoRandMu sync.Mutex
    pseudoRand   = rand.New(rand.NewSource(time.Now().UnixMilli()))
)

func (d *DateTime) Random(max int64) int64 {
    if max <= 0 { return 0 }
    pseudoRandMu.Lock()
    pseudoRand.Seed(time.Now().UnixMilli())
    n := pseudoRand.Int63n(max)
    pseudoRandMu.Unlock()
    return n
}
```

- 保留「毫秒因子」语义（每次调用 `Seed`）。
- `DateTime` 结构体不增字段，避免破坏不可变拷贝模型。

### D2: `IntV2` 复用

委托包级 `pseudoRand` 的兄弟实现：在 `utils_random` 内定义 `sharedRand` + `IntV2` 调用 `Int63` 或 `Int()`，避免每次 `New`。

为减少跨包耦合，`utils_time` 与 `utils_random` **各自**维护包内 `shared *rand.Rand`（两个包各一把锁），不互相 import 循环。

### D3: 安全 API（`utils_random/secure.go`）

```go
// SecureIntn 返回 [0, max) 的密码学安全随机整数。
func SecureIntn(max int64) (int64, error)

// SecureInt64 返回 [0, 2^63) 的密码学安全随机整数（等同 IntWithSafety）。
func SecureInt64() (int64, error)
```

- `SecureIntn`：`max <= 0` 返回 `(0, nil)` 或 `errors.New` — 与 `Random` 一致返回 0。
- `IntWithSafety` 保留，内部可委托 `SecureInt64()`。

### D4: 示例

- `utils_random/secure_example_test.go` 或 README 代码块：`SecureIntn` vs `RandUtil.Intn` vs `DateTime.Random`。

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| 全局锁竞争 | 仅用于轻量随机；高 QPS 用独立 `RandUtil` 实例 |
| 每次 `Seed` 降低伪随机统计质量 | 与现行为一致，任务 4 不要求改语义 |
| 同毫秒并发相同种子 | 文档说明；安全场景用 `SecureIntn` |

## Migration Plan

1. 实现共享源 + secure API + 测试。
2. 更新 README 任务 4、CHANGELOG。
3. `/opsx:apply` → validate → archive。

## Open Questions

（无）

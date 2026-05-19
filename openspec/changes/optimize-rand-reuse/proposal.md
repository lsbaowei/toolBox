# optimize-rand-reuse

## Why

README 任务 4 要求：伪随机场景复用 `*rand.Rand` 提升性能；安全场景单独用 `crypto/rand` 并给出封装示例。当前 `DateTime.Random` 与 `IntV2` 每次调用都 `rand.New`，高并发下重复分配；`IntWithSafety` 已存在但缺少区间 `[0, max)` 的便捷 API 与文档示例。

## What Changes

- `utils_time.DateTime.Random`：改为复用包内共享 `*rand.Rand`（带锁），按调用以 `UnixMilli()` 更新种子，保持 `[0, max)` 语义不变。
- `utils_random`：新增 `SecureIntn(max int64) (int64, error)` 等 `crypto/rand` 封装；`IntV2` 可选改为复用 `RandUtil` 或包级 `*rand.Rand`。
- 补充测试与 README/CHANGELOG；**不**改变 `Random(max)` 对外签名。

## Capabilities

### New Capabilities

- `utils-random-secure`：`crypto/rand` 区间随机与安全整数 API。

### Modified Capabilities

- `utils-time-object`：`Random` 实现改为复用随机源（行为等价，非 API 变更）。

## Impact

- **代码**：`utils_time/object.go`、`utils_random/`（`secure.go` 或扩展现有文件）
- **BREAKING**：无（签名与返回值范围不变）
- **性能**：减少 `rand.New` 分配；共享源需互斥锁，仍优于每次新建

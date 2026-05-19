# toolbox-code-audit

## Why

README 任务 3 要求对全库代码做质量审查与优化。当前除 `utils_time` 外多数包无测试，`go vet` 虽通过但存在明确逻辑错误、库代码中的 `main`/panic、错误处理缺失及已废弃 API 使用等问题，需在保持对外行为兼容的前提下分批修复并记录变更。

## What Changes

### 必须修复（正确性 / 安全）

| 包 | 问题 |
|----|------|
| `utils_json` | `JSONDecode` 对 `result` 多取地址导致反序列化目标错误；`JSONEncode` 忽略 `Marshal` 错误 |
| `utils_random` | `IntWithSafety` 使用 `2^63`（Go 中为按位异或，非 2⁶³）；`Int()` 使用已废弃的 `rand.Seed` |
| `utils_random` | `RandUtil.Intn(n)` 在 `n <= 0` 时会 panic |
| `utils_csv` | `OnceFullWrite` 遇错 `panic`/`log.Fatal`，且忽略 `context` |
| `utils_exec` | `ExecCmd` 在 `ctx` 取消时未终止子进程，可能泄漏 goroutine |

### 结构 / 可维护性

| 包 | 问题 |
|----|------|
| `utils_csv` | 库包内含 `main()` 与 demo，不适合作为可导入 toolkit |
| `utils_time` | `getLoc` 未使用（死代码） |
| `utils_maps` | `Init()` 未调用时无清理协程；缺少 `Stop()` 关闭清理 |
| 全库 | 除 `utils_time` 外无单元测试 |

### 文档交付

- 新增 `CHANGELOG.md`（或 README「任务3」节）记录每项修复与未改动的已知限制。

## Capabilities

### New Capabilities

- `toolbox-quality`: 工具包库代码的错误处理、可测试性与库包边界约定。

### Modified Capabilities

（无既有 spec 覆盖 json/random/csv 等包）

## Impact

- **代码**：`utils_json`、`utils_random`、`utils_csv`、`utils_exec`、`utils_maps`、`utils_time`；可能新增 `*_test.go`
- **BREAKING**：`JSONDecode` 修复后行为变化（此前可能从未正确工作）；`utils_csv` 移除/迁移 `main` 可能影响直接 `go run` demo 的用户
- **依赖**：无新增外部依赖

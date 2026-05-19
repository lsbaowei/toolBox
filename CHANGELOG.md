# Changelog

## 2026-05-19 — toolbox-code-audit

### utils_json

- **BREAKING 修复**：`JSONDecode` 去掉多余的 `&result`，现与 `encoding/json.Unmarshal` 语义一致。
- **修复**：`JSONDecodeV2`、`ParseStruct` 同样修正反序列化目标。
- **新增**：`JSONEncodeE(v) (string, error)`；`JSONEncode` 委托其实现，失败仍返回 `""`。
- **改进**：`MapFilter` / `MapInt64Filter` 在 `len(input) <= max` 时返回拷贝，避免与调用方共享 map。

### utils_random

- **BREAKING 修复**：`IntWithSafety` 上界改为 `1 << 63`（原 `2^63` 为按位异或）；现返回 `(int64, error)` 不再 panic。
- **修复**：`RandUtil.Intn(n)` 在 `n <= 0` 时返回 `0`，避免 panic。
- **文档**：`Int()` 标记 `Deprecated`，推荐 `IntV2` 或 `RandUtil`。

### utils_csv

- **BREAKING**：库包移除 `main` 与 demo；demo 迁至 `cmd/csvdemo`（`go run ./cmd/csvdemo`）。
- **修复**：`OnceFullWrite` 遇错返回 `error`，支持 `ctx` 取消，不再 `panic`/`log.Fatal`。

### utils_exec

- **修复**：`ExecCmd` 改用 `exec.CommandContext`，取消上下文可终止子进程。

### utils_maps

- **新增**：`Stop()` 停止定时清理 goroutine。
- **修复**：`deepCopy` 使用 `reflect` 按 concrete 类型解码，修复 gob 对 `interface{}` 的解码错误。

### utils_time

- **清理**：删除未使用的 `getLoc`。

### 测试

- 为 `utils_json`、`utils_random`、`utils_csv`、`utils_exec`、`utils_maps` 新增单元测试。

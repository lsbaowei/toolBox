# Design — toolbox-code-audit

## Context

仓库为内部 golang toolkit，按功能分目录。`utils_time` 经 OpenSpec 迭代后质量较好；其余包多为早期草稿或 demo 混入。

## Goals / Non-Goals

**Goals:**

- 修复已确认的逻辑/API 误用，不扩大 scope 做大规模重构。
- 库函数以返回 `error` 替代 `panic`/`log.Fatal`（`OnceFullWrite` 等）。
- 为修复点补充针对性单元测试。
- 用 `CHANGELOG.md` 记录变更明细。

**Non-Goals:**

- 不重写 `utils_gc` 的 stderr 管道架构（仅文档注明平台/用法限制）。
- 不统一所有包的 API 命名风格。
- 不为每个包补全 100% 测试覆盖。

## Decisions

### D1: `JSONDecode` 修复

```go
func JSONDecode(v string, result interface{}) error {
    return json.Unmarshal([]byte(v), result)
}
```

调用方须传入指针（与 `encoding/json` 一致）。在 CHANGELOG 标明 **BREAKING**（若此前依赖错误行为则视为修复）。

### D2: `JSONEncode` 返回 error

新增 `JSONEncodeE(v interface{}) (string, error)`；原 `JSONEncode` 可保留为忽略错误的便捷函数并加注释，或改为返回 error——优先 **新增 `JSONEncodeE`**，原函数委托并 `_` 错误（最小破坏）或文档废弃。apply 阶段采用：`JSONEncode` 仍返回 string，新增 `JSONEncodeE`；内部 `JSONEncode` 调用 `Marshal` 失败时返回 `""` 并文档说明（或改为返回 error — proposal 说修复，用 `JSONEncodeE` + 测试即可）。

简化：**`JSONEncode` 改为返回 `(string, error)`** 会 BREAKING。采用：
- `JSONEncode` 保持签名，内部记录：失败返回 `""` 且可选后续加 `MustJSONEncode`
- 新增 `JSONEncodeE(v) (string, error)` 为推荐 API

### D3: `IntWithSafety`

```go
n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 63))
```

使用 `1<<63` 作为上界（或 `math.MaxInt64` 相关安全范围），修正注释。

### D4: `utils_random.Int`

标记 `Deprecated` 注释，文档推荐 `IntV2` 或 `New()`；不删除以保持兼容。

### D5: `RandUtil.Intn`

```go
if n <= 0 { return 0 } // 或 return error — 方法无 error 返回时用 0 并文档说明
```

### D6: `utils_csv`

- 将 `main` 及 demo 函数移至 `utils_csv/cmd/demo/main.go` 或 `example_test.go`（`Example` 不生成二进制）。
- `OnceFullWrite`：返回 error，用 `writer.Write` 错误返回，尊重 `ctx.Done()`。

### D7: `ExecCmd`

`cmd := exec.CommandContext(ctx, name, args...)` 替代 `exec.Command` + 手写 select，删除多余 goroutine（若 `CommandContext` 足够）。

### D8: `LocalSyncMap`

- 新增 `Stop()` 关闭 `stopChan`（仅一次）。
- 文档强调使用前须 `Init()`。
- `deepCopy` 失败时 `Set` 已返回 error — 保持。

### D9: `utils_time`

删除未使用的 `getLoc`。

### D10: 测试策略

每包至少覆盖本次修复路径；`utils_json`/`utils_random`/`utils_exec`/`utils_csv` 新增 `*_test.go`。

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| `JSONDecode` 修复改变运行时行为 | CHANGELOG + 测试 |
| 移动 csv `main` 破坏习惯 | README 注明 demo 路径 |
| `CommandContext` 在旧 Go 版本 | `go.mod` 为 1.19，支持 |

## Migration Plan

按 `tasks.md` 包顺序实施 → `go test ./...` → `go vet ./...` → 更新 CHANGELOG → validate。

## Open Questions

- `MapFilter` 在 `len(input) <= max` 时返回原 map 引用（调用方修改会影响内部）是否改为始终拷贝？apply 时可选小改 + 注释。

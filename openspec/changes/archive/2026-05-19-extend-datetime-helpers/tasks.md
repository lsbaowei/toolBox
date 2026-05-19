# Tasks — extend-datetime-helpers

## 1. 实现

- [x] 1.1 在 `object.go` 实现 `RemainingSeconds(other *time.Time) int64`
- [x] 1.2 在 `object.go` 实现 `Random(max int64) int64`（`UnixMilli` 播种，`max <= 0` 返回 0）

## 2. 测试

- [x] 2.1 测试 `RemainingSeconds`：显式 other、nil（可用注入或固定 base 与 mock 时间——若难以 mock，用固定 `New(&base)` + 固定 `other` 场景）
- [x] 2.2 测试 `Random`：`max <= 0`、多次调用结果落在 `[0,max)`
- [x] 2.3 运行 `go test ./utils_time/...`

## 3. 文档与收尾

- [x] 3.1 更新 README 任务 2.1：方法签名与行为说明
- [x] 3.2 运行 `openspec validate extend-datetime-helpers`

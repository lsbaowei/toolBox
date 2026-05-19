# Tasks — optimize-rand-reuse

## 1. 伪随机复用

- [ ] 1.1 `utils_time`：包级 `*rand.Rand` + 互斥锁，重构 `DateTime.Random`
- [ ] 1.2 `utils_random`：包级共享源，重构 `IntV2` 避免每次 `New`
- [ ] 1.3 为 `DateTime.Random` 增加「不重复 New」的测试或基准说明（可选 benchmark 注释）

## 2. 安全随机封装

- [ ] 2.1 新增 `secure.go`：`SecureIntn`、`SecureInt64`；`IntWithSafety` 委托 `SecureInt64`
- [ ] 2.2 新增 `secure_test.go` 与 `ExampleSecureIntn`（或 README 示例）

## 3. 文档与验证

- [ ] 3.1 更新 README 任务 4：伪随机 vs 安全 API 对照
- [ ] 3.2 更新 CHANGELOG.md
- [ ] 3.3 `go test ./...`；`openspec validate optimize-rand-reuse`

# Tasks — toolbox-code-audit

## 1. utils_json

- [x] 1.1 修复 `JSONDecode`（去掉多余的 `&result`）
- [x] 1.2 新增 `JSONEncodeE`；为 `JSONEncode`/`JSONDecode` 补充测试
- [x] 1.3 （可选）`MapFilter` 在返回时始终拷贝 map 或文档说明共享引用

## 2. utils_random

- [x] 2.1 修复 `IntWithSafety` 上界为 `1 << 63`
- [x] 2.2 `RandUtil.Intn` 对 `n <= 0` 返回 `0`
- [x] 2.3 为 `IntWithSafety`、`Intn` 添加测试；`Int()` 加 Deprecated 注释

## 3. utils_csv

- [x] 3.1 将 `main` 与 demo 迁至 `cmd/csvdemo` 或 `example_test.go`
- [x] 3.2 修复 `OnceFullWrite`：错误返回、支持 `ctx` 取消
- [x] 3.3 为 `OnceFullWrite` 添加测试（可用 temp 文件）

## 4. utils_exec

- [x] 4.1 改用 `exec.CommandContext`；简化 goroutine
- [x] 4.2 添加 context 取消测试（如 `sleep` + 短超时）

## 5. utils_maps

- [x] 5.1 新增 `Stop()` 停止清理 goroutine
- [x] 5.2 为 `Set`/`Get`/`SafeGet`/`Del` 添加基础测试

## 6. utils_time 与其它

- [x] 6.1 删除未使用的 `getLoc`
- [x] 6.2 新增 `CHANGELOG.md` 记录全部变更
- [x] 6.3 更新 README 任务 3 为已完成并指向 CHANGELOG

## 7. 验证

- [x] 7.1 `go test ./...` 与 `go vet ./...`
- [x] 7.2 `openspec validate toolbox-code-audit`

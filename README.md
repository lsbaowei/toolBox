# Toolkit for golang

`toolBox` 是一个按功能拆分的 Go 工具库，目标是提供简单、可测试、容易直接接入业务代码的小型工具包。

## 安装

```shell
go get github.com/lsbaowei/toolBox
```

按需导入具体子模块：

```go
import "github.com/lsbaowei/toolBox/utils_time"
```

## 模块索引

- [`utils_cache_version`](utils_cache_version/README.md)：第三方版本变更时的缓存版本渐进释放、粘性灰度和最新版本目标选择。
- [`utils_csv`](utils_csv/README.md)：CSV 一次性写入、带缓冲追加写入和结构化数据写入。
- [`utils_exec`](utils_exec/README.md)：外部命令执行封装，捕获 `stdout`、`stderr` 和执行错误。
- [`utils_gc`](utils_gc/README.md)：Go `gctrace` 日志捕获、解析和结构化输出。
- [`utils_json`](utils_json/README.md)：JSON 编解码、map 辅助函数和 protobuf 结构转换。
- [`utils_maps`](utils_maps/README.md)：基于 `sync.Map` 的进程内过期缓存。
- [`utils_random`](utils_random/README.md)：伪随机、安全随机和可指定 seed 的随机工具。
- [`utils_time`](utils_time/README.md)：时间字符串解析、Unix 时间戳解析和不可变 `DateTime`。

## 设计原则

- 按功能目录组织，每个 `utils_*` 目录对应一个 Go package。
- 尽量只依赖 Go 标准库；确需外部依赖时保持明确边界。
- 公共函数保持小而直接，便于复制到业务场景中使用。
- 重要边界通过单元测试覆盖。

## 测试

运行全量测试：

```shell
go test ./...
```

运行 `utils_cache_version` benchmark：

```shell
go test ./utils_cache_version -run=^$ -bench='BenchmarkManagerSelectVersion' -benchmem
```

## 文档

每个子模块都有独立 `README.md`，包含适用场景、主要 API、示例和注意事项。根 README 只保留整体介绍与模块导航。

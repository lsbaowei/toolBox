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

### 缓存

- [`utils_cache_memory`](utils_cache_memory/README.md)：全局共享的并发字节缓存，支持绝对过期时间、零复制读取、主动过期清理和常用 `sync.Map` 操作。
- [`utils_maps`](utils_maps/README.md)：支持过期时间、后台定时清理和深拷贝读取的进程内通用对象缓存。
- [`utils_cache_version`](utils_cache_version/README.md)：通过渐进释放、确定性分桶和粘性灰度选择缓存版本，避免第三方版本切换引发瞬时流量。

### 数据处理

- [`utils_json`](utils_json/README.md)：JSON 编解码、对象与 map 转换、map 过滤与合并，以及 protobuf 和 Go 原生类型转换。
- [`utils_csv`](utils_csv/README.md)：CSV 一次性写入、带缓冲并发追加、表头管理和结构化数据写入。

### 系统与诊断

- [`utils_exec`](utils_exec/README.md)：支持 `context.Context` 的外部命令执行封装，统一捕获 `stdout`、`stderr` 和执行错误。
- [`utils_gc`](utils_gc/README.md)：Go `gctrace` 日志捕获、解析与结构化输出，用于 GC 行为调试和诊断。

### 基础能力

- [`utils_random`](utils_random/README.md)：并发安全伪随机、固定 seed 确定性随机和基于 `crypto/rand` 的安全随机工具。
- [`utils_time`](utils_time/README.md)：常见时间字符串与 Unix 时间戳解析、时区处理，以及不可变 `DateTime` 日期边界计算。

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

运行 `utils_cache_memory` benchmark：

```shell
go test ./utils_cache_memory -run=^$ -bench=. -benchmem
```

## 文档

每个子模块都有独立 `README.md`，包含适用场景、主要 API、示例和注意事项。根 README 只保留整体介绍与模块导航。

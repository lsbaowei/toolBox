# utils_gc

`utils_gc` 提供 Go GC trace 日志解析与重定向工具，适合在本地调试或诊断程序 GC 行为时使用。

## 功能概览

- `RejectGCTraceLog`：重定向 `stderr`，捕获 `GODEBUG=gctrace=1` 输出并解析为 JSON。
- `NewGcTraceLog`：把单行 GC trace 日志解析为 JSON 字符串。
- `ParseGCLogHybridD1`：把单行 GC trace 日志解析为结构化 `GCStatsD1`。
- `GCTrace`：设置 `debug.SetGCPercent(100)`。

## 安装与导入

```go
import "github.com/lsbaowei/toolBox/utils_gc"
```

## 使用前提

GC trace 需要通过环境变量打开：

```shell
GODEBUG=gctrace=1 go run ./cmd/your-app
```

如果需要捕获完整程序运行期间的 trace，`RejectGCTraceLog` 应尽量在 `main` 函数最开始调用。

```go
func main() {
    if err := utils_gc.RejectGCTraceLog(); err != nil {
        panic(err)
    }

    // application code
}
```

## 解析单行 GC trace

```go
line := "gc 25 @59.374s 0%: 0.26+2.8+0.036 ms clock, 2.1+0/2.6/0+0.29 ms cpu, 971->971->971 MB, 974 MB goal, 0 MB stacks, 0 MB globals, 8 P"

stats, err := utils_gc.ParseGCLogHybridD1(line)
if err != nil {
    // handle error
}
_ = stats
```

转成 JSON 字符串：

```go
s, err := utils_gc.NewGcTraceLog(line)
if err != nil {
    // handle error
}
_ = s
```

## `GCStatsD1` 字段

`GCStatsD1` 包含以下主要信息：

- `GCNumber`：GC 次数。
- `Uptime`：程序运行时间，单位秒。
- `CPUPercent`：GC CPU 占比。
- `ClockSTWScan`、`ClockMark`、`ClockMarkTerm`：时钟时间，单位毫秒。
- `CPUSTWScan`、`CPUForced`、`CPUMark`、`CPUAssist`、`CPUMarkTerm`：CPU 时间，单位毫秒。
- `HeapBefore`、`HeapAfter`、`HeapLive`、`HeapGoal`：堆内存信息，单位 MB。
- `Stacks`、`Globals`：栈和全局内存，单位 MB。
- `Procs`：P 的数量。

## 注意事项

- `RejectGCTraceLog` 会重定向进程的 `stderr`，应只在明确需要捕获 GC trace 的程序中使用。
- 它会同时捕获非 GC 的 `stderr` 输出，并以 `[STDERR]` 前缀打印。
- 该工具偏诊断用途，不建议在普通线上业务默认开启。
- `ParseGCLogHybridD1` 依赖 Go 当前 gctrace 文本格式，Go 版本变化可能导致解析失败。

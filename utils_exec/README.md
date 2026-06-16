# utils_exec

`utils_exec` 封装外部命令执行，统一记录命令信息、标准输出、标准错误和执行错误。它适合在工具程序、脚本封装和后台任务中调用系统命令。

## 功能概览

- 使用 `context.Context` 控制命令生命周期。
- 捕获 `stdout` 和 `stderr`。
- 提供 `Success`、`IsError`、`IsStderr` 等结果判断方法。
- 保留命令名称和参数，便于日志记录和排查。

## 安装与导入

```go
import "github.com/lsbaowei/toolBox/utils_exec"
```

## 基本用法

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

info := utils_exec.ExecCmd(ctx, "echo", []string{"hello"})
result := info.GetResult()

if result.Success() {
    fmt.Println(result.GetStdoutString())
}
if result.IsError() {
    fmt.Println(result.Err)
}
if result.IsStderr() {
    fmt.Println(result.GetStderrString())
}
```

## 核心类型

### `ExecInfo`

```go
type ExecInfo struct {
    Command *ExecCommand
    Result  *ExecResult
}
```

`ExecInfo` 保存本次执行的命令和结果。

### `ExecCommand`

```go
type ExecCommand struct {
    Name string
    Args []string
}
```

`ExecCommand` 保存命令名和参数。

### `ExecResult`

```go
type ExecResult struct {
    Stdout bytes.Buffer
    Stderr bytes.Buffer
    Err    error
}
```

`ExecResult` 保存标准输出、标准错误和执行错误。

## 结果判断

- `IsError() bool`：`ExecResult` 为 nil 或 `Err != nil` 时返回 true。
- `IsStderr() bool`：`ExecResult` 为 nil 或 `stderr` 非空时返回 true。
- `Success() bool`：没有执行错误且没有 `stderr` 输出时返回 true。
- `GetStdoutString() string`：返回标准输出字符串。
- `GetStderrString() string`：返回标准错误字符串。

## 注意事项

- `ExecCmd` 使用 `exec.CommandContext`，当 `ctx` 取消或超时时会终止子进程。
- `Success` 会把非空 `stderr` 视为失败；如果某些命令会把提示信息写到 `stderr`，调用方应按业务语义自行判断。
- `args` 应按参数切片传入，不要把整条命令拼成一个字符串。

# utils_csv

`utils_csv` 提供 CSV 文件写入工具，支持一次性写入和带缓冲的追加写入，适合导出报表、落地批处理结果和写入简单结构化数据。

## 功能概览

- `CsvWriterObj.OnceFullWrite`：一次性写入多行 CSV 记录，支持 `context.Context` 取消。
- `CSVWriter`：带缓冲的追加写入器，支持表头、并发写入保护、手动刷新和关闭。
- `WriteData`：支持 `[]string`、`map[string]string` 和普通结构体写入。

## 安装与导入

```go
import "github.com/lsbaowei/toolBox/utils_csv"
```

## 一次性写入

`OnceFullWrite(ctx context.Context, fs string, records [][]string) error` 会以追加模式打开文件，并逐行写入记录。

```go
writer := &utils_csv.CsvWriterObj{}
err := writer.OnceFullWrite(ctx, "report.csv", [][]string{
    {"name", "age"},
    {"alice", "18"},
    {"bob", "20"},
})
if err != nil {
    // handle error
}
```

注意事项：

- `fs == ""` 时返回错误。
- `ctx == nil` 时内部使用 `context.Background()`。
- 如果 `ctx` 在写入过程中取消，会返回 `ctx.Err()`。
- 文件以 `os.O_APPEND|os.O_CREATE|os.O_WRONLY` 模式打开。

## 带缓冲写入

`NewCSVWriter(filename string, header []string) (*CSVWriter, error)` 创建追加写入器。如果文件不存在，会先写入表头。

```go
w, err := utils_csv.NewCSVWriter("users.csv", []string{"name", "age"})
if err != nil {
    // handle error
}
defer w.Close()

_ = w.WriteRow([]string{"alice", "18"})
_ = w.WriteData(map[string]string{
    "name": "bob",
    "age":  "20",
})
_ = w.Flush()
```

`CSVWriter` 内部默认缓冲 10 行，缓冲满时自动写入文件。也可以调用 `Flush` 手动刷新。

## `WriteData` 输入规则

- `[]string`：直接作为一行写入。
- `map[string]string`：按 header 顺序提取字段，缺失字段写入空字符串。
- 其他类型：序列化为 JSON 字符串后作为单列写入。

```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

_ = w.WriteData(User{Name: "cindy", Age: 22})
```

## 并发与资源管理

- `WriteRow`、`WriteData`、`Flush` 内部使用锁保护，可在多个 goroutine 中调用。
- 使用 `NewCSVWriter` 后应调用 `Close`，确保剩余缓冲数据写入并关闭文件。
- 如果业务需要立即落盘，请在关键位置调用 `Flush`。

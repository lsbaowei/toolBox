# utils_csv_sql

`utils_csv_sql` 从带表头的 CSV 文件读取数据，按字段配置完成类型转换，并同时返回按 SQL 批次分组的 `[][]map[string]any` 和批量 `INSERT SQL`。

## 功能概览

- 校验 CSV 表头及配置的源字段是否存在。
- 支持从 CSV 字段取值或为目标字段设置固定值。
- 内置字符串、`int64`、`float64`、`bool` 和时间转换器。
- 支持自定义 `Converter`。
- 支持按最大行数拆分批量 `INSERT SQL`。
- 支持 MySQL `INSERT IGNORE` 和 PostgreSQL `ON CONFLICT DO NOTHING`。
- 可选择在字段转换失败时立即返回错误，或跳过整行。
- SQL 字符串值会使用单引号包裹，并将内部单引号转义为两个单引号。

## 基本用法

```go
cfg := utils_csv_sql.Config{
	Table: "public.users",
	Fields: []utils_csv_sql.FieldConfig{
		{
			Column:    "name",
			CSVField:  "姓名",
			Converter: utils_csv_sql.ToTrimmedString,
		},
		{
			Column:    "age",
			CSVField:  "年龄",
			Converter: utils_csv_sql.ToInt64,
		},
		{
			Column:        "source",
			FixedValue:    "csv",
			UseFixedValue: true,
		},
	},
	BatchSize:       500,
	SkipInvalidRows: true,
	IgnoreConflicts: true,
	Dialect:         utils_csv_sql.DialectPostgreSQL,
}

result, err := utils_csv_sql.ConvertFile("/path/to/users.csv", cfg)
if err != nil {
	// handle error
}

totalRows := 0
for i, sql := range result.SQL {
	fmt.Printf("sql=%s rows=%v\n", sql, result.Rows[i])
	totalRows += len(result.Rows[i])
}
fmt.Printf("rows=%d skipped=%d\n", totalRows, result.SkippedRows)
```

## 配置

### `Config`

- `Table`：目标表名，支持 `schema.table`；每一段只能包含字母、数字和下划线，且不能以数字开头。
- `Fields`：目标字段配置，顺序同时决定 SQL 字段和值的顺序。
- `BatchSize`：每条 SQL 包含的最大行数；小于等于 `0` 时默认为 `100`。
- `SkipInvalidRows`：字段转换失败时是否跳过整行。
- `IgnoreConflicts`：是否忽略写入冲突。
- `Dialect`：忽略冲突时必须设置为 `DialectMySQL` 或 `DialectPostgreSQL`。

冲突忽略语法：

- MySQL：`INSERT IGNORE INTO ...`
- PostgreSQL：`INSERT INTO ... ON CONFLICT DO NOTHING`

### `FieldConfig`

- `Column`：目标表字段名，也作为结果 map 的 key。
- `CSVField`：CSV 表头中的源字段名。
- `Converter`：可选转换函数；为空时保留原始字符串。
- `UseFixedValue`：为 `true` 时不读取 CSVField，而是使用 `FixedValue`。
- `FixedValue`：固定值，可设置为 `nil` 生成 SQL `NULL`。

## 转换器

内置转换器：

- `ToString`
- `ToTrimmedString`
- `ToInt64`
- `ToFloat64`
- `ToBool`
- `ToTime(layout)`

自定义转换器：

```go
upper := func(raw string) (any, error) {
	return strings.ToUpper(strings.TrimSpace(raw)), nil
}
```

转换结果用于生成 SQL 时支持以下类型：

- `nil`
- `string`、`[]byte`
- `bool`
- 所有有符号和无符号整数类型
- `float32`、`float64`
- `time.Time`

## 返回方式

- `ConvertFile`：一次返回 `Result`。`SQL` 与 `Rows` 长度相同，`SQL[i]` 对应 `Rows[i]` 中参与该条批量 SQL 的数据。
- `ConvertToSQL`：仅返回 `[]string` SQL。
- `ConvertToMaps`：忽略 SQL 批次，仅返回扁平的 `[]map[string]any`。

## 异常处理

默认情况下，字段转换失败会返回 `*RowError`，包含 CSV 记录号、源字段、目标字段、原始值和底层错误。设置 `SkipInvalidRows: true` 后，该行不会进入 SQL 或结果 map，并计入 `Result.SkippedRows`。

文件不存在、CSV 为空、表头重复、配置字段不存在、标识符非法或 SQL 值类型不支持时始终返回错误，不受 `SkipInvalidRows` 影响。

## 注意事项

- 输入 CSV 必须包含表头。
- 生成结果是可直接查看或导出的 SQL 文本，不是参数化查询。执行前仍应由调用方确认目标数据库方言和权限。
- 当前生成通用多值 `INSERT INTO ... VALUES ...` 语法，不处理分区、冲突更新、返回值等数据库特有功能。
- SQL 字符串按标准单引号规则转义；二进制 `[]byte` 会按字符串处理，不生成数据库专用二进制字面量。
- 转换过程会将所有结果行和 SQL 保存在内存中，大文件应合理设置 `BatchSize` 并评估内存使用。

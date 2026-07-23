// Package utils_csv_sql 将带表头的 CSV 文件转换为结构化数据和批量 INSERT SQL。
package utils_csv_sql

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const defaultBatchSize = 100

// Dialect 表示生成 INSERT SQL 时使用的数据库方言。
type Dialect string

const (
	// DialectMySQL 使用 MySQL INSERT 语法。
	DialectMySQL Dialect = "mysql"
	// DialectPostgreSQL 使用 PostgreSQL INSERT 语法。
	DialectPostgreSQL Dialect = "postgresql"
)

var (
	// ErrEmptyPath 表示 CSV 文件路径为空。
	ErrEmptyPath = errors.New("csv path must not be empty")
	// ErrEmptyTable 表示目标表名为空。
	ErrEmptyTable = errors.New("table must not be empty")
	// ErrEmptyFields 表示未配置任何目标字段。
	ErrEmptyFields = errors.New("fields must not be empty")
	// ErrEmptyCSV 表示 CSV 文件中没有表头。
	ErrEmptyCSV = errors.New("csv must contain a header")
)

// Converter 将 CSV 原始字符串转换为 SQL 和结果 map 使用的值。
type Converter func(raw string) (any, error)

// FieldConfig 定义一个目标字段的取值方式。
type FieldConfig struct {
	// Column 是 INSERT SQL 和结果 map 使用的目标字段名。
	Column string
	// CSVField 是从 CSV 表头读取的源字段名。
	CSVField string
	// FixedValue 是 UseFixedValue 为 true 时使用的固定值，可为 nil。
	FixedValue any
	// UseFixedValue 表示忽略 CSVField 并使用 FixedValue。
	UseFixedValue bool
	// Converter 对 CSV 原始值进行转换；为空时保留为 string。
	Converter Converter
}

// Config 定义 CSV 转换和 SQL 生成规则。
type Config struct {
	// Table 是 INSERT SQL 的目标表名，支持 schema.table 形式。
	Table string
	// Fields 按 INSERT SQL 字段顺序定义取值规则。
	Fields []FieldConfig
	// BatchSize 是每条 INSERT SQL 包含的最大行数；小于等于 0 时默认为 100。
	BatchSize int
	// SkipInvalidRows 表示字段转换失败时跳过整行；否则立即返回错误。
	SkipInvalidRows bool
	// IgnoreConflicts 表示写入冲突时忽略该行。
	IgnoreConflicts bool
	// Dialect 在 IgnoreConflicts 为 true 时指定 MySQL 或 PostgreSQL。
	Dialect Dialect
}

// Result 同时包含生成的 INSERT SQL 和按 SQL 批次分组的结构化行。
//
// SQL 与 Rows 长度相同，SQL[i] 由 Rows[i] 中的数据生成。
type Result struct {
	SQL         []string
	Rows        [][]map[string]any
	SkippedRows int
}

// RowError 描述某条 CSV 记录的字段转换错误。
type RowError struct {
	Record   int
	CSVField string
	Column   string
	Value    string
	Err      error
}

// Error 返回包含记录号和字段信息的错误文本。
func (e *RowError) Error() string {
	return fmt.Sprintf(
		"convert csv record %d field %q to column %q with value %q: %v",
		e.Record,
		e.CSVField,
		e.Column,
		e.Value,
		e.Err,
	)
}

// Unwrap 返回底层转换错误。
func (e *RowError) Unwrap() error {
	return e.Err
}

// ConvertFile 读取带表头的 CSV 文件，返回转换后的数据和批量 INSERT SQL。
func ConvertFile(path string, cfg Config) (Result, error) {
	if path == "" {
		return Result{}, ErrEmptyPath
	}
	if err := validateConfig(cfg); err != nil {
		return Result{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		return Result{}, fmt.Errorf("open csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return Result{}, ErrEmptyCSV
	}
	if err != nil {
		return Result{}, fmt.Errorf("read csv header: %w", err)
	}

	index, err := buildHeaderIndex(header)
	if err != nil {
		return Result{}, err
	}
	if err := validateSourceFields(cfg.Fields, index); err != nil {
		return Result{}, err
	}

	result := Result{}
	rows := make([]map[string]any, 0)
	for recordNumber := 2; ; recordNumber++ {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return Result{}, fmt.Errorf("read csv record %d: %w", recordNumber, readErr)
		}

		row, convertErr := convertRecord(recordNumber, record, index, cfg.Fields)
		if convertErr != nil {
			if cfg.SkipInvalidRows {
				result.SkippedRows++
				continue
			}
			return Result{}, convertErr
		}
		rows = append(rows, row)
	}

	result.SQL, result.Rows, err = buildInsertSQL(cfg, rows)
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

// ConvertToSQL 读取 CSV 文件并仅返回批量 INSERT SQL。
func ConvertToSQL(path string, cfg Config) ([]string, error) {
	result, err := ConvertFile(path, cfg)
	if err != nil {
		return nil, err
	}
	return result.SQL, nil
}

// ConvertToMaps 读取 CSV 文件并仅返回转换后的结构化数据。
func ConvertToMaps(path string, cfg Config) ([]map[string]any, error) {
	result, err := ConvertFile(path, cfg)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0)
	for _, batch := range result.Rows {
		rows = append(rows, batch...)
	}
	return rows, nil
}

// ToString 原样返回 CSV 字符串。
func ToString(raw string) (any, error) {
	return raw, nil
}

// ToTrimmedString 返回去除首尾空白后的字符串。
func ToTrimmedString(raw string) (any, error) {
	return strings.TrimSpace(raw), nil
}

// ToInt64 将 CSV 字符串转换为 int64。
func ToInt64(raw string) (any, error) {
	return strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
}

// ToFloat64 将 CSV 字符串转换为有限 float64。
func ToFloat64(raw string) (any, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return nil, err
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, errors.New("float must be finite")
	}
	return value, nil
}

// ToBool 使用 strconv.ParseBool 支持的格式转换布尔值。
func ToBool(raw string) (any, error) {
	return strconv.ParseBool(strings.TrimSpace(raw))
}

// ToTime 返回按 layout 解析时间的转换函数。
func ToTime(layout string) Converter {
	return func(raw string) (any, error) {
		return time.Parse(layout, strings.TrimSpace(raw))
	}
}

func validateConfig(cfg Config) error {
	if !isQualifiedIdentifier(cfg.Table) {
		if cfg.Table == "" {
			return ErrEmptyTable
		}
		return fmt.Errorf("invalid table name %q", cfg.Table)
	}
	if len(cfg.Fields) == 0 {
		return ErrEmptyFields
	}
	if cfg.IgnoreConflicts && cfg.Dialect != DialectMySQL && cfg.Dialect != DialectPostgreSQL {
		return fmt.Errorf(
			"ignore conflicts requires dialect %q or %q",
			DialectMySQL,
			DialectPostgreSQL,
		)
	}

	columns := make(map[string]struct{}, len(cfg.Fields))
	for i, field := range cfg.Fields {
		if !isIdentifier(field.Column) {
			return fmt.Errorf("invalid column name %q at fields[%d]", field.Column, i)
		}
		if _, exists := columns[field.Column]; exists {
			return fmt.Errorf("duplicate column %q", field.Column)
		}
		columns[field.Column] = struct{}{}
		if !field.UseFixedValue && field.CSVField == "" {
			return fmt.Errorf("fields[%d] must configure CSVField or UseFixedValue", i)
		}
	}
	return nil
}

func buildHeaderIndex(header []string) (map[string]int, error) {
	index := make(map[string]int, len(header))
	for i, name := range header {
		if name == "" {
			return nil, fmt.Errorf("csv header at index %d is empty", i)
		}
		if _, exists := index[name]; exists {
			return nil, fmt.Errorf("duplicate csv header %q", name)
		}
		index[name] = i
	}
	return index, nil
}

func validateSourceFields(fields []FieldConfig, index map[string]int) error {
	for _, field := range fields {
		if field.UseFixedValue {
			continue
		}
		if _, exists := index[field.CSVField]; !exists {
			return fmt.Errorf("csv field %q for column %q does not exist", field.CSVField, field.Column)
		}
	}
	return nil
}

func convertRecord(
	recordNumber int,
	record []string,
	index map[string]int,
	fields []FieldConfig,
) (map[string]any, error) {
	row := make(map[string]any, len(fields))
	for _, field := range fields {
		if field.UseFixedValue {
			row[field.Column] = field.FixedValue
			continue
		}

		raw := record[index[field.CSVField]]
		value := any(raw)
		var err error
		if field.Converter != nil {
			value, err = field.Converter(raw)
		}
		if err != nil {
			return nil, &RowError{
				Record:   recordNumber,
				CSVField: field.CSVField,
				Column:   field.Column,
				Value:    raw,
				Err:      err,
			}
		}
		row[field.Column] = value
	}
	return row, nil
}

func buildInsertSQL(cfg Config, rows []map[string]any) ([]string, [][]map[string]any, error) {
	if len(rows) == 0 {
		return []string{}, [][]map[string]any{}, nil
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	columns := make([]string, len(cfg.Fields))
	for i, field := range cfg.Fields {
		columns[i] = field.Column
	}
	prefix := insertPrefix(cfg) + cfg.Table + " (" + strings.Join(columns, ", ") + ") VALUES "
	suffix := insertSuffix(cfg)

	sqlList := make([]string, 0, (len(rows)+batchSize-1)/batchSize)
	rowBatches := make([][]map[string]any, 0, cap(sqlList))
	for start := 0; start < len(rows); start += batchSize {
		end := start + batchSize
		if end > len(rows) {
			end = len(rows)
		}

		values := make([]string, 0, end-start)
		for _, row := range rows[start:end] {
			parts := make([]string, len(columns))
			for i, column := range columns {
				literal, err := sqlLiteral(row[column])
				if err != nil {
					return nil, nil, fmt.Errorf("format column %q: %w", column, err)
				}
				parts[i] = literal
			}
			values = append(values, "("+strings.Join(parts, ", ")+")")
		}
		sqlList = append(sqlList, prefix+strings.Join(values, ", ")+suffix+";")
		rowBatches = append(rowBatches, rows[start:end])
	}
	return sqlList, rowBatches, nil
}

func insertPrefix(cfg Config) string {
	if cfg.IgnoreConflicts && cfg.Dialect == DialectMySQL {
		return "INSERT IGNORE INTO "
	}
	return "INSERT INTO "
}

func insertSuffix(cfg Config) string {
	if cfg.IgnoreConflicts && cfg.Dialect == DialectPostgreSQL {
		return " ON CONFLICT DO NOTHING"
	}
	return ""
}

func sqlLiteral(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "NULL", nil
	case string:
		return quoteSQLString(typed), nil
	case []byte:
		return quoteSQLString(string(typed)), nil
	case bool:
		if typed {
			return "TRUE", nil
		}
		return "FALSE", nil
	case int:
		return strconv.FormatInt(int64(typed), 10), nil
	case int8:
		return strconv.FormatInt(int64(typed), 10), nil
	case int16:
		return strconv.FormatInt(int64(typed), 10), nil
	case int32:
		return strconv.FormatInt(int64(typed), 10), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case uint:
		return strconv.FormatUint(uint64(typed), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(typed), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(typed), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(typed), 10), nil
	case uint64:
		return strconv.FormatUint(typed, 10), nil
	case float32:
		return formatFloat(float64(typed), 32)
	case float64:
		return formatFloat(typed, 64)
	case time.Time:
		return quoteSQLString(typed.Format(time.RFC3339Nano)), nil
	default:
		return "", fmt.Errorf("unsupported SQL value type %T", value)
	}
}

func quoteSQLString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func formatFloat(value float64, bitSize int) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", errors.New("float must be finite")
	}
	return strconv.FormatFloat(value, 'g', -1, bitSize), nil
}

func isQualifiedIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, part := range strings.Split(value, ".") {
		if !isIdentifier(part) {
			return false
		}
	}
	return true
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

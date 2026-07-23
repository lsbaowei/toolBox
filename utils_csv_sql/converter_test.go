package utils_csv_sql

import (
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestConvertFile(t *testing.T) {
	path := writeCSV(t, [][]string{
		{"name", "age", "active", "score"},
		{"O'Reilly", "18", "true", "91.5"},
		{"Bob", "20", "false", "88"},
		{"Cindy", "22", "true", "95"},
	})
	cfg := Config{
		Table: "public.users",
		Fields: []FieldConfig{
			{Column: "name", CSVField: "name", Converter: ToTrimmedString},
			{Column: "age", CSVField: "age", Converter: ToInt64},
			{Column: "active", CSVField: "active", Converter: ToBool},
			{Column: "score", CSVField: "score", Converter: ToFloat64},
			{Column: "source", FixedValue: "csv", UseFixedValue: true},
			{Column: "deleted_at", FixedValue: nil, UseFixedValue: true},
		},
		BatchSize: 2,
	}

	result, err := ConvertFile(path, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.SkippedRows != 0 || len(result.Rows) != 2 {
		t.Fatalf("result counts = batches:%d skipped:%d", len(result.Rows), result.SkippedRows)
	}
	if len(result.Rows[0]) != 2 || len(result.Rows[1]) != 1 {
		t.Fatalf("row batch sizes = %d, %d", len(result.Rows[0]), len(result.Rows[1]))
	}
	if got := result.Rows[0][0]["age"]; got != int64(18) {
		t.Fatalf("first age = %#v", got)
	}
	if got := result.Rows[0][0]["active"]; got != true {
		t.Fatalf("first active = %#v", got)
	}
	wantSQL := []string{
		"INSERT INTO public.users (name, age, active, score, source, deleted_at) VALUES ('O''Reilly', 18, TRUE, 91.5, 'csv', NULL), ('Bob', 20, FALSE, 88, 'csv', NULL);",
		"INSERT INTO public.users (name, age, active, score, source, deleted_at) VALUES ('Cindy', 22, TRUE, 95, 'csv', NULL);",
	}
	if !reflect.DeepEqual(result.SQL, wantSQL) {
		t.Fatalf("SQL = %#v", result.SQL)
	}
}

func TestConvertFileSkipInvalidRows(t *testing.T) {
	path := writeCSV(t, [][]string{
		{"id", "name"},
		{"1", "Alice"},
		{"invalid", "Bad"},
		{"2", "Bob"},
	})
	cfg := Config{
		Table: "users",
		Fields: []FieldConfig{
			{Column: "id", CSVField: "id", Converter: ToInt64},
			{Column: "name", CSVField: "name"},
		},
		SkipInvalidRows: true,
	}

	result, err := ConvertFile(path, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.SkippedRows != 1 || len(result.Rows) != 1 || len(result.Rows[0]) != 2 {
		t.Fatalf("result counts = batches:%d rows:%d skipped:%d", len(result.Rows), len(result.Rows[0]), result.SkippedRows)
	}
	if len(result.SQL) != 1 || strings.Contains(result.SQL[0], "Bad") {
		t.Fatalf("SQL = %#v", result.SQL)
	}
}

func TestConvertFileIgnoreConflicts(t *testing.T) {
	path := writeCSV(t, [][]string{
		{"id"},
		{"1"},
	})
	base := Config{
		Table: "users",
		Fields: []FieldConfig{
			{Column: "id", CSVField: "id", Converter: ToInt64},
		},
		IgnoreConflicts: true,
	}

	tests := []struct {
		name    string
		dialect Dialect
		want    string
	}{
		{
			name:    "mysql",
			dialect: DialectMySQL,
			want:    "INSERT IGNORE INTO users (id) VALUES (1);",
		},
		{
			name:    "postgresql",
			dialect: DialectPostgreSQL,
			want:    "INSERT INTO users (id) VALUES (1) ON CONFLICT DO NOTHING;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			cfg.Dialect = tt.dialect
			result, err := ConvertFile(path, cfg)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.SQL) != 1 || result.SQL[0] != tt.want {
				t.Fatalf("SQL = %#v", result.SQL)
			}
			if len(result.Rows) != 1 || len(result.Rows[0]) != 1 {
				t.Fatalf("Rows = %#v", result.Rows)
			}
		})
	}
}

func TestConvertFileReturnsRowError(t *testing.T) {
	path := writeCSV(t, [][]string{
		{"id"},
		{"invalid"},
	})
	cfg := Config{
		Table: "users",
		Fields: []FieldConfig{
			{Column: "id", CSVField: "id", Converter: ToInt64},
		},
	}

	_, err := ConvertFile(path, cfg)
	var rowErr *RowError
	if !errors.As(err, &rowErr) {
		t.Fatalf("error = %v", err)
	}
	if rowErr.Record != 2 || rowErr.CSVField != "id" || rowErr.Column != "id" {
		t.Fatalf("row error = %+v", rowErr)
	}
}

func TestConvertFileValidatesHeaderAndConfig(t *testing.T) {
	validPath := writeCSV(t, [][]string{{"id"}, {"1"}})

	tests := []struct {
		name string
		path string
		cfg  Config
		want string
	}{
		{
			name: "empty path",
			cfg:  Config{Table: "users", Fields: []FieldConfig{{Column: "id", CSVField: "id"}}},
			want: ErrEmptyPath.Error(),
		},
		{
			name: "invalid table",
			path: validPath,
			cfg:  Config{Table: "users;drop", Fields: []FieldConfig{{Column: "id", CSVField: "id"}}},
			want: "invalid table",
		},
		{
			name: "missing source field",
			path: validPath,
			cfg:  Config{Table: "users", Fields: []FieldConfig{{Column: "name", CSVField: "name"}}},
			want: "does not exist",
		},
		{
			name: "duplicate column",
			path: validPath,
			cfg: Config{Table: "users", Fields: []FieldConfig{
				{Column: "id", CSVField: "id"},
				{Column: "id", CSVField: "id"},
			}},
			want: "duplicate column",
		},
		{
			name: "missing conflict dialect",
			path: validPath,
			cfg: Config{
				Table:           "users",
				Fields:          []FieldConfig{{Column: "id", CSVField: "id"}},
				IgnoreConflicts: true,
			},
			want: "requires dialect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ConvertFile(tt.path, tt.cfg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestConvertFileEmptyAndDuplicateHeader(t *testing.T) {
	emptyPath := filepath.Join(t.TempDir(), "empty.csv")
	if err := os.WriteFile(emptyPath, nil, 0600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Table:  "users",
		Fields: []FieldConfig{{Column: "id", CSVField: "id"}},
	}
	if _, err := ConvertFile(emptyPath, cfg); !errors.Is(err, ErrEmptyCSV) {
		t.Fatalf("empty CSV error = %v", err)
	}

	duplicatePath := writeCSV(t, [][]string{{"id", "id"}, {"1", "2"}})
	if _, err := ConvertFile(duplicatePath, cfg); err == nil || !strings.Contains(err.Error(), "duplicate csv header") {
		t.Fatalf("duplicate header error = %v", err)
	}
}

func TestConvertWrappersAndTimeConverter(t *testing.T) {
	path := writeCSV(t, [][]string{
		{"created_at"},
		{"2026-07-23"},
	})
	cfg := Config{
		Table: "events",
		Fields: []FieldConfig{
			{Column: "created_at", CSVField: "created_at", Converter: ToTime("2006-01-02")},
		},
	}

	rows, err := ConvertToMaps(path, cfg)
	if err != nil {
		t.Fatal(err)
	}
	gotTime, ok := rows[0]["created_at"].(time.Time)
	if !ok || gotTime.Format("2006-01-02") != "2026-07-23" {
		t.Fatalf("created_at = %#v", rows[0]["created_at"])
	}

	sqlList, err := ConvertToSQL(path, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(sqlList) != 1 || sqlList[0] != "INSERT INTO events (created_at) VALUES ('2026-07-23T00:00:00Z');" {
		t.Fatalf("SQL = %#v", sqlList)
	}
}

func TestUnsupportedFixedValue(t *testing.T) {
	path := writeCSV(t, [][]string{{"id"}, {"1"}})
	cfg := Config{
		Table: "users",
		Fields: []FieldConfig{
			{Column: "metadata", FixedValue: map[string]string{"a": "b"}, UseFixedValue: true},
		},
	}

	if _, err := ConvertFile(path, cfg); err == nil || !strings.Contains(err.Error(), "unsupported SQL value type") {
		t.Fatalf("error = %v", err)
	}
}

func writeCSV(t *testing.T, records [][]string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "input.csv")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := csv.NewWriter(file)
	writer.WriteAll(records)
	if err := writer.Error(); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

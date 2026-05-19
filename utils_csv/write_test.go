package utils_csv

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOnceFullWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")

	obj := &CsvWriterObj{}
	records := [][]string{{"a", "b"}, {"1", "2"}}
	if err := obj.OnceFullWrite(context.Background(), path, records); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("rows: %d", len(got))
	}
}

func TestOnceFullWriteContextCancel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	obj := &CsvWriterObj{}
	err := obj.OnceFullWrite(ctx, path, [][]string{{"x"}})
	if err != context.Canceled {
		t.Fatalf("got %v", err)
	}
}

func TestOnceFullWriteEmptyPath(t *testing.T) {
	obj := &CsvWriterObj{}
	if err := obj.OnceFullWrite(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestOnceFullWriteManyRowsCancel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)

	obj := &CsvWriterObj{}
	records := make([][]string, 100)
	for i := range records {
		records[i] = []string{"x"}
	}
	_ = obj.OnceFullWrite(ctx, path, records)
}

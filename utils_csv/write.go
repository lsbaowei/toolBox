package utils_csv

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

type CsvWriterObj struct {
}

// OnceFullWrite 一次性 CSV 全量写入；ctx 取消时中止写入。
func (c *CsvWriterObj) OnceFullWrite(ctx context.Context, fs string, records [][]string) error {
	if fs == "" {
		return errors.New("fs is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	file, err := os.OpenFile(fs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return writer.Error()
}

// CSVWriter 封装CSV写入器
type CSVWriter struct {
	filename   string
	file       *os.File
	writer     *csv.Writer
	hasHeader  bool
	mu         sync.Mutex
	bufferSize int
	buffer     [][]string
	header     []string
}

// NewCSVWriter 创建CSV写入器
func NewCSVWriter(filename string, header []string) (*CSVWriter, error) {
	writer := &CSVWriter{
		filename:   filename,
		header:     header,
		bufferSize: 10, // 缓冲区大小
		buffer:     make([][]string, 0),
	}

	// 检查文件是否存在
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		// 文件不存在，创建并写入表头
		writer.hasHeader = false
	} else {
		// 文件已存在，不需要再写表头
		writer.hasHeader = true
	}

	// 打开文件（追加模式）
	writer.file, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	writer.writer = csv.NewWriter(writer.file)

	// 如果文件是新建的，写入表头
	if !writer.hasHeader {
		if err := writer.writer.Write(header); err != nil {
			return nil, err
		}
		writer.writer.Flush()
		writer.hasHeader = true
	}

	return writer, nil
}

// WriteRow 写入单行数据（带锁保护）
func (w *CSVWriter) WriteRow(row []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 添加到缓冲区
	w.buffer = append(w.buffer, row)

	// 如果缓冲区满了，写入文件
	if len(w.buffer) >= w.bufferSize {
		return w.flushBuffer()
	}

	return nil
}

// WriteData 写入结构化数据
func (w *CSVWriter) WriteData(data interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var row []string

	switch v := data.(type) {
	case []string:
		row = v
	case map[string]string:
		row = make([]string, len(w.header))
		for i, h := range w.header {
			if val, ok := v[h]; ok {
				row[i] = val
			}
		}
	default:
		// 尝试转换为JSON字符串
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("无法转换数据: %v", err)
		}
		row = []string{string(jsonBytes)}
	}

	w.buffer = append(w.buffer, row)

	if len(w.buffer) >= w.bufferSize {
		return w.flushBuffer()
	}

	return nil
}

// flushBuffer 将缓冲区数据写入文件
func (w *CSVWriter) flushBuffer() error {
	for _, row := range w.buffer {
		if err := w.writer.Write(row); err != nil {
			return err
		}
	}
	w.writer.Flush()

	// 清空缓冲区
	w.buffer = w.buffer[:0]

	return w.writer.Error()
}

// Flush 手动刷新缓冲区
func (w *CSVWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buffer) > 0 {
		return w.flushBuffer()
	}
	return nil
}

// Close 关闭文件
func (w *CSVWriter) Close() error {
	// 刷新剩余数据
	if err := w.Flush(); err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.writer.Flush()
	return w.file.Close()
}

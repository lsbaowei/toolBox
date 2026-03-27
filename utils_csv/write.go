package utils_csv

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type CsvWriterObj struct {
}

// OnceFullWrite 一次性CSV文件全量写入
func (c *CsvWriterObj) OnceFullWrite(ctx context.Context, fs string, records [][]string) error {
	if fs == "" {
		return errors.New("fs is empty")
	}

	// 创建 CSV 文件
	file, err := os.OpenFile(fs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// 创建 CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			log.Fatal(err)
		}
	}

	return nil
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

// 批量写入示例
func batchWriteExample(filename string, header []string, records [][]string, debug bool) error {
	//header := []string{"ID", "Name", "Price", "Stock", "LastUpdated"}

	writer, err := NewCSVWriter(filename, header)
	if err != nil {
		return err
	}
	defer writer.Close()

	// 批量写入
	for _, record := range records {
		if err := writer.WriteData(record); err != nil {
			return err
		}
		if debug {
			fmt.Printf("已写入产品: %v\n", record)
		}
	}

	if debug {
		fmt.Println("批量写入完成！")
	}

	return nil
}

// 增量追加示例
func incrementalWriteExample(filename string) error {
	header := []string{"ID", "Name", "Price", "Stock", "LastUpdated"}

	writer, err := NewCSVWriter(filename, header)
	if err != nil {
		return err
	}
	defer writer.Close()

	// 模拟增量数据流
	messages := []string{
		"开始处理订单...",
		"验证库存...",
		"计算价格...",
		"更新库存...",
		"生成物流单号...",
	}

	for i, msg := range messages {
		// 模拟实时数据
		data := map[string]string{
			"ID":          fmt.Sprintf("LOG%03d", i+1),
			"Name":        "System Log",
			"Price":       "0.00",
			"Stock":       "0",
			"LastUpdated": time.Now().Format("2006-01-02 15:04:05"),
		}

		if err := writer.WriteData(data); err != nil {
			return err
		}

		fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), msg)

		// 模拟处理延迟
		time.Sleep(1 * time.Second)
	}

	return nil
}

// 并发安全写入示例
func concurrentWriteExample(filename string) {
	header := []string{"Thread", "Message", "Timestamp"}

	writer, err := NewCSVWriter(filename, header)
	if err != nil {
		fmt.Printf("创建写入器失败: %v\n", err)
		return
	}
	defer writer.Close()

	var wg sync.WaitGroup

	// 启动多个goroutine同时写入
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			for j := 1; j <= 3; j++ {
				row := []string{
					fmt.Sprintf("Thread-%d", threadID),
					fmt.Sprintf("Message-%d from thread %d", j, threadID),
					time.Now().Format("2006-01-02 15:04:05.000"),
				}

				if err := writer.WriteRow(row); err != nil {
					fmt.Printf("线程 %d 写入失败: %v\n", threadID, err)
				} else {
					fmt.Printf("线程 %d 写入消息 %d\n", threadID, j)
				}

				time.Sleep(time.Millisecond * 100)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("并发写入完成！")
}

// 读取CSV内容（用于验证）
func readCSV(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	fmt.Printf("\n=== 读取CSV文件内容 ===\n")
	fmt.Printf("总行数: %d\n", len(records))
	for i, record := range records {
		fmt.Printf("行 %d: %v\n", i, record)
	}

	return nil
}

func main() {
	fmt.Println("=== CSV增量写入Demo ===")

	// 示例1: 批量写入
	fmt.Println("\n1. 批量写入示例")
	if err := batchWriteExample("products.csv", nil, nil, false); err != nil {
		fmt.Printf("批量写入失败: %v\n", err)
	}

	// 示例2: 增量追加
	fmt.Println("\n2. 增量追加示例")
	if err := incrementalWriteExample("logs.csv"); err != nil {
		fmt.Printf("增量写入失败: %v\n", err)
	}

	// 示例3: 并发写入
	fmt.Println("\n3. 并发写入示例")
	concurrentWriteExample("concurrent.csv")

	// 验证文件内容
	fmt.Println("\n4. 验证写入结果")

	files := []string{"products.csv", "logs.csv", "concurrent.csv"}
	for _, file := range files {
		fmt.Printf("\n文件: %s\n", file)
		if err := readCSV(file); err != nil {
			fmt.Printf("读取失败: %v\n", err)
		}
	}

	// 清理示例文件
	fmt.Println("\n5. 清理临时文件")
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			fmt.Printf("删除文件 %s 失败: %v\n", file, err)
		} else {
			fmt.Printf("已删除: %s\n", file)
		}
	}

	fmt.Println("\nDemo完成！")
}

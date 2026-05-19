// Demo for utils_csv. Run: go run ./cmd/csvdemo
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/lsbaowei/toolBox/utils_csv"
)

func main() {
	fmt.Println("=== CSV增量写入Demo ===")

	fmt.Println("\n1. 批量写入示例")
	if err := batchWriteExample("products.csv", []string{"ID", "Name"}, [][]string{{"1", "a"}}, false); err != nil {
		fmt.Printf("批量写入失败: %v\n", err)
	}

	fmt.Println("\n2. 增量追加示例")
	if err := incrementalWriteExample("logs.csv"); err != nil {
		fmt.Printf("增量写入失败: %v\n", err)
	}

	fmt.Println("\n3. 并发写入示例")
	concurrentWriteExample("concurrent.csv")

	fmt.Println("\n4. 验证写入结果")
	files := []string{"products.csv", "logs.csv", "concurrent.csv"}
	for _, file := range files {
		fmt.Printf("\n文件: %s\n", file)
		if err := readCSV(file); err != nil {
			fmt.Printf("读取失败: %v\n", err)
		}
	}

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

func batchWriteExample(filename string, header []string, records [][]string, debug bool) error {
	writer, err := utils_csv.NewCSVWriter(filename, header)
	if err != nil {
		return err
	}
	defer writer.Close()

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

func incrementalWriteExample(filename string) error {
	header := []string{"ID", "Name", "Price", "Stock", "LastUpdated"}
	writer, err := utils_csv.NewCSVWriter(filename, header)
	if err != nil {
		return err
	}
	defer writer.Close()

	messages := []string{"开始处理订单...", "验证库存...", "计算价格..."}
	for i, msg := range messages {
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
	}
	return nil
}

func concurrentWriteExample(filename string) {
	header := []string{"Thread", "Message", "Timestamp"}
	writer, err := utils_csv.NewCSVWriter(filename, header)
	if err != nil {
		fmt.Printf("创建写入器失败: %v\n", err)
		return
	}
	defer writer.Close()

	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			row := []string{
				fmt.Sprintf("Thread-%d", threadID),
				"msg",
				time.Now().Format("2006-01-02 15:04:05.000"),
			}
			_ = writer.WriteRow(row)
		}(i)
	}
	wg.Wait()
	fmt.Println("并发写入完成！")
}

func readCSV(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return err
	}
	fmt.Printf("总行数: %d\n", len(records))
	return nil
}

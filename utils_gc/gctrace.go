package utils_gc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func RejectGCTraceLog() error {
	// 开启 gctrace   GODEBUG=gctrace=1
	// _ = os.Setenv("GODEBUG", "gctrace=1")

	// 创建 pipe
	r, w, _ := os.Pipe()

	// 备份原始 fd
	origFd := int(os.Stderr.Fd())

	// 将 pipe 的写端复制到 fd 2（标准错误）
	if err := syscall.Dup2(int(w.Fd()), origFd); err != nil {
		fmt.Fprintf(os.Stderr, "dup2 failed: %v\n", err)
		return errors.New("dup2 failed: " + err.Error())
	}

	// goroutine：实时读取 gctrace 输出
	// 使用带缓冲的 channel 来避免管道积压
	traceChan := make(chan string, 100) // 缓冲 100 行，防止积压

	// 使用 sync.WaitGroup 确保读取完成
	var readDone sync.WaitGroup
	readDone.Add(1)

	// 读取 gctrace 输出到 channel，防止阻塞
	go func() {
		defer readDone.Done()
		scanner := bufio.NewScanner(r)
		// 增大缓冲区，防止超长 panic 堆栈被截断
		buf := make([]byte, 0, 64*1024) // 64KB 缓冲区
		scanner.Buffer(buf, 1024*1024)  // 最大 1MB，足够容纳完整的 panic 堆栈

		for scanner.Scan() {
			line := scanner.Text()
			// 非阻塞发送，如果 channel 满了就丢弃旧数据
			select {
			case traceChan <- line:
				// 成功发送
			default:
				// channel 满了，丢弃最旧的数据，放入新的
				select {
				case <-traceChan: // 移除最旧的数据
					traceChan <- line // 放入新数据
				default:
					// 如果还是失败，直接丢弃
				}
			}
		}
		// 检查是否有扫描错误（如缓冲区溢出）
		if err := scanner.Err(); err != nil {
			fmt.Printf("[SCANNER ERROR] %v\n", err)
		}
		close(traceChan)
	}()

	// 单独的处理 goroutine，避免阻塞读取
	go func() {
		for line := range traceChan {
			if strings.HasPrefix(line, "gc ") {
				// 在这里解析 gctrace
				// fmt.Println("[GC TRACE]", line)
				statsLog, err := NewGcTraceLog(line)
				if err != nil {
					fmt.Println("[GC TRACE ERROR]", err, line)
				} else {
					fmt.Println(statsLog)
				}
			} else {
				switch line {
				case "GC forced":
					// 触发了强制GC，写log
					fmt.Println("[GC]", line)
				default:
					// 捕获所有 stderr 输出，包括 panic/fatal
					fmt.Println("[STDERR]", line)
				}
			}
		}
	}()

	// 在程序退出前，确保读取完成（处理 panic 场景）
	defer func() {
		// 捕获 panic，确保能处理完 panic 输出
		panicValue := recover()

		if panicValue != nil {
			// 获取完整的堆栈信息
			// 使用 true 参数获取所有 goroutine 的堆栈，这样可以包含 panic 发生时的完整信息
			buf := make([]byte, 64*1024)
			n := runtime.Stack(buf, true)
			stackTrace := string(buf[:n])

			// 输出到 pipe（通过当前 stderr），会被读取 goroutine 捕获并显示为 [STDERR]
			fmt.Fprintf(os.Stderr, "\n=== PANIC RECOVERED ===\n")
			fmt.Fprintf(os.Stderr, "panic: %v\n\n", panicValue)
			fmt.Fprintf(os.Stderr, "%s\n", stackTrace)
			fmt.Fprintf(os.Stderr, "=== END PANIC ===\n\n")

			// 给一些时间让输出写入 pipe 并被读取
			time.Sleep(200 * time.Millisecond)
		}

		// 关闭写端，触发读取结束
		w.Close()

		// 使用带超时的等待，避免无限阻塞
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			readDone.Wait()
			close(done)
		}()

		select {
		case <-done:
			// 读取完成
		case <-ctx.Done():
			// 超时，不再等待（输出到 pipe，会被 [STDERR] 捕获）
			fmt.Fprintf(os.Stderr, "[WARNING] 读取 goroutine 超时，强制退出\n")
		}

		// 如果之前有 panic，重新 panic（让程序正常退出）
		// 注意：重新 panic 时堆栈会从 defer 开始，但我们已经输出了完整堆栈
		if panicValue != nil {
			panic(panicValue)
		}
	}()

	return nil
}

func GCTrace() {
	debug.SetGCPercent(100)
}

type GcTraceLog struct {
	GcTrace GCStatsD1 `json:"gc_trace,omitempty"`
}

func NewGcTraceLog(log string) (string, error) {

	stats, err := ParseGCLogHybridD1(log)
	if err != nil {
		return "", err
	}

	return (&GcTraceLog{
		GcTrace: *stats,
	}).JSONEncode(), nil
}

func (s *GcTraceLog) JSONEncode() string {
	b, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(b)
}

// GCStatsD1 存储解析后的GC统计信息
type GCStatsD1 struct {
	// 基础信息
	GCNumber   int     `json:"gc_number,omitempty"`   // GC次数: 25
	Uptime     float64 `json:"uptime,omitempty"`      // 程序运行时间: 59.374 (秒)
	CPUPercent float64 `json:"cpu_percent,omitempty"` // GC CPU占用: 0 (%)

	// 时钟时间（毫秒）
	ClockSTWScan  float64 `json:"clock_stw_scan,omitempty"`  // STW扫描时间: 0.26
	ClockMark     float64 `json:"clock_mark,omitempty"`      // 并发标记时间: 2.8
	ClockMarkTerm float64 `json:"clock_mark_term,omitempty"` // 标记终止时间: 0.036

	// CPU时间（毫秒）
	CPUSTWScan  float64 `json:"cpu_stw_scan,omitempty"`  // STW扫描CPU时间: 2.1
	CPUForced   float64 `json:"cpu_forced,omitempty"`    // 强制GC时间: 0
	CPUMark     float64 `json:"cpu_mark,omitempty"`      // 并发标记CPU时间: 2.6
	CPUAssist   float64 `json:"cpu_assist,omitempty"`    // 辅助GC时间: 0
	CPUMarkTerm float64 `json:"cpu_mark_term,omitempty"` // 标记终止CPU时间: 0.29

	// 内存信息（MB）
	HeapBefore int `json:"heap_before,omitempty"` // GC前堆大小: 971
	HeapAfter  int `json:"heap_after,omitempty"`  // GC后堆大小: 971
	HeapLive   int `json:"heap_live,omitempty"`   // 存活堆大小: 971
	HeapGoal   int `json:"heap_goal,omitempty"`   // 堆目标大小: 974
	Stacks     int `json:"stacks,omitempty"`      // 栈内存: 0
	Globals    int `json:"globals,omitempty"`     // 全局内存: 0

	// 处理器信息
	Procs int `json:"procs"` // P的数量: 8
}

// ParseGCLogHybrid 混合方案解析GC日志
func ParseGCLogHybridD1(log string) (*GCStatsD1, error) {
	if log == "" {
		return nil, fmt.Errorf("empty log")
	}

	// 1. 使用 strings.Fields 快速分割（按空白字符）
	fields := strings.Fields(log)
	if len(fields) < 20 {
		return nil, fmt.Errorf("invalid log format: too few fields")
	}

	stats := &GCStatsD1{}
	var err error

	// 2. 按位置解析各个字段（假设格式完全固定）
	// 格式: gc 25 @59.374s 0%: 0.26+2.8+0.036 ms clock, 2.1+0/2.6/0+0.29 ms cpu, ...

	// 字段索引映射（更清晰）
	const (
		idxGC           = 0  // "gc"
		idxGCNumber     = 1  // "25"
		idxUptime       = 2  // "@59.374s"
		idxCPUPercent   = 3  // "0%:"
		idxClockTimes   = 4  // "0.26+2.8+0.036"
		idxClockUnit    = 5  // "ms"
		idxClockLabel   = 6  // "clock,"
		idxCPUTimes     = 7  // "2.1+0/2.6/0+0.29"
		idxCPUUnit      = 8  // "ms"
		idxCPULabel     = 9  // "cpu,"
		idxHeapTriple   = 10 // "971->971->971"
		idxHeapUnit     = 11 // "MB,"
		idxHeapGoal     = 12 // "974"
		idxGoalUnit     = 13 // "MB"
		idxGoalLabel    = 14 // "goal,"
		idxStacks       = 15 // "0"
		idxStacksUnit   = 16 // "MB"
		idxStacksLabel  = 17 // "stacks,"
		idxGlobals      = 18 // "0"
		idxGlobalsUnit  = 19 // "MB"
		idxGlobalsLabel = 20 // "globals,"
		idxProcs        = 21 // "8"
		idxProcsLabel   = 22 // "P"
	)

	// 验证基本格式
	if fields[idxGC] != "gc" {
		return nil, fmt.Errorf("invalid log: missing 'gc' prefix")
	}

	// 3. 解析各个字段
	// 3.1 解析GC次数
	stats.GCNumber, err = strconv.Atoi(fields[idxGCNumber])
	if err != nil {
		return nil, fmt.Errorf("invalid GC number: %v", err)
	}

	// 3.2 解析运行时间（去掉 @ 和 s）
	uptimeStr := fields[idxUptime]
	if !strings.HasPrefix(uptimeStr, "@") || !strings.HasSuffix(uptimeStr, "s") {
		return nil, fmt.Errorf("invalid uptime format: %s", uptimeStr)
	}
	uptimeStr = strings.TrimPrefix(uptimeStr, "@")
	uptimeStr = strings.TrimSuffix(uptimeStr, "s")
	stats.Uptime, err = strconv.ParseFloat(uptimeStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid uptime value: %v", err)
	}

	// 3.3 解析CPU百分比（去掉 %:）
	cpuStr := fields[idxCPUPercent]
	if !strings.HasSuffix(cpuStr, "%:") {
		return nil, fmt.Errorf("invalid CPU percent format: %s", cpuStr)
	}
	cpuStr = strings.TrimSuffix(cpuStr, "%:")
	stats.CPUPercent, err = strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU percent: %v", err)
	}

	// 3.4 解析时钟时间（格式: 0.26+2.8+0.036）
	clockTimes := fields[idxClockTimes]
	clockParts := strings.Split(clockTimes, "+")
	if len(clockParts) != 3 {
		return nil, fmt.Errorf("invalid clock times format: %s", clockTimes)
	}

	stats.ClockSTWScan, err = strconv.ParseFloat(clockParts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid clock STW scan time: %v", err)
	}

	stats.ClockMark, err = strconv.ParseFloat(clockParts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid clock mark time: %v", err)
	}

	stats.ClockMarkTerm, err = strconv.ParseFloat(clockParts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid clock mark termination time: %v", err)
	}

	// 验证时钟时间单位
	if fields[idxClockUnit] != "ms" || fields[idxClockLabel] != "clock," {
		return nil, fmt.Errorf("invalid clock time unit or label")
	}

	// 3.5 解析CPU时间（格式: 2.1+0/2.6/0+0.29）
	cpuTimes := fields[idxCPUTimes]
	// 先按 + 分割，得到 2.1、0/2.6/0、0.29
	cpuTimeParts := strings.Split(cpuTimes, "+")
	if len(cpuTimeParts) != 3 {
		return nil, fmt.Errorf("invalid CPU times format: %s", cpuTimes)
	}

	// 解析 STW 扫描 CPU 时间
	stats.CPUSTWScan, err = strconv.ParseFloat(cpuTimeParts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU STW scan time: %v", err)
	}

	// 解析中间的 0/2.6/0
	middleParts := strings.Split(cpuTimeParts[1], "/")
	if len(middleParts) != 3 {
		return nil, fmt.Errorf("invalid CPU middle times format: %s", cpuTimeParts[1])
	}

	stats.CPUForced, err = strconv.ParseFloat(middleParts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU forced time: %v", err)
	}

	stats.CPUMark, err = strconv.ParseFloat(middleParts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU mark time: %v", err)
	}

	stats.CPUAssist, err = strconv.ParseFloat(middleParts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU assist time: %v", err)
	}

	// 解析标记终止 CPU 时间
	stats.CPUMarkTerm, err = strconv.ParseFloat(cpuTimeParts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU mark termination time: %v", err)
	}

	// 验证CPU时间单位
	if fields[idxCPUUnit] != "ms" || fields[idxCPULabel] != "cpu," {
		return nil, fmt.Errorf("invalid CPU time unit or label")
	}

	// 3.6 解析堆内存三元组（格式: 971->971->971）
	heapTriple := fields[idxHeapTriple]
	heapParts := strings.Split(heapTriple, "->")
	if len(heapParts) != 3 {
		return nil, fmt.Errorf("invalid heap triple format: %s", heapTriple)
	}

	stats.HeapBefore, err = strconv.Atoi(heapParts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid heap before size: %v", err)
	}

	stats.HeapAfter, err = strconv.Atoi(heapParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid heap after size: %v", err)
	}

	stats.HeapLive, err = strconv.Atoi(heapParts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid heap live size: %v", err)
	}

	// 验证堆内存单位
	if fields[idxHeapUnit] != "MB," {
		return nil, fmt.Errorf("invalid heap memory unit")
	}

	// 3.7 解析堆目标大小
	stats.HeapGoal, err = strconv.Atoi(fields[idxHeapGoal])
	if err != nil {
		return nil, fmt.Errorf("invalid heap goal: %v", err)
	}

	// 验证目标内存单位
	if fields[idxGoalUnit] != "MB" || fields[idxGoalLabel] != "goal," {
		return nil, fmt.Errorf("invalid heap goal unit or label")
	}

	// 3.8 解析栈内存
	stats.Stacks, err = strconv.Atoi(fields[idxStacks])
	if err != nil {
		return nil, fmt.Errorf("invalid stacks size: %v", err)
	}

	if fields[idxStacksUnit] != "MB" || fields[idxStacksLabel] != "stacks," {
		return nil, fmt.Errorf("invalid stacks unit or label")
	}

	// 3.9 解析全局内存
	stats.Globals, err = strconv.Atoi(fields[idxGlobals])
	if err != nil {
		return nil, fmt.Errorf("invalid globals size: %v", err)
	}

	if fields[idxGlobalsUnit] != "MB" || fields[idxGlobalsLabel] != "globals," {
		return nil, fmt.Errorf("invalid globals unit or label")
	}

	// 3.10 解析P的数量
	stats.Procs, err = strconv.Atoi(fields[idxProcs])
	if err != nil {
		return nil, fmt.Errorf("invalid procs count: %v", err)
	}

	if fields[idxProcsLabel] != "P" {
		return nil, fmt.Errorf("invalid procs label")
	}

	return stats, nil
}

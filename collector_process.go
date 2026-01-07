// ============================================
// 文件: collector_process.go
// 进程监控收集器
// ============================================
package main

import (
	"log"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessCollector 进程监控收集器
type ProcessCollector struct {
	maxProcesses int // 最多收集的进程数
}

// NewProcessCollector 创建进程收集器
func NewProcessCollector(maxProcesses int) *ProcessCollector {
	if maxProcesses <= 0 {
		maxProcesses = 50 // 默认最多50个进程
	}
	return &ProcessCollector{
		maxProcesses: maxProcesses,
	}
}

// Name 返回采集器名称
func (c *ProcessCollector) Name() string {
	return "process"
}

// Collect 采集进程信息
func (c *ProcessCollector) Collect() (interface{}, error) {
	processes, err := process.Processes()
	if err != nil {
		log.Printf("Failed to get process list: %v", err)
		return nil, err
	}

	log.Printf("Found %d total processes, collecting up to %d", len(processes), c.maxProcesses)

	var processList []ProcessInfo
	count := 0
	errorCount := 0

	for _, p := range processes {
		if count >= c.maxProcesses {
			break
		}

		info, err := c.getProcessInfo(p)
		if err != nil {
			errorCount++
			continue // 跳过无法获取信息的进程
		}

		processList = append(processList, *info)
		count++
	}

	log.Printf("Collected %d processes, %d errors", len(processList), errorCount)

	return &ProcessMetrics{
		Processes: processList,
		Total:     len(processes),
		Collected: len(processList),
	}, nil
}

// getProcessInfo 获取单个进程的详细信息
func (c *ProcessCollector) getProcessInfo(p *process.Process) (*ProcessInfo, error) {
	name, err := p.Name()
	if err != nil {
		return nil, err
	}
	
	username, _ := p.Username()
	cpuPercent, _ := p.CPUPercent()
	
	memInfo, err := p.MemoryInfo()
	var memoryBytes uint64 = 0
	if err == nil && memInfo != nil {
		memoryBytes = memInfo.RSS // 实际物理内存使用
	}
	
	memPercent, _ := p.MemoryPercent()
	createTime, _ := p.CreateTime()
	statusSlice, _ := p.Status()
	cmdline, _ := p.Cmdline()

	// 限制命令长度
	if len(cmdline) > 200 {
		cmdline = cmdline[:200] + "..."
	}

	// 处理status（可能是[]string）
	statusStr := ""
	if len(statusSlice) > 0 {
		statusStr = statusSlice[0]
	}

	// 转换memPercent为float64
	memPercentFloat64 := float64(memPercent)

	return &ProcessInfo{
		PID:          int32(p.Pid),
		Name:         name,
		User:         username,
		CPUPercent:   cpuPercent,
		MemoryPercent: memPercentFloat64,
		MemoryBytes:  memoryBytes,
		CreateTime:   createTime / 1000, // 转换为秒
		Status:       statusStr,
		Command:      cmdline,
	}, nil
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID          int32   `json:"pid"`
	Name         string  `json:"name"`
	User         string  `json:"user"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryBytes  uint64  `json:"memory_bytes"`
	CreateTime   int64   `json:"create_time"`
	Status       string  `json:"status"`
	Command      string  `json:"command"`
}

// ProcessMetrics 进程指标
type ProcessMetrics struct {
	Processes []ProcessInfo `json:"processes"`
	Total     int           `json:"total"`
	Collected int           `json:"collected"`
}


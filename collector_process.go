// ============================================
// 文件: collector_process.go
// 进程监控收集器
// ============================================
package main

import (
	"log"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
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

	log.Printf("Found %d total processes, collecting top %d by CPU usage", len(processes), c.maxProcesses)

	// 获取系统总内存，用于计算内存百分比
	vmStat, err := mem.VirtualMemory()
	totalMemory := uint64(0)
	if err == nil && vmStat != nil {
		totalMemory = vmStat.Total
	} else {
		log.Printf("Warning: Failed to get total memory: %v", err)
	}

	// 为了避免系统进程太多时速度太慢，设置一个合理的上限
	// 但为了确保能找到高CPU进程（包括新创建的进程），上限应该足够大
	// 如果进程总数较少，检查所有进程；如果进程总数较多，检查前2000个
	maxCheckProcesses := 2000
	if len(processes) < maxCheckProcesses {
		maxCheckProcesses = len(processes)
	}

	// 第一步：初始化进程的CPU统计（用于准确计算CPU使用率）
	// 第一次调用CPUPercent来初始化内部状态
	log.Printf("Initializing CPU stats for %d processes (out of %d total)...", maxCheckProcesses, len(processes))
	initCount := 0
	for i := 0; i < maxCheckProcesses; i++ {
		p := processes[i]
		_, err := p.CPUPercent()
		if err == nil {
			initCount++
		}
	}
	log.Printf("Initialized CPU stats for %d processes", initCount)

	// 等待1秒，让CPU统计有时间累积（与top命令类似，需要时间间隔来准确计算）
	time.Sleep(1 * time.Second)

	// 第二步：获取所有检查范围内进程的CPU使用率（经过1秒等待后的准确值）
	type processCPU struct {
		proc *process.Process
		cpu  float64
	}
	var processCPUList []processCPU
	
	log.Printf("Collecting CPU usage for %d processes...", maxCheckProcesses)
	collectedCount := 0
	maxCPU := 0.0
	for i := 0; i < maxCheckProcesses; i++ {
		p := processes[i]
		cpuPercent, err := p.CPUPercent()
		if err != nil {
			continue
		}
		
		// 收集所有进程用于排序（包括CPU为0的，但会按CPU使用率排序）
		processCPUList = append(processCPUList, processCPU{
			proc: p,
			cpu:  cpuPercent,
		})
		collectedCount++
		
		// 记录最大CPU使用率（用于日志输出）
		if cpuPercent > maxCPU {
			maxCPU = cpuPercent
		}
	}
	log.Printf("Collected CPU usage for %d processes (max CPU: %.2f%%)", collectedCount, maxCPU)

	// 第三步：按CPU使用率降序排序（使用Go标准库的快速排序）
	sort.Slice(processCPUList, func(i, j int) bool {
		return processCPUList[i].cpu > processCPUList[j].cpu
	})

	// 输出排序后的前5个进程的CPU使用率（用于调试）
	if len(processCPUList) > 0 {
		log.Printf("Top 5 processes by CPU usage:")
		for i := 0; i < 5 && i < len(processCPUList); i++ {
			pid := processCPUList[i].proc.Pid
			name, _ := processCPUList[i].proc.Name()
			log.Printf("  #%d: PID=%d, Name=%s, CPU=%.2f%%", i+1, pid, name, processCPUList[i].cpu)
		}
	}

	// 第四步：只取前maxProcesses个CPU使用率最高的进程
	if len(processCPUList) > c.maxProcesses {
		processCPUList = processCPUList[:c.maxProcesses]
	}

	// 第五步：获取每个进程的详细信息
	var processList []ProcessInfo
	errorCount := 0
	
	log.Printf("Getting detailed info for top %d processes...", len(processCPUList))
	for _, pc := range processCPUList {
		info, err := c.getProcessInfo(pc.proc, totalMemory)
		if err != nil {
			errorCount++
			log.Printf("Failed to get info for process: %v", err)
			continue // 跳过无法获取信息的进程
		}

		// 使用之前计算的CPU使用率（更准确）
		info.CPUPercent = pc.cpu

		processList = append(processList, *info)
	}

	log.Printf("Collected %d processes, %d errors", len(processList), errorCount)

	return &ProcessMetrics{
		Processes: processList,
		Total:     len(processes),
		Collected: len(processList),
	}, nil
}

// getProcessInfo 获取单个进程的详细信息
func (c *ProcessCollector) getProcessInfo(p *process.Process, totalMemory uint64) (*ProcessInfo, error) {
	name, err := p.Name()
	if err != nil {
		return nil, err
	}
	
	username, _ := p.Username()
	
	memInfo, err := p.MemoryInfo()
	var memoryBytes uint64 = 0
	var memoryPercent float64 = 0.0
	
	if err == nil && memInfo != nil {
		memoryBytes = memInfo.RSS // RSS: 实际物理内存使用（Resident Set Size）
		
		// 手动计算内存百分比（与top命令一致）
		// top命令使用 RSS / 总内存 * 100
		if totalMemory > 0 {
			memoryPercent = float64(memoryBytes) / float64(totalMemory) * 100.0
		} else {
			// 如果无法获取总内存，尝试使用MemoryPercent（可能不准确）
			if memPercent, err := p.MemoryPercent(); err == nil {
				memoryPercent = float64(memPercent)
			}
		}
	} else {
		// 如果MemoryInfo失败，尝试使用MemoryPercent作为备选
		if memPercent, err := p.MemoryPercent(); err == nil {
			memoryPercent = float64(memPercent)
		}
	}
	
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

	// CPU使用率将在Collect中通过时间间隔计算，这里先设为0
	return &ProcessInfo{
		PID:          int32(p.Pid),
		Name:         name,
		User:         username,
		CPUPercent:   0.0, // 将在Collect中设置
		MemoryPercent: memoryPercent,
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


// ============================================
// 文件: collector_memory.go
// ============================================
package main

import (
	"github.com/shirou/gopsutil/v3/mem"
)

// MemoryCollector 内存采集器
type MemoryCollector struct{}

// Name 返回采集器名称
func (c *MemoryCollector) Name() string {
	return "memory"
}

// Collect 采集内存指标
func (c *MemoryCollector) Collect() (interface{}, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	metrics := &MemoryMetrics{
		Total:       vmStat.Total,
		Used:        vmStat.Used,
		Free:        vmStat.Free,
		UsedPercent: vmStat.UsedPercent,
		Available:   vmStat.Available,
	}

	return metrics, nil
}

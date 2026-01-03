// ============================================
// 文件: collector_cpu.go
// ============================================
package main

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
)

// CPUCollector CPU采集器
type CPUCollector struct{}

// Name 返回采集器名称
func (c *CPUCollector) Name() string {
	return "cpu"
}

// Collect 采集CPU指标
func (c *CPUCollector) Collect() (interface{}, error) {
	// CPU使用率
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, err
	}

	// 负载平均值
	loadAvg, err := load.Avg()
	if err != nil {
		return nil, err
	}

	// CPU核心数
	coreCount, err := cpu.Counts(true)
	if err != nil {
		coreCount = 0
	}

	metrics := &CPUMetrics{
		UsagePercent: percent[0],
		LoadAvg1:     loadAvg.Load1,
		LoadAvg5:     loadAvg.Load5,
		LoadAvg15:    loadAvg.Load15,
		CoreCount:    coreCount,
	}

	return metrics, nil
}

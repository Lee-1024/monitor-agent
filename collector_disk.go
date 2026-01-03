// ============================================
// 文件: collector_disk.go
// ============================================
package main

import (
	"github.com/shirou/gopsutil/v3/disk"
)

// DiskCollector 磁盘采集器
type DiskCollector struct{}

// Name 返回采集器名称
func (c *DiskCollector) Name() string {
	return "disk"
}

// Collect 采集磁盘指标
func (c *DiskCollector) Collect() (interface{}, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	metrics := &DiskMetrics{
		Partitions: make([]PartitionMetrics, 0),
	}

	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		pm := PartitionMetrics{
			Device:      partition.Device,
			Mountpoint:  partition.Mountpoint,
			Fstype:      partition.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		}
		metrics.Partitions = append(metrics.Partitions, pm)
	}

	return metrics, nil
}

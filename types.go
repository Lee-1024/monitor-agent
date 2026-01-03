// ============================================
// 文件: types.go
// ============================================
package main

// MetricsData 指标数据
type MetricsData struct {
	HostID    string                 `json:"host_id"`
	Timestamp int64                  `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// Collector 采集器接口
type Collector interface {
	Name() string
	Collect() (interface{}, error)
}

// CPUMetrics CPU指标
type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
	LoadAvg1     float64 `json:"load_avg_1"`
	LoadAvg5     float64 `json:"load_avg_5"`
	LoadAvg15    float64 `json:"load_avg_15"`
	CoreCount    int     `json:"core_count"`
}

// MemoryMetrics 内存指标
type MemoryMetrics struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	Available   uint64  `json:"available"`
}

// DiskMetrics 磁盘指标
type DiskMetrics struct {
	Partitions []PartitionMetrics `json:"partitions"`
}

// PartitionMetrics 分区指标
type PartitionMetrics struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkMetrics 网络指标
type NetworkMetrics struct {
	Interfaces []InterfaceMetrics `json:"interfaces"`
}

// InterfaceMetrics 网卡指标
type InterfaceMetrics struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	Errin       uint64 `json:"errin"`
	Errout      uint64 `json:"errout"`
}

// ============================================
// 文件: collector_network.go
// ============================================
package main

import (
	"github.com/shirou/gopsutil/v3/net"
)

// NetworkCollector 网络采集器
type NetworkCollector struct{}

// Name 返回采集器名称
func (c *NetworkCollector) Name() string {
	return "network"
}

// Collect 采集网络指标
func (c *NetworkCollector) Collect() (interface{}, error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	metrics := &NetworkMetrics{
		Interfaces: make([]InterfaceMetrics, 0),
	}

	for _, io := range ioCounters {
		// 过滤lo接口
		if io.Name == "lo" {
			continue
		}

		im := InterfaceMetrics{
			Name:        io.Name,
			BytesSent:   io.BytesSent,
			BytesRecv:   io.BytesRecv,
			PacketsSent: io.PacketsSent,
			PacketsRecv: io.PacketsRecv,
			Errin:       io.Errin,
			Errout:      io.Errout,
		}
		metrics.Interfaces = append(metrics.Interfaces, im)
	}

	return metrics, nil
}

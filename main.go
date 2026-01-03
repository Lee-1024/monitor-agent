// ============================================
// 文件: main.go (修改版)
// ============================================
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"
)

type Agent struct {
	HostID          string
	CollectInterval time.Duration
	collectors      []Collector
	reporter        *Reporter
}

func NewAgent(hostID string, interval time.Duration, reporter *Reporter) *Agent {
	return &Agent{
		HostID:          hostID,
		CollectInterval: interval,
		reporter:        reporter,
		collectors: []Collector{
			&CPUCollector{},
			&MemoryCollector{},
			&DiskCollector{},
			&NetworkCollector{},
		},
	}
}

func (a *Agent) Start() {
	log.Printf("Agent started, HostID: %s, Interval: %v\n", a.HostID, a.CollectInterval)

	ticker := time.NewTicker(a.CollectInterval)
	defer ticker.Stop()

	// 心跳ticker
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := a.collectMetrics()
			a.reportMetrics(metrics)
		case <-heartbeatTicker.C:
			if a.reporter != nil {
				a.reporter.SendHeartbeat()
			}
		}
	}
}

func (a *Agent) collectMetrics() *MetricsData {
	data := &MetricsData{
		HostID:    a.HostID,
		Timestamp: time.Now().Unix(),
		Metrics:   make(map[string]interface{}),
	}

	for _, collector := range a.collectors {
		metrics, err := collector.Collect()
		if err != nil {
			log.Printf("Error collecting %s: %v\n", collector.Name(), err)
			continue
		}
		data.Metrics[collector.Name()] = metrics
	}

	return data
}

func (a *Agent) reportMetrics(data *MetricsData) {
	// 如果配置了reporter，通过gRPC上报
	if a.reporter != nil {
		if err := a.reporter.Report(data); err != nil {
			log.Printf("Failed to report via gRPC: %v", err)
		} else {
			log.Printf("Metrics reported successfully to server")
		}
	} else {
		// 否则输出到控制台（调试模式）
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Printf("\n=== Metrics Report ===\n%s\n", string(jsonData))
	}
}

func main() {
	// 命令行参数
	serverAddr := flag.String("server", "", "Collector server address (e.g. localhost:50051)")
	hostID := flag.String("host-id", "host-001", "Host ID")
	interval := flag.Int("interval", 10, "Collect interval in seconds")
	debug := flag.Bool("debug", false, "Debug mode (print to console)")
	flag.Parse()

	var reporter *Reporter
	var err error

	// 如果指定了服务器地址且非调试模式，则使用gRPC上报
	if *serverAddr != "" && !*debug {
		reporter, err = NewReporter(*serverAddr, *hostID)
		if err != nil {
			log.Fatalf("Failed to create reporter: %v", err)
		}
		defer reporter.Close()
	} else {
		log.Println("Running in debug mode, metrics will be printed to console")
	}

	agent := NewAgent(*hostID, time.Duration(*interval)*time.Second, reporter)
	agent.Start()
}

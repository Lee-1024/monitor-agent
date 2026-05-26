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
	logCollector    *LogCollector // 独立的日志收集器
}

func NewAgent(hostID string, interval time.Duration, reporter *Reporter, config *AgentConfig) *Agent {
	// 基础收集器
	collectors := []Collector{
		&CPUCollector{},
		&MemoryCollector{},
		&DiskCollector{},
		&NetworkCollector{},
	}

	// 进程监控收集器
	processCollector := NewProcessCollector(50) // 最多收集50个进程
	collectors = append(collectors, processCollector)

	var logCollector *LogCollector
	if config.LogCollectionEnabled() {
		log.Printf("Loaded %d log paths from config", len(config.LogPaths))
		logCollector = NewLogCollector(config.LogPaths, 100) // 每个文件最多100行
		collectors = append(collectors, logCollector)
	} else {
		log.Printf("No log paths configured, log collection disabled")
	}

	// 脚本执行器（从配置读取脚本列表）
	if len(config.Scripts) > 0 {
		log.Printf("Loaded %d scripts from config", len(config.Scripts))
		scriptExecutor := NewScriptExecutor(config.Scripts)
		collectors = append(collectors, scriptExecutor)
	} else {
		log.Printf("No scripts configured")
	}

	// 服务状态检测器（从配置读取服务列表）
	// 优先使用新的服务端口配置（支持端口检查）
	var serviceCollector *ServiceCollector
	if len(config.ServicePorts) > 0 {
		log.Printf("Loaded %d service ports from config", len(config.ServicePorts))
		// 转换配置类型
		ports := make([]ServicePortConfig, len(config.ServicePorts))
		for i, p := range config.ServicePorts {
			ports[i] = ServicePortConfig{
				Name:        p.Name,
				Port:        p.Port,
				Host:        p.Host,
				Description: p.Description,
			}
		}
		serviceCollector = NewServiceCollectorWithPorts(ports)
	} else if len(config.Services) > 0 {
		// 兼容旧格式
		log.Printf("Loaded %d services from config (legacy format)", len(config.Services))
		serviceCollector = NewServiceCollector(config.Services)
	} else {
		// 使用默认服务
		log.Printf("No services configured, service monitoring disabled")
		serviceCollector = NewServiceCollector(nil)
	}
	collectors = append(collectors, serviceCollector)

	return &Agent{
		HostID:          hostID,
		CollectInterval: interval,
		reporter:        reporter,
		collectors:      collectors,
		logCollector:    logCollector,
	}
}

func (a *Agent) Start() {
	log.Printf("Agent started, HostID: %s, Interval: %v\n", a.HostID, a.CollectInterval)

	ticker := time.NewTicker(a.CollectInterval)
	defer ticker.Stop()

	// 心跳ticker
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	// 日志采集ticker（独立间隔，默认1分钟）
	logTicker := time.NewTicker(60 * time.Second) // 日志采集间隔改为1分钟
	defer logTicker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := a.collectMetrics()
			// 上报额外的监控数据（进程、脚本、服务，不包括日志）
			a.reportCoreMetrics(metrics)
			a.reportMetrics(metrics)
		case <-logTicker.C:
			// 单独采集和上报日志
			a.collectAndReportLogs()
		case <-heartbeatTicker.C:
			if a.reporter != nil {
				a.reporter.SendHeartbeat()
			}
		}
	}
}

// reportCoreMetrics 上报核心监控数据（不包括日志）
func (a *Agent) reportCoreMetrics(metrics *MetricsData) {
	if a.reporter == nil {
		return
	}

	// 上报进程数据
	if processData, ok := metrics.Metrics["process"].(*ProcessMetrics); ok {
		log.Printf("Reporting %d processes to server", len(processData.Processes))
		if err := a.reporter.ReportProcesses(processData); err != nil {
			log.Printf("Failed to report processes: %v", err)
		} else {
			log.Printf("Successfully reported %d processes", len(processData.Processes))
		}
	}

	// 上报脚本执行结果
	if scriptData, ok := metrics.Metrics["script"].(*ScriptMetrics); ok {
		if err := a.reporter.ReportScriptResults(scriptData); err != nil {
			log.Printf("Failed to report script results: %v", err)
		}
	}

	if serviceData, ok := metrics.Metrics["service"].(*ServiceMetrics); ok {
		if len(serviceData.Services) > 0 {
			if err := a.reporter.ReportServiceStatus(serviceData); err != nil {
				log.Printf("Failed to report service status: %v", err)
			}
		}
	}
}

// collectAndReportLogs 单独采集和上报日志
func (a *Agent) collectAndReportLogs() {
	if a.reporter == nil || a.logCollector == nil {
		return
	}

	// 采集日志
	logData, err := a.logCollector.Collect()
	if err != nil {
		log.Printf("Failed to collect logs: %v", err)
		return
	}

	// 上报日志
	if logMetrics, ok := logData.(*LogMetrics); ok {
		if len(logMetrics.Entries) > 0 {
			log.Printf("Reporting %d log entries to server", len(logMetrics.Entries))
			if err := a.reporter.ReportLogs(logMetrics); err != nil {
				log.Printf("Failed to report logs: %v", err)
			} else {
				log.Printf("Successfully reported %d log entries", len(logMetrics.Entries))
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

// reportAdditionalMetrics 上报额外的监控数据（进程、日志、脚本、服务）
func (a *Agent) reportAdditionalMetrics(metrics *MetricsData) {
	if a.reporter == nil {
		return
	}

	// 上报进程数据
	if processData, ok := metrics.Metrics["process"].(*ProcessMetrics); ok {
		log.Printf("Reporting %d processes to server", len(processData.Processes))
		if err := a.reporter.ReportProcesses(processData); err != nil {
			log.Printf("Failed to report processes: %v", err)
		} else {
			log.Printf("Successfully reported %d processes", len(processData.Processes))
		}
	} else {
		log.Printf("No process data found in metrics (type: %T, value: %v)", metrics.Metrics["process"], metrics.Metrics["process"])
	}

	// 上报日志数据
	if logData, ok := metrics.Metrics["log"].(*LogMetrics); ok {
		if err := a.reporter.ReportLogs(logData); err != nil {
			log.Printf("Failed to report logs: %v", err)
		}
	}

	// 上报脚本执行结果
	if scriptData, ok := metrics.Metrics["script"].(*ScriptMetrics); ok {
		if err := a.reporter.ReportScriptResults(scriptData); err != nil {
			log.Printf("Failed to report script results: %v", err)
		}
	}

	// 上报服务状态
	if serviceData, ok := metrics.Metrics["service"].(*ServiceMetrics); ok {
		if err := a.reporter.ReportServiceStatus(serviceData); err != nil {
			log.Printf("Failed to report service status: %v", err)
		}
	}
}

func main() {
	// 命令行参数
	serverAddr := flag.String("server", "", "Collector server address (e.g. localhost:50051)")
	hostID := flag.String("host-id", "host-001", "Host ID")
	interval := flag.Int("interval", 10, "Collect interval in seconds")
	debug := flag.Bool("debug", false, "Debug mode (print to console)")
	configPath := flag.String("config", ConfigPathFromEnv(), "Config file path")
	flag.Parse()

	config := LoadAgentConfigFromPath(*configPath)
	applyFlagOverrides(config, serverAddr, hostID, interval, debug)

	var reporter *Reporter
	var err error

	// 如果指定了服务器地址且非调试模式，则使用gRPC上报
	if *serverAddr != "" && !*debug {
		reporter, err = NewReporterWithConfig(*serverAddr, *hostID, config)
		if err != nil {
			log.Fatalf("Failed to create reporter: %v", err)
		}
		defer reporter.Close()
	} else {
		log.Println("Running in debug mode, metrics will be printed to console")
	}

	agent := NewAgent(*hostID, time.Duration(*interval)*time.Second, reporter, config)
	agent.Start()
}

func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func applyFlagOverrides(config *AgentConfig, serverAddr *string, hostID *string, interval *int, debug *bool) {
	if !isFlagSet("server") {
		*serverAddr = config.ServerAddr
	}
	if !isFlagSet("host-id") {
		*hostID = config.HostID
	}
	if !isFlagSet("interval") {
		*interval = config.CollectInterval
	}
	if !isFlagSet("debug") {
		*debug = config.Debug
	}
}

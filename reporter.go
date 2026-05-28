// ============================================
// 文件: reporter.go (新增)
// ============================================
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	pb "monitor-agent/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Reporter gRPC上报器
type Reporter struct {
	client     pb.CollectorClient
	conn       *grpc.ClientConn
	serverAddr string
	hostID     string
	config     *AgentConfig
	registered bool
	http       *HTTPReporter
	cache      *MetricCache
	grpcReady  bool
}

// NewReporter 创建上报器
func NewReporter(serverAddr, hostID string) (*Reporter, error) {
	return NewReporterWithConfig(serverAddr, hostID, LoadAgentConfig())
}

func NewReporterWithConfig(serverAddr, hostID string, config *AgentConfig) (*Reporter, error) {
	reporter := &Reporter{
		serverAddr: serverAddr,
		hostID:     hostID,
		config:     config,
		registered: false,
	}
	reporter.initFallback()

	connectTimeout := timeoutSeconds(config.GRPC.ConnectTimeout, 5)
	// 建立gRPC连接
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		if reporter.http == nil {
			return nil, fmt.Errorf("connect to gRPC server %s timeout after %s: %w", serverAddr, connectTimeout, err)
		}
		log.Printf("gRPC reporter unavailable, switching to HTTP fallback: %v", err)
		if err := reporter.register(); err != nil {
			return nil, err
		}
		return reporter, nil
	}

	reporter.client = pb.NewCollectorClient(conn)
	reporter.conn = conn
	reporter.grpcReady = true

	// 注册Agent
	if err := reporter.register(); err != nil {
		log.Printf("Failed to register agent: %v", err)
		return nil, err
	}

	return reporter, nil
}

func (r *Reporter) initFallback() {
	if r.config == nil {
		return
	}
	if r.config.Fallback.HTTPEnabled && r.config.Fallback.HTTPBaseURL != "" {
		r.http = NewHTTPReporter(r.config.Fallback.HTTPBaseURL, timeoutSeconds(r.config.GRPC.RequestTimeout, 10))
	}
	if r.config.Fallback.CacheEnabled {
		r.cache = NewMetricCache(r.config.Fallback.CacheDir, r.config.Fallback.MaxCacheFiles)
	}
}

// register 注册Agent
func (r *Reporter) register() error {
	systemHostname, _ := os.Hostname()
	hostname := r.config.EffectiveHostname(systemHostname)
	// 读取配置
	ip := r.getIPAddress()

	req := &pb.RegisterRequest{
		HostId:   r.hostID,
		Hostname: hostname,
		Ip:       ip,
		Os:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Tags: map[string]string{
			"env": "production",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds(r.config.GRPC.RegisterTimeout, 5))
	defer cancel()

	var err error
	var resp *pb.RegisterResponse
	if r.grpcReady && r.client != nil {
		resp, err = r.client.RegisterAgent(ctx, req)
		if err != nil {
			log.Printf("Failed to register via gRPC: %v", err)
		}
	}
	if (resp == nil || err != nil) && r.http != nil {
		err = r.http.Register(ctx, req)
		if err == nil {
			r.registered = true
			log.Printf("Agent registered successfully via HTTP fallback")
			return nil
		}
	}
	if err != nil {
		return err
	}

	if resp.Success {
		r.registered = true
		log.Printf("Agent registered successfully: %s", resp.Message)
	} else {
		log.Printf("Agent registration failed: %s", resp.Message)
	}

	return nil
}

// Report 上报指标数据
func (r *Reporter) Report(data *MetricsData) error {
	if !r.registered {
		return nil
	}

	// 转换为protobuf格式
	req := r.metricsRequest(data)

	if r.cache != nil {
		if flushed, err := r.cache.Flush(r.sendMetricsRequest); err != nil {
			log.Printf("Failed to flush metric cache: %v", err)
		} else if flushed > 0 {
			log.Printf("Flushed %d cached metric reports", flushed)
		}
	}

	if err := r.sendMetricsRequest(req); err != nil {
		if r.cache != nil {
			if cacheErr := r.cache.Store(req); cacheErr != nil {
				log.Printf("Failed to cache metrics: %v", cacheErr)
			} else {
				log.Printf("Metrics cached locally after report failure")
			}
		}
		return err
	}

	return nil
}

func (r *Reporter) metricsRequest(data *MetricsData) *pb.MetricsRequest {
	req := &pb.MetricsRequest{
		HostId:    data.HostID,
		Timestamp: data.Timestamp,
	}

	// CPU指标
	if cpu, ok := data.Metrics["cpu"].(*CPUMetrics); ok {
		req.Cpu = &pb.CPUMetrics{
			UsagePercent: cpu.UsagePercent,
			LoadAvg_1:    cpu.LoadAvg1,
			LoadAvg_5:    cpu.LoadAvg5,
			LoadAvg_15:   cpu.LoadAvg15,
			CoreCount:    int32(cpu.CoreCount),
		}
	}

	// 内存指标
	if mem, ok := data.Metrics["memory"].(*MemoryMetrics); ok {
		req.Memory = &pb.MemoryMetrics{
			Total:       mem.Total,
			Used:        mem.Used,
			Free:        mem.Free,
			UsedPercent: mem.UsedPercent,
			Available:   mem.Available,
		}
	}

	// 磁盘指标
	if disk, ok := data.Metrics["disk"].(*DiskMetrics); ok {
		diskMetrics := &pb.DiskMetrics{
			Partitions: make([]*pb.PartitionMetrics, 0),
		}
		for _, p := range disk.Partitions {
			diskMetrics.Partitions = append(diskMetrics.Partitions, &pb.PartitionMetrics{
				Device:      p.Device,
				Mountpoint:  p.Mountpoint,
				Fstype:      p.Fstype,
				Total:       p.Total,
				Used:        p.Used,
				Free:        p.Free,
				UsedPercent: p.UsedPercent,
			})
		}
		req.Disk = diskMetrics
	}

	// 网络指标
	if net, ok := data.Metrics["network"].(*NetworkMetrics); ok {
		netMetrics := &pb.NetworkMetrics{
			Interfaces: make([]*pb.InterfaceMetrics, 0),
		}
		for _, iface := range net.Interfaces {
			netMetrics.Interfaces = append(netMetrics.Interfaces, &pb.InterfaceMetrics{
				Name:        iface.Name,
				BytesSent:   iface.BytesSent,
				BytesRecv:   iface.BytesRecv,
				PacketsSent: iface.PacketsSent,
				PacketsRecv: iface.PacketsRecv,
				Errin:       iface.Errin,
				Errout:      iface.Errout,
			})
		}
		req.Network = netMetrics
	}

	if gpu, ok := data.Metrics["gpu"].(*GPUMetrics); ok {
		gpuMetrics := &pb.GPUMetrics{
			Devices: make([]*pb.GPUDeviceMetrics, 0, len(gpu.Devices)),
		}
		for _, device := range gpu.Devices {
			gpuMetrics.Devices = append(gpuMetrics.Devices, &pb.GPUDeviceMetrics{
				Index:              int32(device.Index),
				Name:               device.Name,
				Vendor:             device.Vendor,
				Model:              device.Model,
				Uuid:               device.UUID,
				DriverVersion:      device.DriverVersion,
				UtilizationPercent: device.UtilizationPercent,
				MemoryTotal:        device.MemoryTotal,
				MemoryUsed:         device.MemoryUsed,
				MemoryUsedPercent:  device.MemoryUsedPercent,
				Temperature:        device.Temperature,
				PowerWatts:         device.PowerWatts,
				FanSpeedPercent:    device.FanSpeedPercent,
			})
		}
		req.Gpu = gpuMetrics
	}

	return req
}

func (r *Reporter) sendMetricsRequest(req *pb.MetricsRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds(r.config.GRPC.ReportTimeout, 5))
	defer cancel()

	if r.grpcReady && r.client != nil {
		resp, err := r.client.ReportMetrics(ctx, req)
		if err == nil {
			if !resp.Success {
				log.Printf("Server rejected metrics: %s", resp.Message)
			}
			return nil
		}
		log.Printf("Failed to report metrics via gRPC: %v", err)
	}

	if r.http != nil {
		if err := r.http.ReportMetrics(ctx, req); err != nil {
			return err
		}
		log.Printf("Metrics reported successfully via HTTP fallback")
		return nil
	}
	return fmt.Errorf("no metrics reporter available")
}

// SendHeartbeat 发送心跳
func (r *Reporter) SendHeartbeat() error {
	if !r.registered {
		return nil
	}

	req := &pb.HeartbeatRequest{
		HostId:    r.hostID,
		Timestamp: time.Now().Unix(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds(r.config.GRPC.HeartbeatTimeout, 3))
	defer cancel()

	if r.grpcReady && r.client != nil {
		_, err := r.client.Heartbeat(ctx, req)
		if err == nil {
			return nil
		}
		log.Printf("Heartbeat via gRPC failed: %v", err)
	}

	if r.http != nil {
		return r.http.Heartbeat(ctx, req)
	}

	return fmt.Errorf("no heartbeat reporter available")
}

// ReportProcesses 上报进程监控数据
func (r *Reporter) ReportProcesses(data *ProcessMetrics) error {
	if !r.registered {
		log.Printf("Reporter not registered, skipping process report")
		return nil
	}

	if data == nil || len(data.Processes) == 0 {
		log.Printf("No process data to report")
		return nil
	}

	req := &pb.ProcessReportRequest{
		HostId:    r.hostID,
		Timestamp: time.Now().Unix(),
		Processes: make([]*pb.ProcessInfo, 0, len(data.Processes)),
	}

	for _, p := range data.Processes {
		req.Processes = append(req.Processes, &pb.ProcessInfo{
			Pid:           p.PID,
			Name:          p.Name,
			User:          p.User,
			CpuPercent:    p.CPUPercent,
			MemoryPercent: p.MemoryPercent,
			MemoryBytes:   p.MemoryBytes,
			CreateTime:    p.CreateTime,
			Status:        p.Status,
			Command:       p.Command,
		})
	}

	log.Printf("Sending %d processes to server", len(req.Processes))

	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds(r.config.GRPC.RequestTimeout, 10))
	defer cancel()

	resp, err := r.client.ReportProcesses(ctx, req)
	if err != nil {
		log.Printf("Failed to send process data: %v", err)
		return err
	}

	if resp != nil && resp.Success {
		log.Printf("Process data reported successfully: %s", resp.Message)
	} else {
		log.Printf("Server response: success=%v, message=%s", resp != nil && resp.Success, resp.GetMessage())
	}

	return nil
}

// ReportLogs 上报日志数据
func (r *Reporter) ReportLogs(data *LogMetrics) error {
	if !r.registered {
		return nil
	}

	req := &pb.LogReportRequest{
		HostId:    r.hostID,
		Timestamp: time.Now().Unix(),
		Logs:      make([]*pb.LogEntry, 0),
	}

	for _, log := range data.Entries {
		req.Logs = append(req.Logs, &pb.LogEntry{
			Source:    log.Source,
			Level:     log.Level,
			Message:   log.Message,
			Timestamp: log.Timestamp,
			Tags:      log.Tags,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds(r.config.GRPC.RequestTimeout, 10))
	defer cancel()

	_, err := r.client.ReportLogs(ctx, req)
	return err
}

// ReportScriptResults 上报脚本执行结果
func (r *Reporter) ReportScriptResults(data *ScriptMetrics) error {
	if !r.registered {
		return nil
	}

	for _, result := range data.Results {
		req := &pb.ScriptResultRequest{
			HostId:     r.hostID,
			ScriptId:   result.ScriptID,
			ScriptName: result.ScriptName,
			Timestamp:  result.Timestamp,
			Success:    result.Success,
			Output:     result.Output,
			Error:      result.Error,
			ExitCode:   int32(result.ExitCode),
			DurationMs: result.Duration,
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds(r.config.GRPC.RequestTimeout, 10))
		_, err := r.client.ReportScriptResult(ctx, req)
		cancel()

		if err != nil {
			log.Printf("Failed to report script result %s: %v", result.ScriptID, err)
		}
	}

	return nil
}

// ReportServiceStatus 上报服务状态
func (r *Reporter) ReportServiceStatus(data *ServiceMetrics) error {
	if !r.registered {
		log.Printf("Reporter not registered, skipping service status report")
		return nil
	}

	if data == nil || len(data.Services) == 0 {
		log.Printf("No service status data to report")
		return nil
	}

	req := &pb.ServiceStatusRequest{
		HostId:    r.hostID,
		Timestamp: time.Now().Unix(),
		Services:  make([]*pb.ServiceInfo, 0, len(data.Services)),
	}

	for _, s := range data.Services {
		svcInfo := &pb.ServiceInfo{
			Name:          s.Name,
			Status:        s.Status,
			Enabled:       s.Enabled,
			Description:   s.Description,
			UptimeSeconds: s.Uptime,
		}
		// 如果有端口信息，添加端口和端口检查结果
		if s.Port > 0 {
			svcInfo.Port = int32(s.Port)
			svcInfo.PortAccessible = s.PortAccessible
		}
		req.Services = append(req.Services, svcInfo)
	}

	log.Printf("Sending %d service statuses to server", len(req.Services))

	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds(r.config.GRPC.RequestTimeout, 10))
	defer cancel()

	resp, err := r.client.ReportServiceStatus(ctx, req)
	if err != nil {
		log.Printf("Failed to send service status data: %v", err)
		return err
	}

	if resp != nil && resp.Success {
		log.Printf("Service status reported successfully: %s", resp.Message)
	} else {
		log.Printf("Server rejected service status: success=%v, message=%s", resp != nil && resp.Success, resp.GetMessage())
	}

	return err
}

func (r *Reporter) ReportDockerContainers(data *DockerMetrics) error {
	if !r.registered {
		return nil
	}
	if data == nil {
		return nil
	}

	req := &pb.LogReportRequest{
		HostId:    r.hostID,
		Timestamp: time.Now().Unix(),
		Logs:      make([]*pb.LogEntry, 0, len(data.Containers)),
	}
	for _, container := range data.Containers {
		payload, err := json.Marshal(container)
		if err != nil {
			log.Printf("Failed to encode docker container %s: %v", container.Name, err)
			continue
		}
		req.Logs = append(req.Logs, &pb.LogEntry{
			Source:    "docker",
			Level:     "INFO",
			Message:   string(payload),
			Timestamp: req.Timestamp,
			Tags: map[string]string{
				"container_id": container.ContainerID,
				"name":         container.Name,
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds(r.config.GRPC.RequestTimeout, 10))
	defer cancel()
	resp, err := r.client.ReportDockerContainers(ctx, req)
	if err != nil {
		log.Printf("Failed to send docker container data: %v", err)
		return err
	}
	if resp != nil && !resp.Success {
		log.Printf("Server rejected docker container data: %s", resp.Message)
	}
	return nil
}

// Close 关闭连接
func (r *Reporter) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
}

// getLocalIP 获取本机真实IP
func getLocalIP() string {
	// 方法1：通过连接外部地址获取（最准确）
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Printf("Failed to get IP by dial: %v", err)

		// 方法2：遍历网络接口
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return "127.0.0.1"
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					return ipNet.IP.String()
				}
			}
		}

		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func (r *Reporter) getIPAddress() string {
	config := r.config

	if config.ManualIP != "" {
		log.Printf("Using manual IP from config: %s", config.ManualIP)
		return config.ManualIP
	}

	ip := getLocalIP()
	log.Printf("Auto-detected IP: %s", ip)
	return ip
}

func timeoutSeconds(seconds int, fallback int) time.Duration {
	if seconds <= 0 {
		seconds = fallback
	}
	return time.Duration(seconds) * time.Second
}

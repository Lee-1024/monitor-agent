// ============================================
// 文件: reporter.go (新增)
// ============================================
package main

import (
	"context"
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
	registered bool
}

// NewReporter 创建上报器
func NewReporter(serverAddr, hostID string) (*Reporter, error) {
	// 建立gRPC连接
	conn, err := grpc.Dial(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, err
	}

	client := pb.NewCollectorClient(conn)

	reporter := &Reporter{
		client:     client,
		conn:       conn,
		serverAddr: serverAddr,
		hostID:     hostID,
		registered: false,
	}

	// 注册Agent
	if err := reporter.register(); err != nil {
		log.Printf("Failed to register agent: %v", err)
		return nil, err
	}

	return reporter, nil
}

// register 注册Agent
func (r *Reporter) register() error {
	hostname, _ := os.Hostname()
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := r.client.RegisterAgent(ctx, req)
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

	// 发送请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := r.client.ReportMetrics(ctx, req)
	if err != nil {
		log.Printf("Failed to report metrics: %v", err)
		return err
	}

	if !resp.Success {
		log.Printf("Server rejected metrics: %s", resp.Message)
	}

	return nil
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.client.Heartbeat(ctx, req)
	if err != nil {
		log.Printf("Heartbeat failed: %v", err)
		return err
	}

	return nil
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
			Pid:          p.PID,
			Name:         p.Name,
			User:         p.User,
			CpuPercent:   p.CPUPercent,
			MemoryPercent: p.MemoryPercent,
			MemoryBytes:  p.MemoryBytes,
			CreateTime:   p.CreateTime,
			Status:       p.Status,
			Command:      p.Command,
		})
	}

	log.Printf("Sending %d processes to server", len(req.Processes))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
			ScriptId:    result.ScriptID,
			ScriptName: result.ScriptName,
			Timestamp:  result.Timestamp,
			Success:    result.Success,
			Output:     result.Output,
			Error:      result.Error,
			ExitCode:   int32(result.ExitCode),
			DurationMs: result.Duration,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		return nil
	}

	req := &pb.ServiceStatusRequest{
		HostId:    r.hostID,
		Timestamp: time.Now().Unix(),
		Services:  make([]*pb.ServiceInfo, 0),
	}

	for _, s := range data.Services {
		req.Services = append(req.Services, &pb.ServiceInfo{
			Name:         s.Name,
			Status:       s.Status,
			Enabled:      s.Enabled,
			Description:  s.Description,
			UptimeSeconds: s.Uptime,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.client.ReportServiceStatus(ctx, req)
	return err
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
	config := LoadAgentConfig()

	if config.ManualIP != "" {
		log.Printf("Using manual IP from config: %s", config.ManualIP)
		return config.ManualIP
	}

	ip := getLocalIP()
	log.Printf("Auto-detected IP: %s", ip)
	return ip
}

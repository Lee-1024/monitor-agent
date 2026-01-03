// ============================================
// 文件: reporter.go (新增)
// ============================================
package main

import (
	"context"
	"log"
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
	ip := getLocalIP()

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

// Close 关闭连接
func (r *Reporter) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
}

// getLocalIP 获取本机IP
func getLocalIP() string {
	// 简化实现，实际应该获取真实IP
	return "127.0.0.1"
}

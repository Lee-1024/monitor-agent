// ============================================
// 文件: collector_service.go
// 服务状态检测器（增强版：支持端口检查）
// ============================================
package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ServiceCollector 服务状态检测器
type ServiceCollector struct {
	services     []string            // 要检测的服务列表（兼容旧格式）
	servicePorts []ServicePortConfig // 服务端口配置（新格式，使用 config_agent.go 中的定义）
}

// NewServiceCollector 创建服务检测器
func NewServiceCollector(services []string) *ServiceCollector {
	return &ServiceCollector{
		services: services,
	}
}

// NewServiceCollectorWithPorts 创建支持端口检查的服务检测器
func NewServiceCollectorWithPorts(servicePorts []ServicePortConfig) *ServiceCollector {
	return &ServiceCollector{
		servicePorts: servicePorts,
	}
}

// Name 返回采集器名称
func (s *ServiceCollector) Name() string {
	return "service"
}

// Collect 检测服务状态
func (s *ServiceCollector) Collect() (interface{}, error) {
	var serviceList []ServiceInfo

	// 优先使用新的服务端口配置
	if len(s.servicePorts) > 0 {
		for _, svcConfig := range s.servicePorts {
			info := s.checkServiceWithPort(svcConfig)
			if info != nil {
				serviceList = append(serviceList, *info)
			}
		}
	} else {
		// 兼容旧格式：使用服务名称列表
		servicesToCheck := s.services
		if len(servicesToCheck) == 0 {
			servicesToCheck = s.getDefaultServices()
		}

		for _, serviceName := range servicesToCheck {
			info := s.checkService(serviceName)
			if info != nil {
				serviceList = append(serviceList, *info)
			}
		}
	}

	return &ServiceMetrics{
		Services: serviceList,
		Count:    len(serviceList),
	}, nil
}

// checkService 检查单个服务状态
func (s *ServiceCollector) checkService(serviceName string) *ServiceInfo {
	var cmd *exec.Cmd
	var status string
	var enabled bool
	var description string
	var uptime int64

	if runtime.GOOS == "windows" {
		// Windows系统使用sc命令
		cmd = exec.Command("sc", "query", serviceName)
		output, err := cmd.Output()
		if err != nil {
			return nil // 服务不存在或无法查询
		}

		outputStr := string(output)
		if strings.Contains(outputStr, "RUNNING") {
			status = "running"
		} else if strings.Contains(outputStr, "STOPPED") {
			status = "stopped"
		} else {
			status = "unknown"
		}

		// 检查是否开机自启
		cmd = exec.Command("sc", "qc", serviceName)
		output, _ = cmd.Output()
		if strings.Contains(string(output), "AUTO_START") {
			enabled = true
		}
	} else {
		// Linux/Unix系统使用systemctl
		cmd = exec.Command("systemctl", "is-active", serviceName)
		output, err := cmd.Output()
		if err != nil {
			status = "stopped"
		} else {
			if strings.TrimSpace(string(output)) == "active" {
				status = "running"
			} else {
				status = "stopped"
			}
		}

		// 检查是否开机自启
		cmd = exec.Command("systemctl", "is-enabled", serviceName)
		output, _ = cmd.Output()
		if strings.Contains(string(output), "enabled") {
			enabled = true
		}

		// 获取服务描述
		cmd = exec.Command("systemctl", "show", serviceName, "--property=Description")
		output, _ = cmd.Output()
		description = strings.TrimSpace(strings.TrimPrefix(string(output), "Description="))

		// 获取运行时长
		cmd = exec.Command("systemctl", "show", serviceName, "--property=ActiveEnterTimestamp")
		output, _ = cmd.Output()
		if timestampStr := strings.TrimSpace(strings.TrimPrefix(string(output), "ActiveEnterTimestamp=")); timestampStr != "" {
			// 解析时间戳并计算运行时长
			// 这里简化处理，实际应该解析systemd的时间戳格式
			uptime = 0 // 需要更复杂的解析逻辑
		}
	}

	return &ServiceInfo{
		Name:        serviceName,
		Status:      status,
		Enabled:     enabled,
		Description: description,
		Uptime:      uptime,
	}
}

// getDefaultServices 获取默认要检测的服务列表
func (s *ServiceCollector) getDefaultServices() []string {
	if runtime.GOOS == "windows" {
		return []string{
			"Spooler",      // 打印服务
			"Themes",       // 主题服务
			"WSearch",      // Windows搜索
		}
	} else {
		return []string{
			"sshd",         // SSH服务
			"docker",       // Docker服务
			"nginx",        // Nginx
			"mysql",        // MySQL
			"postgresql",   // PostgreSQL
		}
	}
}

// checkServiceWithPort 检查服务状态（支持端口检查，类似 telnet）
func (s *ServiceCollector) checkServiceWithPort(config ServicePortConfig) *ServiceInfo {
	// 首先检查系统服务状态
	serviceInfo := s.checkService(config.Name)
	
	// 如果没有配置端口，只返回系统服务状态
	if config.Port <= 0 {
		return serviceInfo
	}
	
	// 如果有配置端口，进行端口检查（类似 telnet）
	host := config.Host
	if host == "" {
		host = "localhost"
	}
	
	// 执行端口探测
	portAccessible := s.probePort(host, config.Port)
	
	// 创建服务信息
	info := &ServiceInfo{
		Name:        config.Name,
		Status:      serviceInfo.Status,
		Enabled:     serviceInfo.Enabled,
		Description: config.Description,
		Uptime:      serviceInfo.Uptime,
	}
	
	// 如果配置了描述，使用配置的描述
	if config.Description != "" {
		info.Description = config.Description
	}
	
	// 设置端口信息
	info.Port = config.Port
	info.PortAccessible = portAccessible
	
	// 如果系统服务状态是 running，但端口不可访问，则标记为 failed
	if serviceInfo.Status == "running" && !portAccessible {
		info.Status = "failed"
	} else if serviceInfo.Status == "stopped" && portAccessible {
		// 如果系统服务状态是 stopped，但端口可访问，可能是服务名称不对，但端口确实在运行
		info.Status = "running"
	} else if serviceInfo.Status == "" && portAccessible {
		// 如果系统服务不存在，但端口可访问，标记为 running
		info.Status = "running"
	}
	
	return info
}

// probePort 端口探测（类似 telnet）
func (s *ServiceCollector) probePort(host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	timeout := 3 * time.Second
	
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	
	if conn != nil {
		conn.Close()
	}
	return true
}

// ServiceInfo 服务信息（扩展版：支持端口检查）
type ServiceInfo struct {
	Name          string `json:"name"`
	Status        string `json:"status"`          // running, stopped, failed, unknown
	Enabled       bool   `json:"enabled"`         // 是否开机自启
	Description   string `json:"description"`     // 服务描述
	Uptime        int64  `json:"uptime_seconds"`  // 运行时长（秒）
	Port          int    `json:"port,omitempty"`  // 服务端口
	PortAccessible bool  `json:"port_accessible,omitempty"` // 端口是否可访问
}

// ServiceMetrics 服务指标
type ServiceMetrics struct {
	Services []ServiceInfo `json:"services"`
	Count    int           `json:"count"`
}


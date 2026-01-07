// ============================================
// 文件: collector_service.go
// 服务状态检测器
// ============================================
package main

import (
	"os/exec"
	"runtime"
	"strings"
)

// ServiceCollector 服务状态检测器
type ServiceCollector struct {
	services []string // 要检测的服务列表
}

// NewServiceCollector 创建服务检测器
func NewServiceCollector(services []string) *ServiceCollector {
	return &ServiceCollector{
		services: services,
	}
}

// Name 返回采集器名称
func (s *ServiceCollector) Name() string {
	return "service"
}

// Collect 检测服务状态
func (s *ServiceCollector) Collect() (interface{}, error) {
	var serviceList []ServiceInfo

	// 如果没有指定服务列表，尝试检测常见服务
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

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name        string `json:"name"`
	Status      string `json:"status"`      // running, stopped, failed, unknown
	Enabled     bool   `json:"enabled"`     // 是否开机自启
	Description string `json:"description"` // 服务描述
	Uptime      int64  `json:"uptime_seconds"` // 运行时长（秒）
}

// ServiceMetrics 服务指标
type ServiceMetrics struct {
	Services []ServiceInfo `json:"services"`
	Count    int           `json:"count"`
}


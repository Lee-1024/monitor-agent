package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type ServiceCollector struct {
	services     []string
	servicePorts []ServicePortConfig
}

func NewServiceCollector(services []string) *ServiceCollector {
	return &ServiceCollector{services: services}
}

func NewServiceCollectorWithPorts(servicePorts []ServicePortConfig) *ServiceCollector {
	return &ServiceCollector{servicePorts: servicePorts}
}

func (s *ServiceCollector) Name() string {
	return "service"
}

func (s *ServiceCollector) Collect() (interface{}, error) {
	serviceList := make([]ServiceInfo, 0)

	if len(s.servicePorts) > 0 {
		for _, svcConfig := range s.servicePorts {
			info := s.checkServiceWithPort(svcConfig)
			if info != nil {
				serviceList = append(serviceList, *info)
			}
		}
	} else {
		for _, serviceName := range s.services {
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

func (s *ServiceCollector) checkService(serviceName string) *ServiceInfo {
	var cmd *exec.Cmd
	var status string
	var enabled bool
	var description string

	if runtime.GOOS == "windows" {
		cmd = exec.Command("sc", "query", serviceName)
		output, err := cmd.Output()
		if err != nil {
			return nil
		}

		outputStr := string(output)
		if strings.Contains(outputStr, "RUNNING") {
			status = "running"
		} else if strings.Contains(outputStr, "STOPPED") {
			status = "stopped"
		} else {
			status = "unknown"
		}

		cmd = exec.Command("sc", "qc", serviceName)
		output, _ = cmd.Output()
		enabled = strings.Contains(string(output), "AUTO_START")
	} else {
		cmd = exec.Command("systemctl", "is-active", serviceName)
		output, err := cmd.Output()
		if err != nil || strings.TrimSpace(string(output)) != "active" {
			status = "stopped"
		} else {
			status = "running"
		}

		cmd = exec.Command("systemctl", "is-enabled", serviceName)
		output, _ = cmd.Output()
		enabled = strings.Contains(string(output), "enabled")

		cmd = exec.Command("systemctl", "show", serviceName, "--property=Description")
		output, _ = cmd.Output()
		description = strings.TrimSpace(strings.TrimPrefix(string(output), "Description="))
	}

	return &ServiceInfo{
		Name:        serviceName,
		Status:      status,
		Enabled:     enabled,
		Description: description,
		Uptime:      0,
	}
}

func (s *ServiceCollector) checkServiceWithPort(config ServicePortConfig) *ServiceInfo {
	if config.Port <= 0 {
		return &ServiceInfo{
			Name:        config.Name,
			Status:      "unknown",
			Description: config.Description,
		}
	}

	host := config.Host
	if host == "" {
		host = "localhost"
	}

	return serviceInfoFromPortCheck(config, s.probePort(host, config.Port))
}

func serviceInfoFromPortCheck(config ServicePortConfig, portAccessible bool) *ServiceInfo {
	status := "failed"
	if portAccessible {
		status = "running"
	}

	return &ServiceInfo{
		Name:           config.Name,
		Status:         status,
		Enabled:        true,
		Description:    config.Description,
		Uptime:         0,
		Port:           config.Port,
		PortAccessible: portAccessible,
	}
}

func (s *ServiceCollector) probePort(host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

type ServiceInfo struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
	Uptime         int64  `json:"uptime_seconds"`
	Port           int    `json:"port,omitempty"`
	PortAccessible bool   `json:"port_accessible,omitempty"`
}

type ServiceMetrics struct {
	Services []ServiceInfo `json:"services"`
	Count    int           `json:"count"`
}

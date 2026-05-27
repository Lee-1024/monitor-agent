package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type GPUCollector struct {
	config GPUConfig
}

func NewGPUCollector(config GPUConfig) *GPUCollector {
	if config.Provider == "" {
		config.Provider = "auto"
	}
	if config.Timeout <= 0 {
		config.Timeout = 5
	}
	return &GPUCollector{config: config}
}

func (c *GPUCollector) Name() string {
	return "gpu"
}

func (c *GPUCollector) Collect() (interface{}, error) {
	if !c.config.Enabled {
		return &GPUMetrics{Devices: []GPUDeviceMetrics{}}, nil
	}

	providers := c.config.Providers
	if len(providers) == 0 {
		if c.config.Provider == "auto" || c.config.Provider == "" {
			providers = []string{"nvidia-smi", "rocm-smi", "intel_gpu_top"}
		} else {
			providers = []string{c.config.Provider}
		}
	}

	for _, provider := range providers {
		metrics, err := c.collectProvider(provider)
		if err == nil && len(metrics.Devices) > 0 {
			return metrics, nil
		}
	}

	return &GPUMetrics{Devices: []GPUDeviceMetrics{}}, nil
}

func (c *GPUCollector) collectProvider(provider string) (*GPUMetrics, error) {
	switch provider {
	case "nvidia-smi":
		output, err := c.runCommand("nvidia-smi", []string{
			"--query-gpu=index,name,uuid,driver_version,utilization.gpu,memory.total,memory.used,utilization.memory,temperature.gpu,power.draw,fan.speed",
			"--format=csv,noheader,nounits",
		})
		if err != nil {
			return nil, err
		}
		return parseNVIDIASMIOutput(output)
	case "custom_command":
		if c.config.Command == "" {
			return nil, errors.New("custom_command requires command")
		}
		output, err := c.runCommand(c.config.Command, c.config.Args)
		if err != nil {
			return nil, err
		}
		return parseNormalizedGPUJSON(output)
	case "rocm-smi", "intel_gpu_top":
		return nil, errors.New("provider parser not configured")
	default:
		return nil, errors.New("unknown gpu provider")
	}
}

func (c *GPUCollector) runCommand(command string, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.config.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func parseNVIDIASMIOutput(output string) (*GPUMetrics, error) {
	reader := csv.NewReader(strings.NewReader(strings.TrimSpace(output)))
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	metrics := &GPUMetrics{Devices: make([]GPUDeviceMetrics, 0, len(rows))}
	for _, row := range rows {
		if len(row) < 11 {
			continue
		}
		totalMiB := parseUint(row[5])
		usedMiB := parseUint(row[6])
		device := GPUDeviceMetrics{
			Index:              int(parseUint(row[0])),
			Name:               strings.TrimSpace(row[1]),
			Vendor:             "nvidia",
			Model:              strings.TrimSpace(row[1]),
			UUID:               strings.TrimSpace(row[2]),
			DriverVersion:      strings.TrimSpace(row[3]),
			UtilizationPercent: parseFloat(row[4]),
			MemoryTotal:        totalMiB * 1024 * 1024,
			MemoryUsed:         usedMiB * 1024 * 1024,
			MemoryUsedPercent:  parseFloat(row[7]),
			Temperature:        parseFloat(row[8]),
			PowerWatts:         parseFloat(row[9]),
			FanSpeedPercent:    parseFloat(row[10]),
		}
		if device.MemoryUsedPercent == 0 && device.MemoryTotal > 0 {
			device.MemoryUsedPercent = float64(device.MemoryUsed) / float64(device.MemoryTotal) * 100
		}
		metrics.Devices = append(metrics.Devices, device)
	}

	return metrics, nil
}

func parseNormalizedGPUJSON(output string) (*GPUMetrics, error) {
	var metrics GPUMetrics
	if err := json.Unmarshal([]byte(output), &metrics); err != nil {
		var devices []GPUDeviceMetrics
		if err := json.Unmarshal([]byte(output), &devices); err != nil {
			return nil, err
		}
		metrics.Devices = devices
	}
	for i := range metrics.Devices {
		if metrics.Devices[i].MemoryUsedPercent == 0 && metrics.Devices[i].MemoryTotal > 0 {
			metrics.Devices[i].MemoryUsedPercent = float64(metrics.Devices[i].MemoryUsed) / float64(metrics.Devices[i].MemoryTotal) * 100
		}
	}
	return &metrics, nil
}

func parseFloat(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") || strings.EqualFold(value, "[Not Supported]") {
		return 0
	}
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}

func parseUint(value string) uint64 {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") || strings.EqualFold(value, "[Not Supported]") {
		return 0
	}
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}

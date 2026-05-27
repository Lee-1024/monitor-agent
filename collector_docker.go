package main

import (
	"encoding/json"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type DockerCollector struct{}

func NewDockerCollector() *DockerCollector {
	return &DockerCollector{}
}

func (c *DockerCollector) Name() string {
	return "docker"
}

type DockerMetrics struct {
	Containers []DockerContainerInfo `json:"containers"`
	Total      int                   `json:"total"`
}

type DockerContainerInfo struct {
	ContainerID   string    `json:"container_id"`
	Name          string    `json:"name"`
	Image         string    `json:"image"`
	State         string    `json:"state"`
	Status        string    `json:"status"`
	CreatedUnix   int64     `json:"created_unix"`
	StartedAt     time.Time `json:"started_at"`
	RestartCount  int       `json:"restart_count"`
	Ports         string    `json:"ports"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryUsage   uint64    `json:"memory_usage"`
	MemoryLimit   uint64    `json:"memory_limit"`
	MemoryPercent float64   `json:"memory_percent"`
	NetworkRx     uint64    `json:"network_rx"`
	NetworkTx     uint64    `json:"network_tx"`
	BlockRead     uint64    `json:"block_read"`
	BlockWrite    uint64    `json:"block_write"`
}

type dockerCPUStats struct {
	CPUUsageTotal         uint64
	PreCPUUsageTotal      uint64
	SystemCPUUsage        uint64
	PreSystemCPUUsage     uint64
	OnlineCPUs            uint32
	PercpuUsageEntryCount int
}

func (c *DockerCollector) Collect() (interface{}, error) {
	containers, err := c.collectWithDockerCLI()
	if err != nil {
		log.Printf("Docker collection unavailable: %v", err)
		return &DockerMetrics{Containers: []DockerContainerInfo{}, Total: 0}, nil
	}
	return &DockerMetrics{Containers: containers, Total: len(containers)}, nil
}

func (c *DockerCollector) collectWithDockerCLI() ([]DockerContainerInfo, error) {
	listOutput, err := exec.Command("docker", "ps", "-a", "--format", "{{json .}}").Output()
	if err != nil {
		return nil, err
	}

	statsOutput, _ := exec.Command("docker", "stats", "--no-stream", "--format", "{{json .}}").Output()
	statsByID := parseDockerStatsOutput(string(statsOutput))

	lines := strings.Split(strings.TrimSpace(string(listOutput)), "\n")
	containers := make([]DockerContainerInfo, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw dockerPSLine
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		info := DockerContainerInfo{
			ContainerID: raw.ID,
			Name:        raw.Names,
			Image:       raw.Image,
			State:       normalizeDockerState(raw.State),
			Status:      raw.Status,
			Ports:       raw.Ports,
		}
		if stat, ok := statsByID[shortContainerID(raw.ID)]; ok {
			info.CPUPercent = stat.CPUPercent
			info.MemoryUsage = stat.MemoryUsage
			info.MemoryLimit = stat.MemoryLimit
			info.MemoryPercent = stat.MemoryPercent
			info.NetworkRx = stat.NetworkRx
			info.NetworkTx = stat.NetworkTx
			info.BlockRead = stat.BlockRead
			info.BlockWrite = stat.BlockWrite
		}
		containers = append(containers, info)
	}
	return containers, nil
}

type dockerPSLine struct {
	ID     string `json:"ID"`
	Image  string `json:"Image"`
	Names  string `json:"Names"`
	State  string `json:"State"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

type dockerStatsLine struct {
	ID       string `json:"ID"`
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
	MemPerc  string `json:"MemPerc"`
	NetIO    string `json:"NetIO"`
	BlockIO  string `json:"BlockIO"`
}

func parseDockerStatsOutput(output string) map[string]DockerContainerInfo {
	result := make(map[string]DockerContainerInfo)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw dockerStatsLine
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		usage, limit := parseDockerUsagePair(raw.MemUsage)
		rx, tx := parseDockerUsagePair(raw.NetIO)
		read, write := parseDockerUsagePair(raw.BlockIO)
		result[shortContainerID(raw.ID)] = DockerContainerInfo{
			CPUPercent:    parsePercent(raw.CPUPerc),
			MemoryUsage:   usage,
			MemoryLimit:   limit,
			MemoryPercent: parsePercent(raw.MemPerc),
			NetworkRx:     rx,
			NetworkTx:     tx,
			BlockRead:     read,
			BlockWrite:    write,
		}
	}
	return result
}

func calculateDockerCPUPercent(stats dockerCPUStats) float64 {
	cpuDelta := float64(stats.CPUUsageTotal - stats.PreCPUUsageTotal)
	systemDelta := float64(stats.SystemCPUUsage - stats.PreSystemCPUUsage)
	onlineCPUs := float64(stats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(stats.PercpuUsageEntryCount)
	}
	if systemDelta <= 0 || cpuDelta <= 0 || onlineCPUs <= 0 {
		return 0
	}
	return cpuDelta / systemDelta * onlineCPUs * 100
}

func calculateDockerMemoryPercent(usage, limit uint64) float64 {
	if limit == 0 {
		return 0
	}
	return float64(usage) / float64(limit) * 100
}

func normalizeDockerState(state string) string {
	state = strings.ToLower(strings.TrimSpace(state))
	if state == "" {
		return "unknown"
	}
	return state
}

func shortContainerID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func parsePercent(value string) float64 {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}

func parseDockerUsagePair(value string) (uint64, uint64) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	return parseDockerSize(parts[0]), parseDockerSize(parts[1])
}

func parseDockerSize(value string) uint64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	fields := strings.Fields(value)
	if len(fields) == 2 {
		value = fields[0] + fields[1]
	}
	units := []struct {
		suffix string
		mult   float64
	}{
		{"GiB", 1024 * 1024 * 1024},
		{"MiB", 1024 * 1024},
		{"KiB", 1024},
		{"GB", 1000 * 1000 * 1000},
		{"MB", 1000 * 1000},
		{"KB", 1000},
		{"B", 1},
	}
	for _, unit := range units {
		if strings.HasSuffix(value, unit.suffix) {
			number := strings.TrimSpace(strings.TrimSuffix(value, unit.suffix))
			parsed, _ := strconv.ParseFloat(number, 64)
			return uint64(parsed * unit.mult)
		}
	}
	parsed, _ := strconv.ParseFloat(value, 64)
	return uint64(parsed)
}

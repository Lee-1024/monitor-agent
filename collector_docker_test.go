package main

import "testing"

func TestCalculateDockerCPUPercent(t *testing.T) {
	stats := dockerCPUStats{
		CPUUsageTotal:         300,
		PreCPUUsageTotal:      100,
		SystemCPUUsage:        2000,
		PreSystemCPUUsage:     1000,
		OnlineCPUs:            2,
		PercpuUsageEntryCount: 2,
	}

	if got := calculateDockerCPUPercent(stats); got != 40 {
		t.Fatalf("calculateDockerCPUPercent() = %.2f, want 40.00", got)
	}
}

func TestCalculateDockerMemoryPercent(t *testing.T) {
	if got := calculateDockerMemoryPercent(512, 2048); got != 25 {
		t.Fatalf("calculateDockerMemoryPercent() = %.2f, want 25.00", got)
	}
}

func TestDockerCollectorName(t *testing.T) {
	collector := NewDockerCollector()
	if got := collector.Name(); got != "docker" {
		t.Fatalf("Name() = %q, want docker", got)
	}
}

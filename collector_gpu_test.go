package main

import "testing"

func TestParseNVIDIASMIOutput(t *testing.T) {
	output := "0, NVIDIA A100, GPU-123, 535.54.03, 82, 40536, 20268, 50, 71, 250.5, 35\n"

	metrics, err := parseNVIDIASMIOutput(output)
	if err != nil {
		t.Fatalf("parseNVIDIASMIOutput returned error: %v", err)
	}
	if len(metrics.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(metrics.Devices))
	}

	device := metrics.Devices[0]
	if device.Vendor != "nvidia" {
		t.Fatalf("expected vendor nvidia, got %q", device.Vendor)
	}
	if device.Name != "NVIDIA A100" {
		t.Fatalf("expected name NVIDIA A100, got %q", device.Name)
	}
	if device.UUID != "GPU-123" {
		t.Fatalf("expected UUID GPU-123, got %q", device.UUID)
	}
	if device.UtilizationPercent != 82 {
		t.Fatalf("expected utilization 82, got %.1f", device.UtilizationPercent)
	}
	if device.MemoryTotal != 40536*1024*1024 {
		t.Fatalf("expected memory total in bytes, got %d", device.MemoryTotal)
	}
	if device.MemoryUsedPercent != 50 {
		t.Fatalf("expected memory used percent 50, got %.1f", device.MemoryUsedPercent)
	}
}

func TestParseNormalizedGPUJSON(t *testing.T) {
	output := `{"devices":[{"index":0,"name":"Custom GPU","vendor":"custom","uuid":"abc","utilization_percent":45.5,"memory_total":1024,"memory_used":512,"temperature":60}]}`

	metrics, err := parseNormalizedGPUJSON(output)
	if err != nil {
		t.Fatalf("parseNormalizedGPUJSON returned error: %v", err)
	}
	if len(metrics.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(metrics.Devices))
	}
	if metrics.Devices[0].Name != "Custom GPU" {
		t.Fatalf("expected Custom GPU, got %q", metrics.Devices[0].Name)
	}
	if metrics.Devices[0].MemoryUsedPercent != 50 {
		t.Fatalf("expected calculated memory used percent 50, got %.1f", metrics.Devices[0].MemoryUsedPercent)
	}
}

func TestGPUCollectorDisabledReturnsEmptyMetrics(t *testing.T) {
	collector := NewGPUCollector(GPUConfig{Enabled: false})

	data, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	metrics := data.(*GPUMetrics)
	if len(metrics.Devices) != 0 {
		t.Fatalf("expected no devices, got %d", len(metrics.Devices))
	}
}

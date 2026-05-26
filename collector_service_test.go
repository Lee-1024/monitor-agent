package main

import "testing"

func TestServiceCollectorReturnsNoServicesWhenYamlHasNoServices(t *testing.T) {
	collector := NewServiceCollector(nil)

	data, err := collector.Collect()
	if err != nil {
		t.Fatalf("collect services: %v", err)
	}

	metrics := data.(*ServiceMetrics)
	if metrics.Count != 0 {
		t.Fatalf("expected no default services, got %d", metrics.Count)
	}
}

func TestServicePortStatusUsesPortAccessibilityOnly(t *testing.T) {
	info := serviceInfoFromPortCheck(ServicePortConfig{
		Name:        "external-api",
		Host:        "10.0.0.10",
		Port:        8080,
		Description: "External API",
	}, true)

	if info.Status != "running" {
		t.Fatalf("expected running for accessible port, got %q", info.Status)
	}
	if !info.PortAccessible {
		t.Fatal("expected port to be accessible")
	}

	info = serviceInfoFromPortCheck(ServicePortConfig{Name: "external-api", Port: 8080}, false)
	if info.Status != "failed" {
		t.Fatalf("expected failed for inaccessible port, got %q", info.Status)
	}
}

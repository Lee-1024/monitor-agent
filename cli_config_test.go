package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAgentConfigFromPathLoadsSpecifiedConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom-agent.yaml")
	content := []byte(`server_addr: "10.0.0.1:50051"
host_id: "custom-host"
hostname: "custom-name"
collect_interval: 15
manual_ip: "10.0.0.2"
debug: true
`)
	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config := LoadAgentConfigFromPath(configPath)

	if config.ServerAddr != "10.0.0.1:50051" {
		t.Fatalf("expected server_addr from custom config, got %q", config.ServerAddr)
	}
	if config.HostID != "custom-host" {
		t.Fatalf("expected host_id from custom config, got %q", config.HostID)
	}
	if config.CollectInterval != 15 {
		t.Fatalf("expected collect_interval from custom config, got %d", config.CollectInterval)
	}
	if !config.Debug {
		t.Fatal("expected debug from custom config")
	}
}

func TestApplyFlagOverridesUsesConfigWhenFlagsMissing(t *testing.T) {
	oldCommandLine := flag.CommandLine
	t.Cleanup(func() { flag.CommandLine = oldCommandLine })
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	serverAddr := ""
	hostID := "host-001"
	interval := 10
	debug := false
	config := &AgentConfig{
		ServerAddr:      "10.0.0.1:50051",
		HostID:          "config-host",
		CollectInterval: 15,
		Debug:           true,
	}

	applyFlagOverrides(config, &serverAddr, &hostID, &interval, &debug)

	if serverAddr != config.ServerAddr || hostID != config.HostID || interval != config.CollectInterval || debug != config.Debug {
		t.Fatalf("expected values from config, got server=%q host=%q interval=%d debug=%v", serverAddr, hostID, interval, debug)
	}
}

func TestApplyFlagOverridesKeepsExplicitFlags(t *testing.T) {
	oldCommandLine := flag.CommandLine
	t.Cleanup(func() { flag.CommandLine = oldCommandLine })
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	flag.CommandLine.String("server", "", "")
	flag.CommandLine.String("host-id", "", "")
	flag.CommandLine.Int("interval", 0, "")
	flag.CommandLine.Bool("debug", false, "")
	if err := flag.CommandLine.Parse([]string{"-server", "flag-server:50051", "-host-id", "flag-host", "-interval", "20", "-debug"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	serverAddr := "flag-server:50051"
	hostID := "flag-host"
	interval := 20
	debug := true
	config := &AgentConfig{
		ServerAddr:      "config-server:50051",
		HostID:          "config-host",
		CollectInterval: 15,
		Debug:           false,
	}

	applyFlagOverrides(config, &serverAddr, &hostID, &interval, &debug)

	if serverAddr != "flag-server:50051" || hostID != "flag-host" || interval != 20 || !debug {
		t.Fatalf("expected explicit flags to win, got server=%q host=%q interval=%d debug=%v", serverAddr, hostID, interval, debug)
	}
}

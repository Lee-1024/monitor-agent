// ============================================
// 文件: config_agent.go (新增)
// ============================================
package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	ServerAddr      string         `yaml:"server_addr"`
	HostID          string         `yaml:"host_id"`
	CollectInterval int            `yaml:"collect_interval"`
	ManualIP        string         `yaml:"manual_ip"`
	Debug           bool           `yaml:"debug"`
	LogPaths        []string       `yaml:"log_paths"`        // 日志文件路径列表
	Scripts         []ScriptConfig `yaml:"scripts"`           // 脚本配置列表
	Services        []string       `yaml:"services"`          // 要检测的服务列表
}

func LoadAgentConfig() *AgentConfig {
	config := &AgentConfig{
		ServerAddr:      "localhost:50051",
		HostID:          "host-001",
		CollectInterval: 10,
		ManualIP:        "",
		Debug:           false,
	}

	configFile := "agent-config.yaml"
	if data, err := os.ReadFile(configFile); err == nil {
		if err := yaml.Unmarshal(data, config); err != nil {
			log.Printf("Failed to parse config: %v", err)
		} else {
			log.Printf("Loaded config from %s", configFile)
		}
	}

	return config
}

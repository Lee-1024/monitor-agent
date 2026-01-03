// ============================================
// 文件: config_agent.go (新增)
// ============================================
package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// AgentConfig Agent配置
type AgentConfig struct {
	ServerAddr      string `yaml:"server_addr"`
	HostID          string `yaml:"host_id"`
	CollectInterval int    `yaml:"collect_interval"` // 秒
	Debug           bool   `yaml:"debug"`
}

// LoadAgentConfig 加载配置
func LoadAgentConfig() *AgentConfig {
	config := &AgentConfig{
		ServerAddr:      "localhost:50051",
		HostID:          "host-001",
		CollectInterval: 10,
		Debug:           false,
	}

	// 尝试从配置文件加载
	if data, err := os.ReadFile("agent-config.yaml"); err == nil {
		if err := yaml.Unmarshal(data, config); err != nil {
			log.Printf("Failed to parse config: %v", err)
		}
	}

	return config
}

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
	ServerAddr      string              `yaml:"server_addr"`
	HostID          string              `yaml:"host_id"`
	Hostname        string              `yaml:"hostname"`
	CollectInterval int                 `yaml:"collect_interval"`
	ManualIP        string              `yaml:"manual_ip"`
	Debug           bool                `yaml:"debug"`
	LogPaths        []string            `yaml:"log_paths"`     // 日志文件路径列表
	Scripts         []ScriptConfig      `yaml:"scripts"`       // 脚本配置列表
	Services        []string            `yaml:"services"`      // 要检测的服务列表（兼容旧格式）
	ServicePorts    []ServicePortConfig `yaml:"service_ports"` // 服务端口配置（新格式，支持端口检查）
	GRPC            GRPCConfig          `yaml:"grpc"`          // gRPC连接与请求超时配置
	Fallback        FallbackConfig      `yaml:"fallback"`      // gRPC失败后的HTTP兜底和本地缓存配置
	GPU             GPUConfig           `yaml:"gpu"`
}

type GRPCConfig struct {
	ConnectTimeout   int `yaml:"connect_timeout"`   // 建立gRPC连接超时时间（秒）
	RegisterTimeout  int `yaml:"register_timeout"`  // 注册Agent超时时间（秒）
	ReportTimeout    int `yaml:"report_timeout"`    // 指标上报超时时间（秒）
	HeartbeatTimeout int `yaml:"heartbeat_timeout"` // 心跳超时时间（秒）
	RequestTimeout   int `yaml:"request_timeout"`   // 其他请求超时时间（秒）
}

type GPUConfig struct {
	Enabled       bool              `yaml:"enabled"`
	Provider      string            `yaml:"provider"`
	Providers     []string          `yaml:"providers"`
	Command       string            `yaml:"command"`
	Args          []string          `yaml:"args"`
	Timeout       int               `yaml:"timeout"`
	FieldMappings map[string]string `yaml:"field_mappings"`
}

type FallbackConfig struct {
	HTTPEnabled   bool   `yaml:"http_enabled"`
	HTTPBaseURL   string `yaml:"http_base_url"`
	CacheEnabled  bool   `yaml:"cache_enabled"`
	CacheDir      string `yaml:"cache_dir"`
	MaxCacheFiles int    `yaml:"max_cache_files"`
}

// ServicePortConfig 服务端口配置（支持端口检查，类似 telnet）
type ServicePortConfig struct {
	Name        string `yaml:"name"`        // 服务名称
	Port        int    `yaml:"port"`        // 服务端口（用于端口检查，类似 telnet）
	Host        string `yaml:"host"`        // 主机地址（可选，默认为 localhost）
	Description string `yaml:"description"` // 服务描述（可选）
}

func LoadAgentConfig() *AgentConfig {
	return LoadAgentConfigFromPath(ConfigPathFromEnv())
}

func ConfigPathFromEnv() string {
	configFile := os.Getenv("CONFIG_PATH")
	if configFile == "" {
		configFile = "agent-config.yaml"
	}
	return configFile
}

func LoadAgentConfigFromPath(configFile string) *AgentConfig {
	config := &AgentConfig{
		ServerAddr:      "localhost:50051",
		HostID:          "host-001",
		CollectInterval: 10,
		ManualIP:        "",
		Debug:           false,
		GRPC: GRPCConfig{
			ConnectTimeout:   5,
			RegisterTimeout:  5,
			ReportTimeout:    5,
			HeartbeatTimeout: 3,
			RequestTimeout:   10,
		},
		Fallback: FallbackConfig{
			HTTPEnabled:   false,
			CacheEnabled:  true,
			CacheDir:      "./agent-cache",
			MaxCacheFiles: 1000,
		},
		GPU: GPUConfig{
			Enabled:  true,
			Provider: "auto",
			Timeout:  5,
		},
	}

	if configFile == "" {
		configFile = "agent-config.yaml"
	}
	if data, err := os.ReadFile(configFile); err == nil {
		if err := yaml.Unmarshal(data, config); err != nil {
			log.Printf("Failed to parse config: %v", err)
		} else {
			log.Printf("Loaded config from %s", configFile)
		}
	}

	return config
}

func (c *AgentConfig) EffectiveHostname(systemHostname string) string {
	if c != nil && c.Hostname != "" {
		return c.Hostname
	}
	return systemHostname
}

func (c *AgentConfig) LogCollectionEnabled() bool {
	return c != nil && len(c.LogPaths) > 0
}

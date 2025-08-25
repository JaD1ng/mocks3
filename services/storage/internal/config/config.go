package config

import (
	"fmt"
	"mocks3/shared/utils"
)

// Config 存储服务配置
type Config struct {
	Server     ServerConfig     `yaml:"server" json:"server"`
	Storage    StorageConfig    `yaml:"storage" json:"storage"`
	Metadata   MetadataConfig   `yaml:"metadata" json:"metadata"`
	ThirdParty ThirdPartyConfig `yaml:"third_party" json:"third_party"`
	LogLevel   string           `yaml:"log_level" json:"log_level"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	Environment string `yaml:"environment" json:"environment"`
	Version     string `yaml:"version" json:"version"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	DataDir string       `yaml:"data_dir" json:"data_dir"`
	Nodes   []NodeConfig `yaml:"nodes" json:"nodes"`
}

// NodeConfig 存储节点配置
type NodeConfig struct {
	ID   string `yaml:"id" json:"id"`
	Path string `yaml:"path" json:"path"`
}

// MetadataConfig 元数据服务配置
type MetadataConfig struct {
	ServiceURL string `yaml:"service_url" json:"service_url"`
	Timeout    string `yaml:"timeout" json:"timeout"`
}

// ThirdPartyConfig 第三方服务配置
type ThirdPartyConfig struct {
	ServiceURL string `yaml:"service_url" json:"service_url"`
	Timeout    string `yaml:"timeout" json:"timeout"`
	Enabled    bool   `yaml:"enabled" json:"enabled"`
}

// GetAddress 获取服务器地址
func (s *ServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Load 加载配置
func Load() *Config {
	// 默认配置
	config := &Config{
		Server: ServerConfig{
			Host:        "0.0.0.0",
			Port:        8082,
			Environment: "development",
			Version:     "1.0.0",
		},
		Storage: StorageConfig{
			DataDir: "./data/storage",
			Nodes: []NodeConfig{
				{
					ID:   "stg1",
					Path: "./data/storage/stg1",
				},
				{
					ID:   "stg2",
					Path: "./data/storage/stg2",
				},
				{
					ID:   "stg3",
					Path: "./data/storage/stg3",
				},
			},
		},
		Metadata: MetadataConfig{
			ServiceURL: "http://localhost:8081",
			Timeout:    "30s",
		},
		ThirdParty: ThirdPartyConfig{
			ServiceURL: "http://localhost:8084",
			Timeout:    "30s",
			Enabled:    true,
		},
		LogLevel: "info",
	}

	// 尝试从YAML文件加载配置
	if err := utils.LoadServiceConfig("storage", config); err != nil {
		// 如果YAML配置文件不存在，使用默认配置
		fmt.Printf("Warning: Failed to load YAML config, using defaults: %v\n", err)
	}

	return config
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Storage.DataDir == "" {
		return fmt.Errorf("storage data directory is required")
	}

	if len(c.Storage.Nodes) == 0 {
		return fmt.Errorf("at least one storage node is required")
	}

	for _, node := range c.Storage.Nodes {
		if node.ID == "" {
			return fmt.Errorf("storage node ID is required")
		}
		if node.Path == "" {
			return fmt.Errorf("storage node path is required")
		}
	}

	if c.Metadata.ServiceURL == "" {
		return fmt.Errorf("metadata service URL is required")
	}

	return nil
}

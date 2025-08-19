package config

import (
	"fmt"
	"mocks3/shared/utils"
)

// Config 存储服务配置
type Config struct {
	Server     ServerConfig     `json:"server"`
	Storage    StorageConfig    `json:"storage"`
	Metadata   MetadataConfig   `json:"metadata"`
	ThirdParty ThirdPartyConfig `json:"third_party"`
	LogLevel   string           `json:"log_level"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	DataDir string       `json:"data_dir"`
	Nodes   []NodeConfig `json:"nodes"`
}

// NodeConfig 存储节点配置
type NodeConfig struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// MetadataConfig 元数据服务配置
type MetadataConfig struct {
	ServiceURL string `json:"service_url"`
	Timeout    string `json:"timeout"`
}

// ThirdPartyConfig 第三方服务配置
type ThirdPartyConfig struct {
	ServiceURL string `json:"service_url"`
	Timeout    string `json:"timeout"`
	Enabled    bool   `json:"enabled"`
}

// GetAddress 获取服务器地址
func (s *ServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:        utils.GetEnv("SERVER_HOST", "0.0.0.0"),
			Port:        utils.GetEnvInt("SERVICE_PORT", 8082),
			Environment: utils.GetEnv("ENVIRONMENT", "development"),
			Version:     utils.GetEnv("SERVICE_VERSION", "1.0.0"),
		},
		Storage: StorageConfig{
			DataDir: utils.GetEnv("STORAGE_DATA_DIR", "./data/storage"),
			Nodes: []NodeConfig{
				{
					ID:   "stg1",
					Path: utils.GetEnv("STORAGE_STG1_PATH", "./data/storage/stg1"),
				},
				{
					ID:   "stg2",
					Path: utils.GetEnv("STORAGE_STG2_PATH", "./data/storage/stg2"),
				},
				{
					ID:   "stg3",
					Path: utils.GetEnv("STORAGE_STG3_PATH", "./data/storage/stg3"),
				},
			},
		},
		Metadata: MetadataConfig{
			ServiceURL: utils.GetEnv("METADATA_SERVICE_URL", "http://localhost:8081"),
			Timeout:    utils.GetEnv("METADATA_SERVICE_TIMEOUT", "30s"),
		},
		ThirdParty: ThirdPartyConfig{
			ServiceURL: utils.GetEnv("THIRD_PARTY_SERVICE_URL", "http://localhost:8084"),
			Timeout:    utils.GetEnv("THIRD_PARTY_SERVICE_TIMEOUT", "30s"),
			Enabled:    utils.GetEnvBool("THIRD_PARTY_ENABLED", true),
		},
		LogLevel: utils.GetEnv("LOG_LEVEL", "info"),
	}
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

package utils

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 可观测性配置
type Config struct {
	ServiceName    string        `yaml:"service_name"`
	ServiceVersion string        `yaml:"service_version"`
	Environment    string        `yaml:"environment"`
	OTLPEndpoint   string        `yaml:"otlp_endpoint"`
	LogLevel       string        `yaml:"log_level"`
	SamplingRatio  float64       `yaml:"sampling_ratio"`
	ExportInterval time.Duration `yaml:"export_interval"`
}

// ObservabilityConfig 通用可观测性配置
type ObservabilityConfig struct {
	Environment  string `yaml:"environment"`
	OTLPEndpoint string `yaml:"otlp_endpoint"`
	LogLevel     string `yaml:"log_level"`
}

// LoadObservabilityConfig 从YAML配置文件加载可观测性配置
func LoadObservabilityConfig(serviceName, configPath string) (*Config, error) {
	// 默认配置
	config := &Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "development",
		OTLPEndpoint:   "localhost:4318",
		LogLevel:       "info",
		SamplingRatio:  1.0,
		ExportInterval: 30 * time.Second,
	}

	// 如果配置文件存在，则读取YAML配置
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
			}

			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse YAML config: %w", err)
			}
		}
	}

	// 设置服务名称（如果配置文件中没有指定）
	if config.ServiceName == "" {
		config.ServiceName = serviceName
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// Validate 验证可观测性配置
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	if c.OTLPEndpoint == "" {
		return fmt.Errorf("otlp_endpoint is required")
	}
	if c.SamplingRatio < 0 || c.SamplingRatio > 1 {
		return fmt.Errorf("sampling_ratio must be between 0 and 1")
	}
	if c.ExportInterval <= 0 {
		return fmt.Errorf("export_interval must be positive")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level: %s", c.LogLevel)
	}

	return nil
}

// LoadConfigFromYAML 通用的YAML配置加载函数
func LoadConfigFromYAML(configPath string, config any) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return nil
}

// SaveConfigToYAML 将配置保存为YAML文件
func SaveConfigToYAML(configPath string, config any) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// ServerConfig 通用服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

// DatabaseConfig 通用数据库配置
type DatabaseConfig struct {
	Driver          string `yaml:"driver"`
	DSN             string `yaml:"dsn"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	DataDir string       `yaml:"data_dir"`
	Nodes   []NodeConfig `yaml:"nodes"`
}

// NodeConfig 存储节点配置
type NodeConfig struct {
	ID   string `yaml:"id"`
	Path string `yaml:"path"`
}

// QueueConfig 队列配置
type QueueConfig struct {
	Size        int    `yaml:"size"`
	RedisStream string `yaml:"redis_stream"`
	WorkerCount int    `yaml:"worker_count"`
}

// Address 返回服务器地址
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

// FindConfigFile 查找配置文件
func FindConfigFile(serviceName string, customPaths ...string) string {
	configFileName := fmt.Sprintf("%s.yaml", serviceName)

	// 默认搜索路径
	defaultPaths := []string{
		fmt.Sprintf("./config/%s", configFileName),
		fmt.Sprintf("./services/%s/config/%s", serviceName, configFileName),
		fmt.Sprintf("../../../config/%s", configFileName),
		fmt.Sprintf("/etc/mocks3/%s", configFileName),
	}

	// 合并自定义路径和默认路径
	allPaths := append(customPaths, defaultPaths...)

	for _, path := range allPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "" // 未找到配置文件
}

// LoadServiceConfig 通用服务配置加载
func LoadServiceConfig(serviceName string, config any, customPaths ...string) error {
	configPath := FindConfigFile(serviceName, customPaths...)
	if configPath == "" {
		return fmt.Errorf("config file for service '%s' not found", serviceName)
	}

	return LoadConfigFromYAML(configPath, config)
}
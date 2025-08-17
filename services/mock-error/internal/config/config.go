package config

import (
	"fmt"
	"os"
	"strconv"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

// GetAddress 获取服务器地址
func (s *ServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// ConsulConfig Consul配置
type ConsulConfig struct {
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}

// ErrorEngineConfig 错误引擎配置
type ErrorEngineConfig struct {
	MaxRules           int     `json:"max_rules"`
	EnableScheduling   bool    `json:"enable_scheduling"`
	DefaultProbability float64 `json:"default_probability"`
	EnableStatistics   bool    `json:"enable_statistics"`
	StatRetentionHours int     `json:"stat_retention_hours"`
}

// InjectionConfig 注入配置
type InjectionConfig struct {
	MaxDelayMs           int     `json:"max_delay_ms"`
	EnableHTTPErrors     bool    `json:"enable_http_errors"`
	EnableNetworkErrors  bool    `json:"enable_network_errors"`
	EnableDatabaseErrors bool    `json:"enable_database_errors"`
	EnableStorageErrors  bool    `json:"enable_storage_errors"`
	GlobalProbability    float64 `json:"global_probability"`
}

// Config 应用配置
type Config struct {
	Server      ServerConfig      `json:"server"`
	Consul      ConsulConfig      `json:"consul"`
	ErrorEngine ErrorEngineConfig `json:"error_engine"`
	Injection   InjectionConfig   `json:"injection"`
	LogLevel    string            `json:"log_level"`
}

// Load 加载配置
func Load() *Config {
	config := &Config{
		Server: ServerConfig{
			Host:        getEnv("SERVER_HOST", "0.0.0.0"),
			Port:        getEnvAsInt("SERVER_PORT", 8085),
			Environment: getEnv("ENVIRONMENT", "development"),
			Version:     getEnv("VERSION", "1.0.0"),
		},
		Consul: ConsulConfig{
			Address: getEnv("CONSUL_ADDR", "localhost:8500"),
			Enabled: getEnvAsBool("CONSUL_ENABLED", true),
		},
		ErrorEngine: ErrorEngineConfig{
			MaxRules:           getEnvAsInt("ERROR_MAX_RULES", 1000),
			EnableScheduling:   getEnvAsBool("ERROR_ENABLE_SCHEDULING", true),
			DefaultProbability: getEnvAsFloat("ERROR_DEFAULT_PROBABILITY", 0.1),
			EnableStatistics:   getEnvAsBool("ERROR_ENABLE_STATISTICS", true),
			StatRetentionHours: getEnvAsInt("ERROR_STAT_RETENTION_HOURS", 24),
		},
		Injection: InjectionConfig{
			MaxDelayMs:           getEnvAsInt("INJECTION_MAX_DELAY_MS", 10000),
			EnableHTTPErrors:     getEnvAsBool("INJECTION_ENABLE_HTTP_ERRORS", true),
			EnableNetworkErrors:  getEnvAsBool("INJECTION_ENABLE_NETWORK_ERRORS", true),
			EnableDatabaseErrors: getEnvAsBool("INJECTION_ENABLE_DATABASE_ERRORS", true),
			EnableStorageErrors:  getEnvAsBool("INJECTION_ENABLE_STORAGE_ERRORS", true),
			GlobalProbability:    getEnvAsFloat("INJECTION_GLOBAL_PROBABILITY", 1.0),
		},
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	return config
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.ErrorEngine.MaxRules <= 0 {
		return fmt.Errorf("max_rules must be positive")
	}

	if c.ErrorEngine.DefaultProbability < 0 || c.ErrorEngine.DefaultProbability > 1 {
		return fmt.Errorf("default_probability must be between 0 and 1")
	}

	if c.Injection.MaxDelayMs < 0 {
		return fmt.Errorf("max_delay_ms must be non-negative")
	}

	if c.Injection.GlobalProbability < 0 || c.Injection.GlobalProbability > 1 {
		return fmt.Errorf("global_probability must be between 0 and 1")
	}

	return nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt 获取环境变量并转换为int
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool 获取环境变量并转换为bool
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsFloat 获取环境变量并转换为float64
func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

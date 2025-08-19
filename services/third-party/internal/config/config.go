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

// CacheConfig 缓存配置
type CacheConfig struct {
	TTL      int    `json:"ttl_seconds"`
	MaxSize  int64  `json:"max_size_mb"`
	Strategy string `json:"strategy"`
	Enabled  bool   `json:"enabled"`
}

// DataSourceConfig 数据源配置
type DataSourceConfig struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Endpoint    string            `json:"endpoint"`
	AccessKey   string            `json:"access_key"`
	SecretKey   string            `json:"secret_key"`
	Region      string            `json:"region"`
	BucketName  string            `json:"bucket_name"`
	Timeout     int               `json:"timeout_seconds"`
	Enabled     bool              `json:"enabled"`
	Priority    int               `json:"priority"`
	ExtraConfig map[string]string `json:"extra_config"`
}

// Config 应用配置
type Config struct {
	Server      ServerConfig       `json:"server"`
	Cache       CacheConfig        `json:"cache"`
	DataSources []DataSourceConfig `json:"data_sources"`
	LogLevel    string             `json:"log_level"`
}

// Load 加载配置
func Load() *Config {
	config := &Config{
		Server: ServerConfig{
			Host:        getEnv("SERVER_HOST", "0.0.0.0"),
			Port:        getEnvAsInt("SERVER_PORT", 8084),
			Environment: getEnv("ENVIRONMENT", "development"),
			Version:     getEnv("VERSION", "1.0.0"),
		},
		Cache: CacheConfig{
			TTL:      getEnvAsInt("CACHE_TTL", 3600),
			MaxSize:  getEnvAsInt64("CACHE_MAX_SIZE", 1024),
			Strategy: getEnv("CACHE_STRATEGY", "lru"),
			Enabled:  getEnvAsBool("CACHE_ENABLED", true),
		},
		DataSources: []DataSourceConfig{
			{
				Name:       getEnv("DATASOURCE_1_NAME", "backup-s3"),
				Type:       getEnv("DATASOURCE_1_TYPE", "s3"),
				Endpoint:   getEnv("DATASOURCE_1_ENDPOINT", "https://s3.amazonaws.com"),
				AccessKey:  getEnv("DATASOURCE_1_ACCESS_KEY", ""),
				SecretKey:  getEnv("DATASOURCE_1_SECRET_KEY", ""),
				Region:     getEnv("DATASOURCE_1_REGION", "us-east-1"),
				BucketName: getEnv("DATASOURCE_1_BUCKET", "backup-bucket"),
				Timeout:    getEnvAsInt("DATASOURCE_1_TIMEOUT", 30),
				Enabled:    getEnvAsBool("DATASOURCE_1_ENABLED", true),
				Priority:   1,
			},
			{
				Name:     getEnv("DATASOURCE_2_NAME", "backup-http"),
				Type:     getEnv("DATASOURCE_2_TYPE", "http"),
				Endpoint: getEnv("DATASOURCE_2_ENDPOINT", "https://backup.example.com/api"),
				Timeout:  getEnvAsInt("DATASOURCE_2_TIMEOUT", 30),
				Enabled:  getEnvAsBool("DATASOURCE_2_ENABLED", false),
				Priority: 2,
			},
		},
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	return config
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

// getEnvAsInt64 获取环境变量并转换为int64
func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
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

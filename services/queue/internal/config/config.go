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

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// GetAddress 获取Redis地址
func (r *RedisConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// QueueConfig 队列配置
type QueueConfig struct {
	MaxWorkers     int    `json:"max_workers"`
	MaxRetries     int    `json:"max_retries"`
	StreamName     string `json:"stream_name"`
	ConsumerGroup  string `json:"consumer_group"`
	BatchSize      int    `json:"batch_size"`
	ProcessTimeout int    `json:"process_timeout_seconds"`
}

// Config 应用配置
type Config struct {
	Server   ServerConfig `json:"server"`
	Redis    RedisConfig  `json:"redis"`
	Queue    QueueConfig  `json:"queue"`
	LogLevel string       `json:"log_level"`
}

// Load 加载配置
func Load() *Config {
	config := &Config{
		Server: ServerConfig{
			Host:        getEnv("SERVER_HOST", "0.0.0.0"),
			Port:        getEnvAsInt("SERVER_PORT", 8083),
			Environment: getEnv("ENVIRONMENT", "development"),
			Version:     getEnv("VERSION", "1.0.0"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Queue: QueueConfig{
			MaxWorkers:     getEnvAsInt("QUEUE_MAX_WORKERS", 3),
			MaxRetries:     getEnvAsInt("QUEUE_MAX_RETRIES", 3),
			StreamName:     getEnv("QUEUE_STREAM_NAME", "mocks3:tasks"),
			ConsumerGroup:  getEnv("QUEUE_CONSUMER_GROUP", "queue-workers"),
			BatchSize:      getEnvAsInt("QUEUE_BATCH_SIZE", 10),
			ProcessTimeout: getEnvAsInt("QUEUE_PROCESS_TIMEOUT", 30),
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

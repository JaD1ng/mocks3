package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port              int    `json:"port"`
	ServiceName       string `json:"service_name"`
	ServiceAddress    string `json:"service_address"`
	ConsulAddress     string `json:"consul_address"`
	RedisQueueURL     string `json:"redis_queue_url"`
	StorageServiceURL string `json:"storage_service_url"`
	WorkerCount       int    `json:"worker_count"`
}

func Load() *Config {
	cfg := &Config{
		Port:              getEnvInt("PORT", 8083),
		ServiceName:       getEnv("SERVICE_NAME", "task-processor"),
		ServiceAddress:    getEnv("SERVICE_ADDRESS", "task-processor"),
		ConsulAddress:     getEnv("CONSUL_ADDRESS", "consul:8500"),
		RedisQueueURL:     getEnv("REDIS_QUEUE_URL", "redis://redis-queue:6379"),
		StorageServiceURL: getEnv("STORAGE_SERVICE_URL", "http://storage:8082"),
		WorkerCount:       getEnvInt("WORKER_COUNT", 3),
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
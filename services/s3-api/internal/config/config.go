package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               int    `json:"port"`
	ServiceName        string `json:"service_name"`
	ServiceAddress     string `json:"service_address"`
	ConsulAddress      string `json:"consul_address"`
	MetadataServiceURL string `json:"metadata_service_url"`
	StorageServiceURL  string `json:"storage_service_url"`
	TaskServiceURL     string `json:"task_service_url"`
}

func Load() *Config {
	cfg := &Config{
		Port:               getEnvInt("PORT", 8080),
		ServiceName:        getEnv("SERVICE_NAME", "s3-api"),
		ServiceAddress:     getEnv("SERVICE_ADDRESS", "s3-api"),
		ConsulAddress:      getEnv("CONSUL_ADDRESS", "consul:8500"),
		MetadataServiceURL: getEnv("METADATA_SERVICE_URL", "http://metadata:8081"),
		StorageServiceURL:  getEnv("STORAGE_SERVICE_URL", "http://storage:8082"),
		TaskServiceURL:     getEnv("TASK_SERVICE_URL", "http://task-processor:8083"),
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
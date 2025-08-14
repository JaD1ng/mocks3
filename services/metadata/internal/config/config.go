package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port            int    `json:"port"`
	ServiceName     string `json:"service_name"`
	ServiceAddress  string `json:"service_address"`
	ConsulAddress   string `json:"consul_address"`
	PostgresURL     string `json:"postgres_url"`
	RedisCacheURL   string `json:"redis_cache_url"`
}

func Load() *Config {
	cfg := &Config{
		Port:            getEnvInt("PORT", 8081),
		ServiceName:     getEnv("SERVICE_NAME", "metadata"),
		ServiceAddress:  getEnv("SERVICE_ADDRESS", "metadata"),
		ConsulAddress:   getEnv("CONSUL_ADDRESS", "consul:8500"),
		PostgresURL:     getEnv("POSTGRES_URL", "postgres://micro_s3:micro_s3_password@postgres:5432/micro_s3?sslmode=disable"),
		RedisCacheURL:   getEnv("REDIS_CACHE_URL", "redis://redis-cache:6379"),
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
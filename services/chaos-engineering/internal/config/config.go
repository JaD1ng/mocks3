package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           int    `json:"port"`
	ServiceName    string `json:"service_name"`
	ServiceAddress string `json:"service_address"`
	ConsulAddress  string `json:"consul_address"`
	RulesDir       string `json:"rules_dir"`
}

func Load() *Config {
	cfg := &Config{
		Port:           getEnvInt("PORT", 8084),
		ServiceName:    getEnv("SERVICE_NAME", "chaos-engineering"),
		ServiceAddress: getEnv("SERVICE_ADDRESS", "chaos-engineering"),
		ConsulAddress:  getEnv("CONSUL_ADDRESS", "consul:8500"),
		RulesDir:       getEnv("RULES_DIR", "/app/rules"),
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
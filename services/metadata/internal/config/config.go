package config

import (
	"fmt"
	"mocks3/shared/utils"
)

// Config 元数据服务配置
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	LogLevel string         `json:"log_level"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode"`
}

// GetAddress 获取服务器地址
func (s *ServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// GetDSN 获取数据库连接字符串
func (d *DatabaseConfig) GetDSN() string {
	switch d.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			d.Host, d.Port, d.Username, d.Password, d.Database, d.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			d.Username, d.Password, d.Host, d.Port, d.Database)
	case "sqlite3":
		return d.Database
	default:
		return ""
	}
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:        utils.GetEnv("SERVER_HOST", "0.0.0.0"),
			Port:        utils.GetEnvInt("SERVICE_PORT", 8081),
			Environment: utils.GetEnv("ENVIRONMENT", "development"),
			Version:     utils.GetEnv("SERVICE_VERSION", "1.0.0"),
		},
		Database: DatabaseConfig{
			Driver:   utils.GetEnv("DB_DRIVER", "postgres"),
			Host:     utils.GetEnv("DB_HOST", "localhost"),
			Port:     utils.GetEnvInt("DB_PORT", 5432),
			Username: utils.GetEnv("DB_USERNAME", "postgres"),
			Password: utils.GetEnv("DB_PASSWORD", "password"),
			Database: utils.GetEnv("DB_DATABASE", "mocks3_metadata"),
			SSLMode:  utils.GetEnv("DB_SSL_MODE", "disable"),
		},
		LogLevel: utils.GetEnv("LOG_LEVEL", "info"),
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Driver == "" {
		return fmt.Errorf("database driver is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	return nil
}

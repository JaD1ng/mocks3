package config

import (
	"fmt"
	"mocks3/shared/utils"
)

// Config 元数据服务配置
type Config struct {
	Server   ServerConfig   `yaml:"server" json:"server"`
	Database DatabaseConfig `yaml:"database" json:"database"`
	LogLevel string         `yaml:"log_level" json:"log_level"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	Environment string `yaml:"environment" json:"environment"`
	Version     string `yaml:"version" json:"version"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `yaml:"driver" json:"driver"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	Database string `yaml:"database" json:"database"`
	SSLMode  string `yaml:"ssl_mode" json:"ssl_mode"`
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
	// 默认配置
	config := &Config{
		Server: ServerConfig{
			Host:        "0.0.0.0",
			Port:        8081,
			Environment: "development",
			Version:     "1.0.0",
		},
		Database: DatabaseConfig{
			Driver:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "password",
			Database: "mocks3_metadata",
			SSLMode:  "disable",
		},
		LogLevel: "info",
	}

	// 尝试从YAML文件加载配置
	if err := utils.LoadServiceConfig("metadata", config); err != nil {
		// 如果YAML配置文件不存在，使用默认配置
		fmt.Printf("Warning: Failed to load YAML config, using defaults: %v\n", err)
	}

	return config
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

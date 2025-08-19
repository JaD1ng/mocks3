package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetEnv 获取环境变量，如果不存在则返回默认值
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt 获取整数类型的环境变量
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool 获取布尔类型的环境变量
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetEnvDuration 获取时间间隔类型的环境变量
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetEnvFloat64 获取浮点数类型的环境变量
func GetEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// GetEnvStringSlice 获取字符串切片类型的环境变量（逗号分隔）
func GetEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	ServiceName    string        `json:"service_name"`
	ServicePort    int           `json:"service_port"`
	ServiceVersion string        `json:"service_version"`
	Environment    string        `json:"environment"`
	LogLevel       string        `json:"log_level"`
	ConsulAddr     string        `json:"consul_addr"`
	OTLPEndpoint   string        `json:"otlp_endpoint"`
	DatabaseURL    string        `json:"database_url"`
	RedisURL       string        `json:"redis_url"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"max_retries"`
}

// LoadServiceConfig 加载服务配置
func LoadServiceConfig(serviceName string) *ServiceConfig {
	return &ServiceConfig{
		ServiceName:    serviceName,
		ServicePort:    GetEnvInt("SERVICE_PORT", 8080),
		ServiceVersion: GetEnv("SERVICE_VERSION", "1.0.0"),
		Environment:    GetEnv("ENVIRONMENT", "development"),
		LogLevel:       GetEnv("LOG_LEVEL", "info"),
		ConsulAddr:     GetEnv("CONSUL_ADDR", "localhost:8500"),
		OTLPEndpoint:   GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318"),
		DatabaseURL:    GetEnv("DATABASE_URL", ""),
		RedisURL:       GetEnv("REDIS_URL", "redis://localhost:6379"),
		Timeout:        GetEnvDuration("DEFAULT_TIMEOUT", 30*time.Second),
		MaxRetries:     GetEnvInt("MAX_RETRIES", 3),
	}
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string        `json:"driver"`
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Username        string        `json:"username"`
	Password        string        `json:"password"`
	Database        string        `json:"database"`
	SSLMode         string        `json:"ssl_mode"`
	MaxConnections  int           `json:"max_connections"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

// LoadDatabaseConfig 加载数据库配置
func LoadDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Driver:          GetEnv("DB_DRIVER", "postgres"),
		Host:            GetEnv("DB_HOST", "localhost"),
		Port:            GetEnvInt("DB_PORT", 5432),
		Username:        GetEnv("DB_USERNAME", "postgres"),
		Password:        GetEnv("DB_PASSWORD", ""),
		Database:        GetEnv("DB_DATABASE", "mocks3"),
		SSLMode:         GetEnv("DB_SSL_MODE", "disable"),
		MaxConnections:  GetEnvInt("DB_MAX_CONNECTIONS", 25),
		MaxIdleConns:    GetEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: GetEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}
}

// GetDSN 获取数据库连接字符串
func (dc *DatabaseConfig) GetDSN() string {
	switch dc.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			dc.Host, dc.Port, dc.Username, dc.Password, dc.Database, dc.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			dc.Username, dc.Password, dc.Host, dc.Port, dc.Database)
	case "sqlite3":
		return dc.Database
	default:
		return ""
	}
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Password     string        `json:"password"`
	Database     int           `json:"database"`
	PoolSize     int           `json:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

// LoadRedisConfig 加载Redis配置
func LoadRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:         GetEnv("REDIS_HOST", "localhost"),
		Port:         GetEnvInt("REDIS_PORT", 6379),
		Password:     GetEnv("REDIS_PASSWORD", ""),
		Database:     GetEnvInt("REDIS_DATABASE", 0),
		PoolSize:     GetEnvInt("REDIS_POOL_SIZE", 10),
		MinIdleConns: GetEnvInt("REDIS_MIN_IDLE_CONNS", 2),
		DialTimeout:  GetEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:  GetEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
		WriteTimeout: GetEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
	}
}

// GetRedisAddr 获取Redis地址
func (rc *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", rc.Host, rc.Port)
}

// ParseConfigFromJSON 从JSON解析配置
func ParseConfigFromJSON(jsonData []byte, config interface{}) error {
	return json.Unmarshal(jsonData, config)
}

// ConfigToJSON 将配置转换为JSON
func ConfigToJSON(config interface{}) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// ValidateConfig 验证配置
func ValidateConfig(config interface{}) error {
	// 这里可以添加配置验证逻辑
	// 例如检查必填字段、格式验证等
	return nil
}

// LoadConfigFromFile 从文件加载配置
func LoadConfigFromFile(filename string, config interface{}) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return ParseConfigFromJSON(data, config)
}

// SaveConfigToFile 将配置保存到文件
func SaveConfigToFile(filename string, config interface{}) error {
	data, err := ConfigToJSON(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// MergeConfigs 合并配置（环境变量优先）
func MergeConfigs(defaultConfig, envConfig interface{}) interface{} {
	// 这里可以实现配置合并逻辑
	// 简单实现：环境变量配置优先于默认配置
	return envConfig
}

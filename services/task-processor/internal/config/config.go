package config

import (
	"os"
	"strconv"
)

// Config 任务处理器配置结构
// 包含服务运行所需的所有配置项，支持环境变量覆盖
type Config struct {
	Port              int    `json:"port"`               // 服务监听端口
	ServiceName       string `json:"service_name"`       // 服务名称，用于服务发现
	ServiceAddress    string `json:"service_address"`    // 服务地址，用于服务注册
	ConsulAddress     string `json:"consul_address"`     // Consul注册中心地址
	RedisQueueURL     string `json:"redis_queue_url"`    // Redis队列连接URL
	StorageServiceURL string `json:"storage_service_url"` // 存储服务API地址
	WorkerCount       int    `json:"worker_count"`       // 工作协程数量
}

// Load 加载配置
// 从环境变量加载配置，如果环境变量不存在则使用默认值
// 返回: 初始化后的配置对象
func Load() *Config {
	// 创建配置对象并设置默认值
	cfg := &Config{
		Port:              getEnvInt("PORT", 8083),                                      // 默认端口8083
		ServiceName:       getEnv("SERVICE_NAME", "task-processor"),                     // 默认服务名
		ServiceAddress:    getEnv("SERVICE_ADDRESS", "task-processor"),                  // 默认服务地址
		ConsulAddress:     getEnv("CONSUL_ADDRESS", "consul:8500"),                      // 默认Consul地址
		RedisQueueURL:     getEnv("REDIS_QUEUE_URL", "redis://redis-queue:6379"),       // 默认Redis队列地址
		StorageServiceURL: getEnv("STORAGE_SERVICE_URL", "http://storage:8082"),         // 默认存储服务地址
		WorkerCount:       getEnvInt("WORKER_COUNT", 3),                                // 默认3个工作协程
	}

	return cfg
}

// getEnv 获取字符串类型的环境变量
// key: 环境变量名
// defaultValue: 默认值
// 返回: 环境变量值或默认值
func getEnv(key, defaultValue string) string {
	// 尝试获取环境变量
	if value := os.Getenv(key); value != "" {
		return value
	}
	// 环境变量不存在或为空，返回默认值
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
// key: 环境变量名
// defaultValue: 默认值
// 返回: 环境变量值（转换为整数）或默认值
func getEnvInt(key string, defaultValue int) int {
	// 尝试获取环境变量
	if value := os.Getenv(key); value != "" {
		// 尝试转换为整数
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		// 转换失败时使用默认值
	}
	// 环境变量不存在或转换失败，返回默认值
	return defaultValue
}
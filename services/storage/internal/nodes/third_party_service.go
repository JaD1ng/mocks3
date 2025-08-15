package nodes

import (
	"fmt"
	"math/rand"
	"mocks3/shared/utils"
	"time"
)

// MockThirdPartyService 模拟第三方服务
type MockThirdPartyService struct {
	name    string
	baseURL string
}

// NewMockThirdPartyService 创建模拟第三方服务
func NewMockThirdPartyService() *MockThirdPartyService {
	return &MockThirdPartyService{
		name:    "mock-third-party-service",
		baseURL: "http://mock-third-party.example.com/api",
	}
}

// GetObject 获取对象（模拟实现）
func (s *MockThirdPartyService) GetObject(key string) (*FileObject, error) {
	// 模拟网络延迟
	time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)

	// 模拟获取失败的情况（20% 概率）
	if rand.Float32() < 0.2 {
		return nil, fmt.Errorf("third party service unavailable for key: %s", key)
	}

	// 生成模拟数据
	mockData := fmt.Sprintf("Mock data for key: %s\nGenerated at: %s\nFrom: %s",
		key, time.Now().Format(time.RFC3339), s.name)

	data := []byte(mockData)

	// 计算 MD5 哈希
	md5Hash := utils.CalculateMD5(data)

	// 解析 bucket 信息
	bucket := ""
	if len(key) > 0 && key[0] != '/' {
		parts := splitKey(key)
		if len(parts) > 0 {
			bucket = parts[0]
		}
	}

	return &FileObject{
		Key:         key,
		Bucket:      bucket,
		Size:        int64(len(data)),
		ContentType: "text/plain",
		MD5Hash:     md5Hash,
		Data:        data,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// GetName 获取服务名称
func (s *MockThirdPartyService) GetName() string {
	return s.name
}

// SetBaseURL 设置基础 URL（用于配置）
func (s *MockThirdPartyService) SetBaseURL(url string) {
	s.baseURL = url
}

// GetBaseURL 获取基础 URL
func (s *MockThirdPartyService) GetBaseURL() string {
	return s.baseURL
}

// IsAvailable 检查服务可用性
func (s *MockThirdPartyService) IsAvailable() bool {
	// 模拟服务可用性检查（90% 可用）
	return rand.Float32() < 0.9
}

// GetLatency 获取模拟延迟
func (s *MockThirdPartyService) GetLatency() time.Duration {
	// 模拟 50-200ms 的网络延迟
	return time.Duration(rand.Intn(150)+50) * time.Millisecond
}

// splitKey 分割 key 获取各个部分
func splitKey(key string) []string {
	if key == "" {
		return []string{}
	}

	var parts []string
	current := ""

	for _, char := range key {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// HTTPThirdPartyService 真实的 HTTP 第三方服务（用于实际环境）
type HTTPThirdPartyService struct {
	name    string
	baseURL string
	timeout time.Duration
}

// NewHTTPThirdPartyService 创建 HTTP 第三方服务
func NewHTTPThirdPartyService(name, baseURL string, timeout time.Duration) *HTTPThirdPartyService {
	return &HTTPThirdPartyService{
		name:    name,
		baseURL: baseURL,
		timeout: timeout,
	}
}

// GetObject 从真实的 HTTP 服务获取对象
func (s *HTTPThirdPartyService) GetObject(key string) (*FileObject, error) {
	// TODO: 实现真实的 HTTP 请求
	// 这里只是占位符，实际实现需要:
	// 1. 发送 HTTP GET 请求到 s.baseURL + key
	// 2. 解析响应数据
	// 3. 构造 FileObject 返回

	return nil, fmt.Errorf("HTTP third party service not implemented")
}

// GetName 获取服务名称
func (s *HTTPThirdPartyService) GetName() string {
	return s.name
}

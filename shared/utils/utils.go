package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// CalculateMD5 计算MD5哈希
func CalculateMD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// GenerateID 生成UUID
func GenerateID() string {
	return uuid.New().String()
}

// RandomString 生成随机字符串
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// ParseJSON 解析JSON
func ParseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// ToJSON 转换为JSON
func ToJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

// ToJSONString 转换为JSON字符串
func ToJSONString(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// Contains 检查字符串切片是否包含指定元素
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Remove 从字符串切片中移除指定元素
func Remove(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// GetCurrentTime 获取当前时间
func GetCurrentTime() time.Time {
	return time.Now()
}

// FormatTime 格式化时间
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseTime 解析时间
func ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// Min 返回两个整数的最小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max 返回两个整数的最大值
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// BuildObjectKey 构建对象键（包含bucket前缀）
func BuildObjectKey(bucket, key string) string {
	return bucket + "/" + key
}

// ExtractObjectKey 从完整key中提取对象key
func ExtractObjectKey(fullKey, bucketPrefix string) string {
	prefixWithSlash := bucketPrefix + "/"
	if len(fullKey) > len(prefixWithSlash) && fullKey[:len(prefixWithSlash)] == prefixWithSlash {
		return fullKey[len(prefixWithSlash):]
	}
	return fullKey
}

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache Redis 缓存客户端
type RedisCache struct {
	client *redis.Client
}

// NewRedisClient 创建 Redis 客户端
func NewRedisClient(redisURL string) (*RedisCache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return &RedisCache{client: client}, nil
}

// GetMetadata 从缓存获取元数据
func (r *RedisCache) GetMetadata(ctx context.Context, key string) (*MetadataEntry, error) {
	cacheKey := fmt.Sprintf("metadata:%s", key)
	data, err := r.client.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, err
	}

	var entry MetadataEntry
	err = json.Unmarshal([]byte(data), &entry)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// SetMetadata 设置元数据缓存
func (r *RedisCache) SetMetadata(ctx context.Context, key string, entry *MetadataEntry) error {
	cacheKey := fmt.Sprintf("metadata:%s", key)
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// 设置缓存，过期时间 1 小时
	return r.client.Set(ctx, cacheKey, data, time.Hour).Err()
}

// DeleteMetadata 删除元数据缓存
func (r *RedisCache) DeleteMetadata(ctx context.Context, key string) error {
	cacheKey := fmt.Sprintf("metadata:%s", key)
	return r.client.Del(ctx, cacheKey).Err()
}

// SetStats 设置统计信息缓存
func (r *RedisCache) SetStats(ctx context.Context, stats map[string]any) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	// 统计信息缓存 5 分钟
	return r.client.Set(ctx, "stats", data, 5*time.Minute).Err()
}

// GetStats 获取统计信息缓存
func (r *RedisCache) GetStats(ctx context.Context) (map[string]any, error) {
	data, err := r.client.Get(ctx, "stats").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, err
	}

	var stats map[string]any
	err = json.Unmarshal([]byte(data), &stats)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// IncrementCounter 增加计数器
func (r *RedisCache) IncrementCounter(ctx context.Context, key string) error {
	return r.client.Incr(ctx, key).Err()
}

// GetCounter 获取计数器值
func (r *RedisCache) GetCounter(ctx context.Context, key string) (int64, error) {
	return r.client.Get(ctx, key).Int64()
}

// SetExpire 设置过期时间
func (r *RedisCache) SetExpire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// Close 关闭 Redis 连接
func (r *RedisCache) Close() error {
	return r.client.Close()
}

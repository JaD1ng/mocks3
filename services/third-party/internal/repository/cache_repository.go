package repository

import (
	"context"
	"fmt"
	"mocks3/shared/models"
	"mocks3/services/third-party/internal/config"
	"sync"
	"time"
)

// CacheItem 缓存项
type CacheItem struct {
	Object    *models.Object `json:"object"`
	CachedAt  time.Time      `json:"cached_at"`
	ExpiresAt time.Time      `json:"expires_at"`
	AccessCount int64        `json:"access_count"`
	LastAccess  time.Time    `json:"last_access"`
}

// IsExpired 检查是否过期
func (item *CacheItem) IsExpired() bool {
	return time.Now().After(item.ExpiresAt)
}

// Touch 更新访问时间和次数
func (item *CacheItem) Touch() {
	item.LastAccess = time.Now()
	item.AccessCount++
}

// CacheRepository 缓存仓库
type CacheRepository struct {
	cache    map[string]*CacheItem
	mu       sync.RWMutex
	config   *config.CacheConfig
	stats    *CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
	Hits        int64 `json:"hits"`
	Misses      int64 `json:"misses"`
	Evictions   int64 `json:"evictions"`
	TotalSize   int64 `json:"total_size"`
	ItemCount   int64 `json:"item_count"`
	LastCleanup time.Time `json:"last_cleanup"`
}

// NewCacheRepository 创建缓存仓库
func NewCacheRepository(config *config.CacheConfig) *CacheRepository {
	repo := &CacheRepository{
		cache:  make(map[string]*CacheItem),
		config: config,
		stats:  &CacheStats{},
	}
	
	// 启动清理goroutine
	if config.Enabled {
		go repo.cleanupExpired()
	}
	
	return repo
}

// Set 设置缓存
func (r *CacheRepository) Set(ctx context.Context, bucket, key string, object *models.Object) error {
	if !r.config.Enabled {
		return nil
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	cacheKey := r.buildKey(bucket, key)
	
	// 检查是否需要清理空间
	if r.needEviction(object) {
		r.evictLRU()
	}
	
	item := &CacheItem{
		Object:      object,
		CachedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(r.config.TTL) * time.Second),
		AccessCount: 0,
		LastAccess:  time.Now(),
	}
	
	r.cache[cacheKey] = item
	r.stats.ItemCount++
	r.stats.TotalSize += object.Size
	
	return nil
}

// Get 获取缓存
func (r *CacheRepository) Get(ctx context.Context, bucket, key string) (*models.Object, error) {
	if !r.config.Enabled {
		return nil, fmt.Errorf("cache disabled")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	cacheKey := r.buildKey(bucket, key)
	
	item, exists := r.cache[cacheKey]
	if !exists {
		r.stats.Misses++
		return nil, fmt.Errorf("not found in cache")
	}
	
	if item.IsExpired() {
		delete(r.cache, cacheKey)
		r.stats.ItemCount--
		r.stats.TotalSize -= item.Object.Size
		r.stats.Misses++
		return nil, fmt.Errorf("cache expired")
	}
	
	item.Touch()
	r.stats.Hits++
	
	return item.Object, nil
}

// Delete 删除缓存
func (r *CacheRepository) Delete(ctx context.Context, bucket, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	cacheKey := r.buildKey(bucket, key)
	
	if item, exists := r.cache[cacheKey]; exists {
		delete(r.cache, cacheKey)
		r.stats.ItemCount--
		r.stats.TotalSize -= item.Object.Size
	}
	
	return nil
}

// Clear 清空缓存
func (r *CacheRepository) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.cache = make(map[string]*CacheItem)
	r.stats.ItemCount = 0
	r.stats.TotalSize = 0
	
	return nil
}

// GetStats 获取缓存统计
func (r *CacheRepository) GetStats() *CacheStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	statsCopy := *r.stats
	return &statsCopy
}

// buildKey 构建缓存键
func (r *CacheRepository) buildKey(bucket, key string) string {
	return fmt.Sprintf("%s/%s", bucket, key)
}

// needEviction 检查是否需要清理
func (r *CacheRepository) needEviction(object *models.Object) bool {
	maxSizeBytes := r.config.MaxSize * 1024 * 1024 // 转换为字节
	return r.stats.TotalSize+object.Size > maxSizeBytes
}

// evictLRU 清理最近最少使用的项
func (r *CacheRepository) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, item := range r.cache {
		if oldestKey == "" || item.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.LastAccess
		}
	}
	
	if oldestKey != "" {
		if item := r.cache[oldestKey]; item != nil {
			delete(r.cache, oldestKey)
			r.stats.ItemCount--
			r.stats.TotalSize -= item.Object.Size
			r.stats.Evictions++
		}
	}
}

// cleanupExpired 清理过期项
func (r *CacheRepository) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		
		for key, item := range r.cache {
			if item.IsExpired() {
				delete(r.cache, key)
				r.stats.ItemCount--
				r.stats.TotalSize -= item.Object.Size
			}
		}
		
		r.stats.LastCleanup = now
		r.mu.Unlock()
	}
}
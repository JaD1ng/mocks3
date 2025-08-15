package handler

import (
	"metadata/internal/db"
	"github.com/mocks3/shared/logger"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MetadataHandler 元数据处理器
// 提供元数据的CRUD操作，支持数据库和缓存二级存储
type MetadataHandler struct {
	repo   *db.MetadataRepository // 数据库仓库层
	cache  *db.RedisCache         // Redis缓存层
	logger logger.Logger          // 日志记录器
}

// NewMetadataHandler 创建元数据处理器
// database: GORM数据库连接
// cache: Redis缓存实例
// 返回: 初始化后的元数据处理器
func NewMetadataHandler(database *gorm.DB, cache *db.RedisCache) *MetadataHandler {
	return &MetadataHandler{
		repo:   db.NewMetadataRepository(database),
		cache:  cache,
		logger: logger.DefaultLogger,
	}
}

// NewMetadataHandlerWithLogger 创建元数据处理器（带自定义日志器）
// database: GORM数据库连接
// cache: Redis缓存实例
// log: 自定义日志记录器
// 返回: 初始化后的元数据处理器
func NewMetadataHandlerWithLogger(database *gorm.DB, cache *db.RedisCache, log logger.Logger) *MetadataHandler {
	return &MetadataHandler{
		repo:   db.NewMetadataRepository(database),
		cache:  cache,
		logger: log,
	}
}

// HealthCheck 健康检查
// c: Gin上下文对象
// 返回: 服务健康状态信息
func (h *MetadataHandler) HealthCheck(c *gin.Context) {
	// 返回服务健康状态和时间戳
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "metadata",
		"timestamp": time.Now(),
	})
}

// SaveMetadata 保存元数据
// c: Gin上下文对象
// 功能: 解析JSON请求，保存到数据库并更新缓存
func (h *MetadataHandler) SaveMetadata(c *gin.Context) {
	// 解析请求体中的JSON数据
	var entry db.MetadataEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 保存到数据库
	err := h.repo.Save(&entry)
	if err != nil {
		ctx := c.Request.Context()
		h.logger.Error(ctx, "保存元数据失败", err, map[string]any{
			"key": entry.Key,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存元数据失败: " + err.Error(),
		})
		return
	}

	// 更新Redis缓存（失败不影响主流程）
	ctx := c.Request.Context()
	err = h.cache.SetMetadata(ctx, entry.Key, &entry)
	if err != nil {
		h.logger.Error(ctx, "更新缓存失败", err, map[string]any{
			"key": entry.Key,
		})
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "元数据保存成功",
		"data":    entry,
	})
}

// GetMetadata 获取元数据
// c: Gin上下文对象
// 功能: 通过key获取元数据，优先从缓存查找，未命中则查询数据库
func (h *MetadataHandler) GetMetadata(c *gin.Context) {
	// 验证路径参数
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "key 参数不能为空",
		})
		return
	}

	ctx := c.Request.Context()

	// 尝试从缓存获取元数据
	entry, err := h.tryGetFromCache(c, key)
	if err != nil {
		// 缓存错误不影响主流程，记录日志后继续
		h.logger.Error(ctx, "缓存查询失败", err, map[string]any{"key": key})
	}

	// 缓存未命中，从数据库获取
	if entry == nil {
		entry, err = h.getFromDatabaseAndCache(c, key)
		if err != nil {
			// 根据错误类型返回不同的HTTP状态码
			h.handleDatabaseError(c, err, key)
			return
		}
	}

	// 返回查询结果
	c.JSON(http.StatusOK, entry)
}

// tryGetFromCache 尝试从缓存获取元数据（私有方法）
// c: Gin上下文对象
// key: 元数据键
// 返回: 元数据对象和错误信息
func (h *MetadataHandler) tryGetFromCache(c *gin.Context, key string) (*db.MetadataEntry, error) {
	ctx := c.Request.Context()
	return h.cache.GetMetadata(ctx, key)
}

// getFromDatabaseAndCache 从数据库获取并更新缓存（私有方法）
// c: Gin上下文对象
// key: 元数据键
// 返回: 元数据对象和错误信息
func (h *MetadataHandler) getFromDatabaseAndCache(c *gin.Context, key string) (*db.MetadataEntry, error) {
	// 从数据库查询
	entry, err := h.repo.GetByKey(key)
	if err != nil {
		return nil, err
	}

	// 更新缓存（失败不影响主流程）
	ctx := c.Request.Context()
	if cacheErr := h.cache.SetMetadata(ctx, key, entry); cacheErr != nil {
		h.logger.Error(ctx, "更新缓存失败", cacheErr, map[string]any{"key": key})
	}

	return entry, nil
}

// handleDatabaseError 处理数据库错误（私有方法）
// c: Gin上下文对象
// err: 数据库错误
// key: 元数据键
func (h *MetadataHandler) handleDatabaseError(c *gin.Context, err error, key string) {
	if err == gorm.ErrRecordNotFound {
		// 记录不存在
		c.JSON(http.StatusNotFound, gin.H{
			"error": "元数据不存在",
		})
	} else {
		// 其他数据库错误
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询元数据失败: " + err.Error(),
		})
	}
}

// UpdateMetadata 更新元数据
func (h *MetadataHandler) UpdateMetadata(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "key 参数不能为空",
		})
		return
	}

	var updates map[string]any
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 更新数据库
	err := h.repo.Update(key, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新元数据失败: " + err.Error(),
		})
		return
	}

	// 删除缓存，强制下次从数据库获取最新数据
	ctx := c.Request.Context()
	err = h.cache.DeleteMetadata(ctx, key)
	if err != nil {
		h.logger.Error(ctx, "删除缓存失败", err, map[string]any{
			"key": key,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "元数据更新成功",
	})
}

// DeleteMetadata 删除元数据
func (h *MetadataHandler) DeleteMetadata(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "key 参数不能为空",
		})
		return
	}

	// 从数据库删除
	err := h.repo.Delete(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除元数据失败: " + err.Error(),
		})
		return
	}

	// 删除缓存
	ctx := c.Request.Context()
	err = h.cache.DeleteMetadata(ctx, key)
	if err != nil {
		h.logger.Error(ctx, "删除缓存失败", err, map[string]any{
			"key": key,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "元数据删除成功",
	})
}

// ListMetadata 列出元数据
func (h *MetadataHandler) ListMetadata(c *gin.Context) {
	// 分页参数
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	entries, err := h.repo.List(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询元数据列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   entries,
		"limit":  limit,
		"offset": offset,
		"count":  len(entries),
	})
}

// SearchMetadata 搜索元数据
func (h *MetadataHandler) SearchMetadata(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "查询参数 q 不能为空",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}

	entries, err := h.repo.Search(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "搜索元数据失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  entries,
		"query": query,
		"limit": limit,
		"count": len(entries),
	})
}

// GetStats 获取统计信息
func (h *MetadataHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	// 先尝试从缓存获取
	stats, err := h.cache.GetStats(ctx)
	if err != nil {
		h.logger.Error(ctx, "缓存查询统计信息失败", err, nil)
	}

	// 缓存未命中，从数据库获取
	if stats == nil {
		stats, err = h.repo.GetStats()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "获取统计信息失败: " + err.Error(),
			})
			return
		}

		// 更新缓存
		err = h.cache.SetStats(ctx, stats)
		if err != nil {
			h.logger.Error(ctx, "更新统计信息缓存失败", err, nil)
		}
	}

	c.JSON(http.StatusOK, stats)
}

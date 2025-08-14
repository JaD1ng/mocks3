package handler

import (
	"net/http"
	"strconv"
	"time"

	"metadata/internal/db"
	"micro-s3/shared/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MetadataHandler struct {
	repo   *db.MetadataRepository
	cache  *db.RedisCache
	logger logger.Logger
}

func NewMetadataHandler(database *gorm.DB, cache *db.RedisCache) *MetadataHandler {
	return &MetadataHandler{
		repo:   db.NewMetadataRepository(database),
		cache:  cache,
		logger: logger.DefaultLogger,
	}
}

func NewMetadataHandlerWithLogger(database *gorm.DB, cache *db.RedisCache, log logger.Logger) *MetadataHandler {
	return &MetadataHandler{
		repo:   db.NewMetadataRepository(database),
		cache:  cache,
		logger: log,
	}
}

// HealthCheck 健康检查
func (h *MetadataHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "metadata",
		"timestamp": time.Now(),
	})
}

// SaveMetadata 保存元数据
func (h *MetadataHandler) SaveMetadata(c *gin.Context) {
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

	// 更新缓存
	ctx := c.Request.Context()
	err = h.cache.SetMetadata(ctx, entry.Key, &entry)
	if err != nil {
		h.logger.Error(ctx, "更新缓存失败", err, map[string]any{
			"key": entry.Key,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "元数据保存成功",
		"data":    entry,
	})
}

// GetMetadata 获取元数据
func (h *MetadataHandler) GetMetadata(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "key 参数不能为空",
		})
		return
	}

	ctx := c.Request.Context()

	// 先尝试从缓存获取
	entry, err := h.cache.GetMetadata(ctx, key)
	if err != nil {
		h.logger.Error(ctx, "缓存查询失败", err, map[string]any{
			"key": key,
		})
	}

	// 缓存未命中，从数据库获取
	if entry == nil {
		entry, err = h.repo.GetByKey(key)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "元数据不存在",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "查询元数据失败: " + err.Error(),
			})
			return
		}

		// 更新缓存
		err = h.cache.SetMetadata(ctx, key, entry)
		if err != nil {
			h.logger.Error(ctx, "更新缓存失败", err, map[string]any{
				"key": key,
			})
		}
	}

	c.JSON(http.StatusOK, entry)
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

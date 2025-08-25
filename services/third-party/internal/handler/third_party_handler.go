package handler

import (
	"net/http"
	"strconv"

	"mocks3/services/third-party/internal/service"
	"mocks3/shared/models"
	"mocks3/shared/observability"

	"github.com/gin-gonic/gin"
)

// ThirdPartyHandler 第三方服务处理器
type ThirdPartyHandler struct {
	service *service.ThirdPartyService
	logger  *observability.Logger
}

// NewThirdPartyHandler 创建第三方服务处理器
func NewThirdPartyHandler(service *service.ThirdPartyService, logger *observability.Logger) *ThirdPartyHandler {
	return &ThirdPartyHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *ThirdPartyHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// 对象操作
		api.GET("/objects/:bucket/:key", h.GetObject)
		api.POST("/objects", h.PutObject)
		api.DELETE("/objects/:bucket/:key", h.DeleteObject)
		api.GET("/objects", h.ListObjects)

		// 元数据操作
		api.GET("/metadata/:bucket/:key", h.GetObjectMetadata)

		// 数据源管理
		api.POST("/datasources", h.SetDataSource)
		api.GET("/datasources", h.GetDataSources)

		// 缓存管理
		api.POST("/cache", h.CacheObject)
		api.DELETE("/cache/:bucket/:key", h.InvalidateCache)

		// 统计信息
		api.GET("/stats", h.GetStats)
	}
}

// GetObject 获取对象
func (h *ThirdPartyHandler) GetObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if bucket == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bucket and key are required",
		})
		return
	}

	object, err := h.service.GetObject(c.Request.Context(), bucket, key)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Object not found",
			"bucket", bucket, "key", key, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Object not found",
		})
		return
	}

	c.JSON(http.StatusOK, object)
}

// PutObjectRequest 存储对象请求
type PutObjectRequest struct {
	Bucket      string `json:"bucket" binding:"required"`
	Key         string `json:"key" binding:"required"`
	ContentType string `json:"content_type"`
	Data        string `json:"data"` // base64编码的数据
}

// PutObject 存储对象
func (h *ThirdPartyHandler) PutObject(c *gin.Context) {
	var req PutObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// 解码数据
	var data []byte
	if req.Data != "" {
		// 这里应该解码base64数据，简化实现
		data = []byte(req.Data)
	}

	object := &models.Object{
		Bucket:      req.Bucket,
		Key:         req.Key,
		Size:        int64(len(data)),
		ContentType: req.ContentType,
		Data:        data,
	}

	if err := h.service.PutObject(c.Request.Context(), object); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to store object", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store object",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"bucket": object.Bucket,
		"key":    object.Key,
		"size":   object.Size,
	})
}

// DeleteObject 删除对象
func (h *ThirdPartyHandler) DeleteObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if bucket == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bucket and key are required",
		})
		return
	}

	if err := h.service.DeleteObject(c.Request.Context(), bucket, key); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to delete object", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete object",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Object deleted successfully",
	})
}

// ListObjects 列出对象
func (h *ThirdPartyHandler) ListObjects(c *gin.Context) {
	bucket := c.Query("bucket")
	prefix := c.Query("prefix")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	objects, err := h.service.ListObjects(c.Request.Context(), bucket, prefix, limit)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list objects", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list objects",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"objects": objects,
		"count":   len(objects),
	})
}

// GetObjectMetadata 获取对象元数据
func (h *ThirdPartyHandler) GetObjectMetadata(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if bucket == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bucket and key are required",
		})
		return
	}

	metadata, err := h.service.GetObjectMetadata(c.Request.Context(), bucket, key)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Metadata not found",
			"bucket", bucket, "key", key, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Metadata not found",
		})
		return
	}

	c.JSON(http.StatusOK, metadata)
}

// SetDataSourceRequest 设置数据源请求
type SetDataSourceRequest struct {
	Name   string `json:"name" binding:"required"`
	Config string `json:"config" binding:"required"`
}

// SetDataSource 设置数据源
func (h *ThirdPartyHandler) SetDataSource(c *gin.Context) {
	var req SetDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	if err := h.service.SetDataSource(c.Request.Context(), req.Name, req.Config); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to set data source", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to set data source",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Data source set successfully",
		"name":    req.Name,
	})
}

// GetDataSources 获取数据源列表
func (h *ThirdPartyHandler) GetDataSources(c *gin.Context) {
	dataSources, err := h.service.GetDataSources(c.Request.Context())
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to get data sources", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get data sources",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"datasources": dataSources,
		"count":       len(dataSources),
	})
}

// CacheObject 缓存对象
func (h *ThirdPartyHandler) CacheObject(c *gin.Context) {
	var object models.Object
	if err := c.ShouldBindJSON(&object); err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	if err := h.service.CacheObject(c.Request.Context(), &object); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to cache object", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to cache object",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Object cached successfully",
	})
}

// InvalidateCache 清除缓存
func (h *ThirdPartyHandler) InvalidateCache(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if bucket == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bucket and key are required",
		})
		return
	}

	if err := h.service.InvalidateCache(c.Request.Context(), bucket, key); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to invalidate cache", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to invalidate cache",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache invalidated successfully",
	})
}

// GetStats 获取统计信息
func (h *ThirdPartyHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to get stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

package handler

import (
	"net/http"
	"strconv"

	"mocks3/shared/interfaces"
	"mocks3/shared/models"
	"mocks3/shared/observability"
	"mocks3/shared/utils"

	"github.com/gin-gonic/gin"
)

// MetadataHandler 元数据处理器
type MetadataHandler struct {
	service interfaces.MetadataService
	logger  *observability.Logger
}

// NewMetadataHandler 创建元数据处理器
func NewMetadataHandler(service interfaces.MetadataService, logger *observability.Logger) *MetadataHandler {
	return &MetadataHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *MetadataHandler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		// 元数据CRUD操作
		v1.POST("/metadata", h.CreateMetadata)
		v1.GET("/metadata/:bucket/:key", h.GetMetadata)
		v1.PUT("/metadata/:bucket/:key", h.UpdateMetadata)
		v1.DELETE("/metadata/:bucket/:key", h.DeleteMetadata)

		// 列表和搜索
		v1.GET("/metadata", h.ListMetadata)
		v1.GET("/metadata/search", h.SearchMetadata)

		// 统计信息
		v1.GET("/stats", h.GetStats)
		v1.GET("/metadata/count", h.CountObjects)
	}
}

// CreateMetadata 创建元数据
func (h *MetadataHandler) CreateMetadata(c *gin.Context) {
	var metadata models.Metadata
	if err := c.ShouldBindJSON(&metadata); err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid request body", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.service.SaveMetadata(c.Request.Context(), &metadata); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to create metadata", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to create metadata: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    metadata,
		"message": "Metadata created successfully",
	})
}

// GetMetadata 获取元数据
func (h *MetadataHandler) GetMetadata(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	metadata, err := h.service.GetMetadata(c.Request.Context(), bucket, key)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Metadata not found",
			"bucket", bucket, "key", key, "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusNotFound, "Metadata not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metadata,
	})
}

// UpdateMetadata 更新元数据
func (h *MetadataHandler) UpdateMetadata(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	var metadata models.Metadata
	if err := c.ShouldBindJSON(&metadata); err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid request body", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// 确保URL参数与请求体一致
	metadata.Bucket = bucket
	metadata.Key = key

	if err := h.service.UpdateMetadata(c.Request.Context(), &metadata); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to update metadata", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to update metadata: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metadata,
		"message": "Metadata updated successfully",
	})
}

// DeleteMetadata 删除元数据
func (h *MetadataHandler) DeleteMetadata(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if err := h.service.DeleteMetadata(c.Request.Context(), bucket, key); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to delete metadata",
			"bucket", bucket, "key", key, "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to delete metadata: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Metadata deleted successfully",
	})
}

// ListMetadata 列出元数据
func (h *MetadataHandler) ListMetadata(c *gin.Context) {
	bucket := c.Query("bucket")
	prefix := c.Query("prefix")

	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		utils.SetErrorResponse(c.Writer, http.StatusBadRequest, "Invalid limit parameter")
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		utils.SetErrorResponse(c.Writer, http.StatusBadRequest, "Invalid offset parameter")
		return
	}

	metadataList, err := h.service.ListMetadata(c.Request.Context(), bucket, prefix, limit, offset)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list metadata", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to list metadata: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"metadata": metadataList,
			"count":    len(metadataList),
			"limit":    limit,
			"offset":   offset,
		},
	})
}

// SearchMetadata 搜索元数据
func (h *MetadataHandler) SearchMetadata(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		utils.SetErrorResponse(c.Writer, http.StatusBadRequest, "Search query is required")
		return
	}

	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		utils.SetErrorResponse(c.Writer, http.StatusBadRequest, "Invalid limit parameter")
		return
	}

	metadataList, err := h.service.SearchMetadata(c.Request.Context(), query, limit)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to search metadata", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to search metadata: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"query":    query,
			"metadata": metadataList,
			"count":    len(metadataList),
			"limit":    limit,
		},
	})
}

// GetStats 获取统计信息
func (h *MetadataHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to get stats", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to get statistics: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// CountObjects 计算对象数量
func (h *MetadataHandler) CountObjects(c *gin.Context) {
	bucket := c.Query("bucket")
	prefix := c.Query("prefix")

	count, err := h.service.CountObjects(c.Request.Context(), bucket, prefix)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to count objects", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to count objects: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"bucket": bucket,
			"prefix": prefix,
			"count":  count,
		},
	})
}

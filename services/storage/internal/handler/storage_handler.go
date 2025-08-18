package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"mocks3/shared/interfaces"
	"mocks3/shared/models"
	"mocks3/shared/observability/log"
	"mocks3/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StorageHandler 存储处理器
type StorageHandler struct {
	service interfaces.StorageService
	logger  *log.Logger
}

// NewStorageHandler 创建存储处理器
func NewStorageHandler(service interfaces.StorageService, logger *log.Logger) *StorageHandler {
	return &StorageHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *StorageHandler) RegisterRoutes(router *gin.Engine) {
	// S3兼容API
	router.PUT("/:bucket/:key", h.PutObject)
	router.GET("/:bucket/:key", h.GetObject)
	router.DELETE("/:bucket/:key", h.DeleteObject)
	router.HEAD("/:bucket/:key", h.HeadObject)
	router.GET("/:bucket", h.ListObjects)

	// 管理API
	v1 := router.Group("/api/v1")
	{
		v1.POST("/objects", h.CreateObject)
		v1.GET("/objects/:bucket/:key", h.GetObjectInfo)
		v1.DELETE("/objects/:bucket/:key", h.DeleteObjectAPI)
		v1.GET("/objects", h.ListObjectsAPI)
		v1.GET("/stats", h.GetStats)
	}
}

// PutObject S3兼容的PUT对象接口
func (h *StorageHandler) PutObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	// 读取请求体
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to read request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// 构建对象
	object := &models.Object{
		ID:          uuid.New().String(),
		Key:         key,
		Bucket:      bucket,
		Size:        int64(len(data)),
		ContentType: c.GetHeader("Content-Type"),
		Data:        data,
		Headers:     make(map[string]string),
		Tags:        make(map[string]string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 如果没有指定Content-Type，设置默认值
	if object.ContentType == "" {
		object.ContentType = "application/octet-stream"
	}

	// 复制相关的HTTP头
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			switch key {
			case "Content-MD5":
				object.MD5Hash = values[0]
			case "Cache-Control", "Content-Disposition", "Content-Encoding", "Content-Language":
				object.Headers[key] = values[0]
			}
		}
	}

	// 写入对象
	if err := h.service.WriteObject(c.Request.Context(), object); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to write object", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write object"})
		return
	}

	// 设置响应头
	c.Header("ETag", object.ETag)
	c.Header("Content-MD5", object.MD5Hash)
	
	c.Status(http.StatusOK)
}

// GetObject S3兼容的GET对象接口
func (h *StorageHandler) GetObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	object, err := h.service.ReadObject(c.Request.Context(), bucket, key)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Object not found", "bucket", bucket, "key", key)
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}

	// 设置响应头
	c.Header("Content-Type", object.ContentType)
	c.Header("Content-Length", strconv.FormatInt(object.Size, 10))
	c.Header("ETag", object.ETag)
	c.Header("Content-MD5", object.MD5Hash)
	c.Header("Last-Modified", object.UpdatedAt.Format(http.TimeFormat))

	// 设置自定义头
	for key, value := range object.Headers {
		c.Header(key, value)
	}

	// 返回文件数据
	c.Data(http.StatusOK, object.ContentType, object.Data)
}

// DeleteObject S3兼容的DELETE对象接口
func (h *StorageHandler) DeleteObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if err := h.service.DeleteObject(c.Request.Context(), bucket, key); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to delete object", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete object"})
		return
	}

	c.Status(http.StatusNoContent)
}

// HeadObject S3兼容的HEAD对象接口
func (h *StorageHandler) HeadObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	object, err := h.service.ReadObject(c.Request.Context(), bucket, key)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Object not found", "bucket", bucket, "key", key)
		c.Status(http.StatusNotFound)
		return
	}

	// 设置响应头（不返回body）
	c.Header("Content-Type", object.ContentType)
	c.Header("Content-Length", strconv.FormatInt(object.Size, 10))
	c.Header("ETag", object.ETag)
	c.Header("Content-MD5", object.MD5Hash)
	c.Header("Last-Modified", object.UpdatedAt.Format(http.TimeFormat))

	// 设置自定义头
	for key, value := range object.Headers {
		c.Header(key, value)
	}

	c.Status(http.StatusOK)
}

// ListObjects S3兼容的列表接口
func (h *StorageHandler) ListObjects(c *gin.Context) {
	bucket := c.Param("bucket")

	req := &models.ListObjectsRequest{
		Bucket:    bucket,
		Prefix:    c.Query("prefix"),
		Delimiter: c.Query("delimiter"),
		MaxKeys:   1000,
	}

	if maxKeysStr := c.Query("max-keys"); maxKeysStr != "" {
		if maxKeys, err := strconv.Atoi(maxKeysStr); err == nil && maxKeys > 0 {
			req.MaxKeys = maxKeys
		}
	}

	if startAfter := c.Query("start-after"); startAfter != "" {
		req.StartAfter = startAfter
	}

	response, err := h.service.ListObjects(c.Request.Context(), req)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list objects", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list objects"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CreateObject 管理API - 创建对象
func (h *StorageHandler) CreateObject(c *gin.Context) {
	var req models.UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid request body", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusBadRequest, "Invalid request body")
		return
	}

	object := &models.Object{
		ID:          uuid.New().String(),
		Key:         req.Key,
		Bucket:      req.Bucket,
		Size:        int64(len(req.Data)),
		ContentType: req.ContentType,
		Data:        req.Data,
		Headers:     req.Headers,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if object.ContentType == "" {
		object.ContentType = "application/octet-stream"
	}

	if object.Headers == nil {
		object.Headers = make(map[string]string)
	}

	if object.Tags == nil {
		object.Tags = make(map[string]string)
	}

	if err := h.service.WriteObject(c.Request.Context(), object); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to create object", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to create object")
		return
	}

	response := &models.UploadResponse{
		Success:   true,
		ObjectID:  object.ID,
		Key:       object.Key,
		Bucket:    object.Bucket,
		Size:      object.Size,
		MD5Hash:   object.MD5Hash,
		ETag:      object.ETag,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusCreated, response)
}

// GetObjectInfo 管理API - 获取对象信息
func (h *StorageHandler) GetObjectInfo(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	object, err := h.service.ReadObject(c.Request.Context(), bucket, key)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Object not found", "bucket", bucket, "key", key)
		utils.SetErrorResponse(c.Writer, http.StatusNotFound, "Object not found")
		return
	}

	// 返回对象信息（不包含数据）
	objectInfo := &models.ObjectInfo{
		ID:          object.ID,
		Key:         object.Key,
		Bucket:      object.Bucket,
		Size:        object.Size,
		ContentType: object.ContentType,
		MD5Hash:     object.MD5Hash,
		ETag:        object.ETag,
		Headers:     object.Headers,
		Tags:        object.Tags,
		CreatedAt:   object.CreatedAt,
		UpdatedAt:   object.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    objectInfo,
	})
}

// DeleteObjectAPI 管理API - 删除对象
func (h *StorageHandler) DeleteObjectAPI(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")

	if err := h.service.DeleteObject(c.Request.Context(), bucket, key); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to delete object", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to delete object")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Object deleted successfully",
	})
}

// ListObjectsAPI 管理API - 列出对象
func (h *StorageHandler) ListObjectsAPI(c *gin.Context) {
	bucket := c.Query("bucket")
	prefix := c.Query("prefix")
	
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		utils.SetErrorResponse(c.Writer, http.StatusBadRequest, "Invalid limit parameter")
		return
	}

	req := &models.ListObjectsRequest{
		Bucket:  bucket,
		Prefix:  prefix,
		MaxKeys: limit,
	}

	response, err := h.service.ListObjects(c.Request.Context(), req)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list objects", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to list objects")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetStats 获取存储统计信息
func (h *StorageHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to get stats", "error", err)
		utils.SetErrorResponse(c.Writer, http.StatusInternalServerError, "Failed to get statistics")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
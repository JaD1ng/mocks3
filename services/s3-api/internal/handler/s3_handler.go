package handler

import (
	"fmt"
	"io"
	"mocks3/services/s3-api/internal/client"
	"github.com/mocks3/shared/logger"
	"github.com/mocks3/shared/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type S3Handler struct {
	clients *client.Clients
	logger  logger.Logger
}

func NewS3Handler(clients *client.Clients) *S3Handler {
	return &S3Handler{
		clients: clients,
		logger:  logger.DefaultLogger,
	}
}

// NewS3HandlerWithLogger 创建带指定日志器的S3Handler
func NewS3HandlerWithLogger(clients *client.Clients, l logger.Logger) *S3Handler {
	return &S3Handler{
		clients: clients,
		logger:  l,
	}
}

// HealthCheck 健康检查
func (h *S3Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "s3-api",
		"timestamp": time.Now(),
	})
}

// PutObject 上传对象
func (h *S3Handler) PutObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")
	objectKey := fmt.Sprintf("%s/%s", bucket, key)

	// 读取请求体
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("读取请求体失败: %v", err),
		})
		return
	}

	// 获取Content-Type
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 计算MD5哈希
	md5Hash := utils.CalculateMD5(data)

	// 创建文件对象
	fileObj := &client.FileObject{
		ID:          utils.GenerateID(),
		Key:         objectKey,
		Bucket:      bucket,
		Size:        int64(len(data)),
		ContentType: contentType,
		MD5Hash:     md5Hash,
		Data:        data,
		CreatedAt:   time.Now(),
	}

	// 写入存储
	ctx := c.Request.Context()
	err = h.clients.Storage.WriteObject(ctx, fileObj)
	if err != nil {
		h.logger.Error(c.Request.Context(), "存储写入失败", err, map[string]any{
			"bucket": bucket,
			"key":    key,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("存储写入失败: %v", err),
		})
		return
	}

	// 保存元数据
	metadata := &client.MetadataEntry{
		ID:           fileObj.ID,
		Key:          objectKey,
		Bucket:       bucket,
		Size:         fileObj.Size,
		ContentType:  contentType,
		MD5Hash:      md5Hash,
		StorageNodes: []string{"stg1", "stg2", "stg3"},
		CreatedAt:    fileObj.CreatedAt,
		UpdatedAt:    time.Now(),
	}

	err = h.clients.Metadata.SaveMetadata(ctx, metadata)
	if err != nil {
		h.logger.Error(c.Request.Context(), "元数据保存失败", err, map[string]any{
			"bucket":    bucket,
			"key":       key,
			"object_id": fileObj.ID,
		})
		// 继续执行，不影响上传结果
	}

	// 提交异步任务
	task := &client.TaskMessage{
		ID:        utils.GenerateID(),
		Type:      "post_upload",
		ObjectKey: objectKey,
		Data: map[string]any{
			"size":         fileObj.Size,
			"content_type": contentType,
		},
		CreatedAt: time.Now(),
	}

	err = h.clients.Task.SubmitTask(ctx, task)
	if err != nil {
		h.logger.Error(c.Request.Context(), "提交异步任务失败", err, map[string]any{
			"task_type": "post-upload",
			"bucket":    bucket,
			"key":       key,
		})
		// 不影响主要流程
	}

	// 返回成功响应
	c.Header("ETag", fmt.Sprintf(`"%s"`, md5Hash))
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"object_id": fileObj.ID,
		"key":       objectKey,
		"bucket":    bucket,
		"size":      fileObj.Size,
		"md5_hash":  md5Hash,
		"message":   "对象上传成功",
	})
}

// GetObject 下载对象
func (h *S3Handler) GetObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")
	objectKey := fmt.Sprintf("%s/%s", bucket, key)

	ctx := c.Request.Context()

	// 获取元数据
	metadata, err := h.clients.Metadata.GetMetadata(ctx, objectKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("对象不存在: %v", err),
		})
		return
	}

	// 读取对象数据
	fileObj, err := h.clients.Storage.ReadObject(ctx, objectKey)
	if err != nil {
		h.logger.Error(c.Request.Context(), "存储读取失败", err, map[string]any{
			"bucket": bucket,
			"key":    key,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("读取对象失败: %v", err),
		})
		return
	}

	// 设置响应头
	c.Header("Content-Type", metadata.ContentType)
	c.Header("Content-Length", strconv.FormatInt(metadata.Size, 10))
	c.Header("ETag", fmt.Sprintf(`"%s"`, metadata.MD5Hash))
	c.Header("Last-Modified", metadata.CreatedAt.Format(http.TimeFormat))

	// 返回文件数据
	c.Data(http.StatusOK, metadata.ContentType, fileObj.Data)
}

// DeleteObject 删除对象
func (h *S3Handler) DeleteObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")
	objectKey := fmt.Sprintf("%s/%s", bucket, key)

	ctx := c.Request.Context()

	// 删除元数据
	err := h.clients.Metadata.DeleteMetadata(ctx, objectKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("对象不存在: %v", err),
		})
		return
	}

	// 提交异步删除任务
	task := &client.TaskMessage{
		ID:        utils.GenerateID(),
		Type:      "delete_object",
		ObjectKey: objectKey,
		Data: map[string]any{
			"bucket": bucket,
			"key":    key,
		},
		CreatedAt: time.Now(),
	}

	err = h.clients.Task.SubmitTask(ctx, task)
	if err != nil {
		h.logger.Error(c.Request.Context(), "提交删除任务失败", err, map[string]any{
			"task_type": "delete",
			"bucket":    bucket,
			"key":       key,
		})
		// 不影响主要流程
	}

	c.JSON(http.StatusNoContent, nil)
}

// HeadObject 获取对象元数据
func (h *S3Handler) HeadObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")
	objectKey := fmt.Sprintf("%s/%s", bucket, key)

	ctx := c.Request.Context()

	// 获取元数据
	metadata, err := h.clients.Metadata.GetMetadata(ctx, objectKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("对象不存在: %v", err),
		})
		return
	}

	// 设置响应头
	c.Header("Content-Type", metadata.ContentType)
	c.Header("Content-Length", strconv.FormatInt(metadata.Size, 10))
	c.Header("ETag", fmt.Sprintf(`"%s"`, metadata.MD5Hash))
	c.Header("Last-Modified", metadata.CreatedAt.Format(http.TimeFormat))

	c.Status(http.StatusOK)
}

// ListObjects 列出对象
func (h *S3Handler) ListObjects(c *gin.Context) {
	bucket := c.Param("bucket")

	// 这里简化实现，实际应该调用元数据服务的列表接口
	c.JSON(http.StatusOK, gin.H{
		"bucket":  bucket,
		"objects": []any{},
		"message": "列表功能待实现",
	})
}

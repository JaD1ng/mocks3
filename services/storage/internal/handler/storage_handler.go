package handler

import (
	"net/http"
	"time"

	"micro-s3/shared/logger"
	"storage/internal/nodes"

	"github.com/gin-gonic/gin"
)

type StorageHandler struct {
	storageManager *nodes.StorageManager
	logger         logger.Logger
}

func NewStorageHandler(storageManager *nodes.StorageManager) *StorageHandler {
	return &StorageHandler{
		storageManager: storageManager,
		logger:         logger.DefaultLogger,
	}
}

func NewStorageHandlerWithLogger(storageManager *nodes.StorageManager, l logger.Logger) *StorageHandler {
	return &StorageHandler{
		storageManager: storageManager,
		logger:         l,
	}
}

// HealthCheck 健康检查
func (h *StorageHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "storage",
		"timestamp": time.Now(),
	})
}

// WriteObject 写入对象
func (h *StorageHandler) WriteObject(c *gin.Context) {
	var obj nodes.FileObject
	if err := c.ShouldBindJSON(&obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 写入所有存储节点
	err := h.storageManager.WriteToAllNodes(&obj)
	if err != nil {
		h.logger.Error(c.Request.Context(), "写入存储节点失败", err, map[string]any{
			"object_key":  obj.Key,
			"object_size": obj.Size,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "写入存储失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "对象写入成功",
		"key":     obj.Key,
		"size":    obj.Size,
		"md5":     obj.MD5Hash,
	})
}

// ReadObject 读取对象
func (h *StorageHandler) ReadObject(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "key 参数不能为空",
		})
		return
	}

	// 优先从 stg1 读取，失败则从第三方服务获取
	obj, err := h.storageManager.ReadFromStg1OrThirdParty(key)
	if err != nil {
		// 如果第三方也失败，尝试从其他节点读取
		obj, err = h.storageManager.ReadFromAnyNode(key)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "对象不存在: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, obj)
}

// DeleteObject 删除对象
func (h *StorageHandler) DeleteObject(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "key 参数不能为空",
		})
		return
	}

	// 从所有节点删除
	err := h.storageManager.DeleteFromAllNodes(key)
	if err != nil {
		h.logger.Error(c.Request.Context(), "删除对象失败", err, map[string]any{
			"object_key": key,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除对象失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "对象删除成功",
		"key":     key,
	})
}

// GetNodes 获取所有存储节点状态
func (h *StorageHandler) GetNodes(c *gin.Context) {
	nodes := h.storageManager.GetNodes()

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// GetNodeStatus 获取指定节点状态
func (h *StorageHandler) GetNodeStatus(c *gin.Context) {
	nodeID := c.Param("nodeId")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "nodeId 参数不能为空",
		})
		return
	}

	status, err := h.storageManager.GetNodeStatus(nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "节点不存在: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetStats 获取存储统计信息
func (h *StorageHandler) GetStats(c *gin.Context) {
	stats := h.storageManager.GetStats()
	c.JSON(http.StatusOK, stats)
}

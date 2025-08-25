package handler

import (
	"net/http"
	"strconv"

	"mocks3/services/queue/internal/service"
	"mocks3/shared/models"
	"mocks3/shared/observability"

	"github.com/gin-gonic/gin"
)

// QueueHandler 队列处理器
type QueueHandler struct {
	service *service.QueueService
	logger  *observability.Logger
}

// NewQueueHandler 创建队列处理器
func NewQueueHandler(service *service.QueueService, logger *observability.Logger) *QueueHandler {
	return &QueueHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *QueueHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// 任务管理
		api.POST("/tasks", h.AddTask)
		api.GET("/tasks/:id", h.GetTask)
		api.GET("/tasks", h.ListTasks)

		// 工作节点管理
		api.POST("/workers/:id/start", h.StartWorker)
		api.POST("/workers/:id/stop", h.StopWorker)

		// 统计信息
		api.GET("/stats", h.GetStats)
	}
}

// AddTaskRequest 添加任务请求
type AddTaskRequest struct {
	Type     string                 `json:"type" binding:"required"`
	Priority int                    `json:"priority"`
	Data     map[string]interface{} `json:"data"`
}

// AddTask 添加任务
func (h *QueueHandler) AddTask(c *gin.Context) {
	var req AddTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// 创建任务
	task := &models.Task{
		Type:     req.Type,
		Priority: req.Priority,
		Data:     req.Data,
	}

	// 生成任务ID
	task.GenerateID()

	// 添加到队列
	if err := h.service.AddTask(c.Request.Context(), task); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to add task", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add task",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"task_id":   task.ID,
		"stream_id": task.StreamID,
		"status":    "pending",
	})
}

// GetTask 获取任务
func (h *QueueHandler) GetTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Task ID is required",
		})
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), taskID)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Task not found", "task_id", taskID)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task not found",
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// ListTasks 列出任务
func (h *QueueHandler) ListTasks(c *gin.Context) {
	status := c.Query("status")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	tasks, err := h.service.ListTasks(c.Request.Context(), status, limit)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list tasks", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list tasks",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// StartWorker 启动工作节点
func (h *QueueHandler) StartWorker(c *gin.Context) {
	workerID := c.Param("id")
	if workerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Worker ID is required",
		})
		return
	}

	if err := h.service.StartWorker(c.Request.Context(), workerID); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to start worker", "worker_id", workerID, "error", err)
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"worker_id": workerID,
		"status":    "started",
	})
}

// StopWorker 停止工作节点
func (h *QueueHandler) StopWorker(c *gin.Context) {
	workerID := c.Param("id")
	if workerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Worker ID is required",
		})
		return
	}

	if err := h.service.StopWorker(c.Request.Context(), workerID); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to stop worker", "worker_id", workerID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"worker_id": workerID,
		"status":    "stopped",
	})
}

// GetStats 获取统计信息
func (h *QueueHandler) GetStats(c *gin.Context) {
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

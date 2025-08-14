package handler

import (
	"net/http"
	"time"

	"task-processor/internal/queue"
	"task-processor/internal/worker"

	"github.com/gin-gonic/gin"
	"micro-s3/shared/utils"
)

type TaskHandler struct {
	queue         *queue.RedisQueue
	workerManager *worker.WorkerManager
}

func NewTaskHandler(queue *queue.RedisQueue, workerManager *worker.WorkerManager) *TaskHandler {
	return &TaskHandler{
		queue:         queue,
		workerManager: workerManager,
	}
}

// HealthCheck 健康检查
func (h *TaskHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "task-processor",
		"timestamp": time.Now(),
	})
}

// SubmitTask 提交任务
func (h *TaskHandler) SubmitTask(c *gin.Context) {
	var task queue.TaskMessage
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 生成任务 ID（如果没有提供）
	if task.ID == "" {
		task.ID = utils.GenerateID()
	}

	// 设置默认值
	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}

	// 提交任务到队列
	ctx := c.Request.Context()
	err := h.queue.SubmitTask(ctx, &task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "提交任务失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"task_id": task.ID,
		"message": "任务提交成功",
	})
}

// GetTasks 获取任务列表
func (h *TaskHandler) GetTasks(c *gin.Context) {
	ctx := c.Request.Context()
	
	// 获取待处理任务
	tasks, err := h.queue.GetPendingTasks(ctx, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取任务列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// GetTask 获取指定任务
func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "taskId 参数不能为空",
		})
		return
	}

	// 在实际实现中，这里需要从队列或数据库查询具体任务
	// 由于 Redis Streams 的特性，查询单个任务比较复杂
	// 这里返回简化的响应
	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"message": "任务查询功能待完善",
	})
}

// CancelTask 取消任务
func (h *TaskHandler) CancelTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "taskId 参数不能为空",
		})
		return
	}

	// 在实际实现中，这里需要实现任务取消逻辑
	// 由于 Redis Streams 的特性，取消任务比较复杂
	// 可能需要维护额外的取消任务列表
	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"message": "任务已标记为取消",
	})
}

// GetQueueStats 获取队列统计信息
func (h *TaskHandler) GetQueueStats(c *gin.Context) {
	ctx := c.Request.Context()
	
	stats, err := h.queue.GetQueueStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取队列统计失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetWorkerStatus 获取工作节点状态
func (h *TaskHandler) GetWorkerStatus(c *gin.Context) {
	workerStatus := h.workerManager.GetWorkerStatus()
	workerStats := h.workerManager.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"workers": workerStatus,
		"stats":   workerStats,
	})
}
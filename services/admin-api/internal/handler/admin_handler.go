package handler

import (
	"mocks3/services/admin-api/internal/client"
	"context"
	"mocks3/shared/logger"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	clients *client.Clients
	logger  logger.Logger
}

func NewAdminHandler(clients *client.Clients) *AdminHandler {
	return &AdminHandler{
		clients: clients,
		logger:  logger.DefaultLogger,
	}
}

func NewAdminHandlerWithLogger(clients *client.Clients, logger logger.Logger) *AdminHandler {
	return &AdminHandler{
		clients: clients,
		logger:  logger,
	}
}

// HealthCheck 健康检查
func (h *AdminHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "admin-api",
		"timestamp": time.Now(),
	})
}

// GetOverview 获取系统概览
func (h *AdminHandler) GetOverview(c *gin.Context) {
	ctx := c.Request.Context()

	overview := gin.H{
		"timestamp": time.Now(),
	}

	// 获取各服务状态
	services := []struct {
		name   string
		client *client.ServiceClient
	}{
		{"metadata", h.clients.Metadata},
		{"storage", h.clients.Storage},
		{"task", h.clients.Task},
		{"chaos", h.clients.Chaos},
	}

	serviceStatus := make(map[string]any)
	for _, svc := range services {
		health, err := svc.client.GetHealth(ctx)
		if err != nil {
			serviceStatus[svc.name] = gin.H{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			serviceStatus[svc.name] = health
		}
	}
	overview["services"] = serviceStatus

	// 获取存储统计
	var storageStats map[string]any
	err := h.clients.Storage.Get(ctx, "/stats", &storageStats)
	if err != nil {
		h.logger.Error(ctx, "Failed to get storage stats", err, map[string]any{
			"operation": "get_storage_stats",
		})
		storageStats = gin.H{"error": err.Error()}
	}
	overview["storage"] = storageStats

	// 获取元数据统计
	var metadataStats map[string]any
	err = h.clients.Metadata.Get(ctx, "/stats", &metadataStats)
	if err != nil {
		h.logger.Error(ctx, "Failed to get metadata stats", err, map[string]any{
			"operation": "get_metadata_stats",
		})
		metadataStats = gin.H{"error": err.Error()}
	}
	overview["metadata"] = metadataStats

	// 获取任务统计
	var taskStats map[string]any
	err = h.clients.Task.Get(ctx, "/queue/stats", &taskStats)
	if err != nil {
		h.logger.Error(ctx, "Failed to get task stats", err, map[string]any{
			"operation": "get_task_stats",
		})
		taskStats = gin.H{"error": err.Error()}
	}
	overview["tasks"] = taskStats

	c.JSON(http.StatusOK, overview)
}

// GetServices 获取所有服务状态
func (h *AdminHandler) GetServices(c *gin.Context) {
	ctx := c.Request.Context()

	services := []struct {
		name   string
		client *client.ServiceClient
	}{
		{"metadata", h.clients.Metadata},
		{"storage", h.clients.Storage},
		{"task-processor", h.clients.Task},
		{"chaos-engineering", h.clients.Chaos},
	}

	result := make([]gin.H, 0, len(services))
	for _, svc := range services {
		health, err := svc.client.GetHealth(ctx)
		serviceInfo := gin.H{
			"name": svc.name,
		}

		if err != nil {
			serviceInfo["status"] = "error"
			serviceInfo["error"] = err.Error()
		} else {
			serviceInfo["status"] = "healthy"
			serviceInfo["health"] = health
		}

		result = append(result, serviceInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"services": result,
		"count":    len(result),
	})
}

// GetServiceStatus 获取指定服务状态
func (h *AdminHandler) GetServiceStatus(c *gin.Context) {
	serviceName := c.Param("serviceName")
	ctx := c.Request.Context()

	var client *client.ServiceClient
	switch serviceName {
	case "metadata":
		client = h.clients.Metadata
	case "storage":
		client = h.clients.Storage
	case "task-processor":
		client = h.clients.Task
	case "chaos-engineering":
		client = h.clients.Chaos
	default:
		c.JSON(http.StatusNotFound, gin.H{
			"error": "服务不存在: " + serviceName,
		})
		return
	}

	health, err := client.GetHealth(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"service": serviceName,
			"status":  "error",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service": serviceName,
		"status":  "healthy",
		"health":  health,
	})
}

// ListObjects 列出对象
func (h *AdminHandler) ListObjects(c *gin.Context) {
	ctx := c.Request.Context()

	limit := c.DefaultQuery("limit", "20")
	offset := c.DefaultQuery("offset", "0")

	var objects map[string]any
	err := h.clients.Metadata.Get(ctx, "/metadata?limit="+limit+"&offset="+offset, &objects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取对象列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, objects)
}

// GetObject 获取指定对象
func (h *AdminHandler) GetObject(c *gin.Context) {
	key := c.Param("key")
	ctx := c.Request.Context()

	var object map[string]any
	err := h.clients.Metadata.Get(ctx, "/metadata/"+key, &object)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "对象不存在: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, object)
}

// CreateObject 创建对象
func (h *AdminHandler) CreateObject(c *gin.Context) {
	var request map[string]any
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	var result map[string]any
	err := h.clients.Metadata.Post(ctx, "/metadata", request, &result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建对象失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// DeleteObject 删除对象
func (h *AdminHandler) DeleteObject(c *gin.Context) {
	key := c.Param("key")
	ctx := c.Request.Context()

	err := h.clients.Metadata.Delete(ctx, "/metadata/"+key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除对象失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "对象删除成功",
	})
}

// GetStorageStats 获取存储统计
func (h *AdminHandler) GetStorageStats(c *gin.Context) {
	ctx := c.Request.Context()

	var stats map[string]any
	err := h.clients.Storage.Get(ctx, "/stats", &stats)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取存储统计失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetStorageNodes 获取存储节点
func (h *AdminHandler) GetStorageNodes(c *gin.Context) {
	ctx := c.Request.Context()

	var nodes map[string]any
	err := h.clients.Storage.Get(ctx, "/nodes", &nodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取存储节点失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, nodes)
}

// GetStorageNode 获取指定存储节点
func (h *AdminHandler) GetStorageNode(c *gin.Context) {
	nodeID := c.Param("nodeId")
	ctx := c.Request.Context()

	var node map[string]any
	err := h.clients.Storage.Get(ctx, "/nodes/"+nodeID+"/status", &node)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "存储节点不存在: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, node)
}

// GetTasks 获取任务列表
func (h *AdminHandler) GetTasks(c *gin.Context) {
	ctx := c.Request.Context()

	var tasks map[string]any
	err := h.clients.Task.Get(ctx, "/tasks", &tasks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取任务列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// CreateTask 创建任务
func (h *AdminHandler) CreateTask(c *gin.Context) {
	var request map[string]any
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	var result map[string]any
	err := h.clients.Task.Post(ctx, "/tasks", request, &result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建任务失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetTaskStats 获取任务统计
func (h *AdminHandler) GetTaskStats(c *gin.Context) {
	ctx := c.Request.Context()

	var stats map[string]any
	err := h.clients.Task.Get(ctx, "/queue/stats", &stats)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取任务统计失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetWorkerStatus 获取工作节点状态
func (h *AdminHandler) GetWorkerStatus(c *gin.Context) {
	ctx := c.Request.Context()

	var workers map[string]any
	err := h.clients.Task.Get(ctx, "/workers/status", &workers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取工作节点状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, workers)
}

// 混沌工程相关处理器

// GetChaosRules 获取混沌规则
func (h *AdminHandler) GetChaosRules(c *gin.Context) {
	ctx := c.Request.Context()

	var rules map[string]any
	err := h.clients.Chaos.Get(ctx, "/rules", &rules)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取混沌规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rules)
}

// CreateChaosRule 创建混沌规则
func (h *AdminHandler) CreateChaosRule(c *gin.Context) {
	var request map[string]any
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	var result map[string]any
	err := h.clients.Chaos.Post(ctx, "/rules", request, &result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建混沌规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// UpdateChaosRule 更新混沌规则
func (h *AdminHandler) UpdateChaosRule(c *gin.Context) {
	ruleID := c.Param("ruleId")

	var request map[string]any
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	var result map[string]any
	err := h.clients.Chaos.Put(ctx, "/rules/"+ruleID, request, &result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新混沌规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteChaosRule 删除混沌规则
func (h *AdminHandler) DeleteChaosRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	ctx := c.Request.Context()

	err := h.clients.Chaos.Delete(ctx, "/rules/"+ruleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除混沌规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "混沌规则删除成功",
	})
}

// EnableChaosRule 启用混沌规则
func (h *AdminHandler) EnableChaosRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	ctx := c.Request.Context()

	var result map[string]any
	err := h.clients.Chaos.Post(ctx, "/rules/"+ruleID+"/enable", nil, &result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "启用混沌规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DisableChaosRule 禁用混沌规则
func (h *AdminHandler) DisableChaosRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	ctx := c.Request.Context()

	var result map[string]any
	err := h.clients.Chaos.Post(ctx, "/rules/"+ruleID+"/disable", nil, &result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "禁用混沌规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetChaosLogs 获取混沌日志
func (h *AdminHandler) GetChaosLogs(c *gin.Context) {
	ctx := c.Request.Context()

	limit := c.DefaultQuery("limit", "50")

	var logs map[string]any
	err := h.clients.Chaos.Get(ctx, "/logs?limit="+limit, &logs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取混沌日志失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetMetrics 获取监控指标
func (h *AdminHandler) GetMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	// 聚合各服务的指标数据
	metrics := gin.H{
		"timestamp": time.Now(),
	}

	// 存储指标
	var storageStats map[string]any
	err := h.clients.Storage.Get(ctx, "/stats", &storageStats)
	if err == nil {
		metrics["storage"] = storageStats
	}

	// 元数据指标
	var metadataStats map[string]any
	err = h.clients.Metadata.Get(ctx, "/stats", &metadataStats)
	if err == nil {
		metrics["metadata"] = metadataStats
	}

	// 任务指标
	var taskStats map[string]any
	err = h.clients.Task.Get(ctx, "/queue/stats", &taskStats)
	if err == nil {
		metrics["tasks"] = taskStats
	}

	// 混沌工程指标
	var chaosStats map[string]any
	err = h.clients.Chaos.Get(ctx, "/stats", &chaosStats)
	if err == nil {
		metrics["chaos"] = chaosStats
	}

	c.JSON(http.StatusOK, metrics)
}

// GetLogs 获取系统日志
func (h *AdminHandler) GetLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	service := c.Query("service")

	// 这里简化实现，实际应该从日志聚合系统获取
	logs := []gin.H{
		{
			"timestamp": time.Now().Add(-time.Minute),
			"level":     "INFO",
			"service":   "metadata",
			"message":   "Successfully processed metadata query",
		},
		{
			"timestamp": time.Now().Add(-2 * time.Minute),
			"level":     "ERROR",
			"service":   "storage",
			"message":   "Failed to write to storage node stg2",
		},
		{
			"timestamp": time.Now().Add(-3 * time.Minute),
			"level":     "WARN",
			"service":   "task-processor",
			"message":   "Task queue is approaching capacity limit",
		},
	}

	// 根据服务过滤
	if service != "" {
		filteredLogs := []gin.H{}
		for _, log := range logs {
			if log["service"] == service {
				filteredLogs = append(filteredLogs, log)
			}
		}
		logs = filteredLogs
	}

	// 限制返回数量
	if limit > 0 && limit < len(logs) {
		logs = logs[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"count": len(logs),
	})
}

// GetDashboard 获取仪表板数据
func (h *AdminHandler) GetDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	dashboard := gin.H{
		"timestamp": time.Now(),
	}

	// 系统概览
	overview, _ := h.getSystemOverview(ctx)
	dashboard["overview"] = overview

	// 服务健康状态
	serviceHealth, _ := h.getServiceHealth(ctx)
	dashboard["service_health"] = serviceHealth

	// 关键指标
	metrics, _ := h.getKeyMetrics(ctx)
	dashboard["metrics"] = metrics

	// 最近的告警或错误
	alerts := h.getRecentAlerts()
	dashboard["alerts"] = alerts

	c.JSON(http.StatusOK, dashboard)
}

// 辅助方法

func (h *AdminHandler) getSystemOverview(ctx context.Context) (gin.H, error) {
	overview := gin.H{}

	// 对象总数
	var metadataStats map[string]any
	err := h.clients.Metadata.Get(ctx, "/stats", &metadataStats)
	if err == nil {
		if totalObjects, ok := metadataStats["total_objects"]; ok {
			overview["total_objects"] = totalObjects
		}
		if totalSize, ok := metadataStats["total_size"]; ok {
			overview["total_size"] = totalSize
		}
	}

	// 存储使用情况
	var storageStats map[string]any
	err = h.clients.Storage.Get(ctx, "/stats", &storageStats)
	if err == nil {
		if usagePercent, ok := storageStats["usage_percent"]; ok {
			overview["storage_usage"] = usagePercent
		}
		if healthyNodes, ok := storageStats["healthy_nodes"]; ok {
			overview["healthy_nodes"] = healthyNodes
		}
	}

	return overview, nil
}

func (h *AdminHandler) getServiceHealth(ctx context.Context) ([]gin.H, error) {
	services := []struct {
		name   string
		client *client.ServiceClient
	}{
		{"metadata", h.clients.Metadata},
		{"storage", h.clients.Storage},
		{"task-processor", h.clients.Task},
		{"chaos-engineering", h.clients.Chaos},
	}

	var serviceHealth []gin.H
	for _, svc := range services {
		health, err := svc.client.GetHealth(ctx)
		status := gin.H{
			"name": svc.name,
		}

		if err != nil {
			status["status"] = "error"
			status["error"] = err.Error()
		} else {
			status["status"] = "healthy"
			status["timestamp"] = health["timestamp"]
		}

		serviceHealth = append(serviceHealth, status)
	}

	return serviceHealth, nil
}

func (h *AdminHandler) getKeyMetrics(ctx context.Context) (gin.H, error) {
	metrics := gin.H{}

	// 任务处理速率
	var taskStats map[string]any
	err := h.clients.Task.Get(ctx, "/queue/stats", &taskStats)
	if err == nil {
		metrics["task_queue_length"] = taskStats["pending_tasks"]
		metrics["task_consumers"] = taskStats["consumer_count"]
	}

	// 存储节点状态
	var storageStats map[string]any
	err = h.clients.Storage.Get(ctx, "/stats", &storageStats)
	if err == nil {
		metrics["storage_nodes_total"] = storageStats["total_nodes"]
		metrics["storage_nodes_healthy"] = storageStats["healthy_nodes"]
	}

	return metrics, nil
}

func (h *AdminHandler) getRecentAlerts() []gin.H {
	// 这里返回模拟的告警数据
	return []gin.H{
		{
			"timestamp": time.Now().Add(-5 * time.Minute),
			"level":     "warning",
			"service":   "storage",
			"message":   "Storage node stg2 usage > 85%",
		},
		{
			"timestamp": time.Now().Add(-10 * time.Minute),
			"level":     "error",
			"service":   "metadata",
			"message":   "Database connection timeout",
		},
	}
}

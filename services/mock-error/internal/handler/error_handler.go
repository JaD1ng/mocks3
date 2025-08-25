package handler

import (
	"net/http"
	"strconv"
	"time"

	"mocks3/services/mock-error/internal/service"
	"mocks3/shared/models"
	"mocks3/shared/observability"

	"github.com/gin-gonic/gin"
)

// ErrorHandler 错误注入处理器
type ErrorHandler struct {
	service *service.ErrorInjectorService
	logger  *observability.Logger
}

// NewErrorHandler 创建错误注入处理器
func NewErrorHandler(service *service.ErrorInjectorService, logger *observability.Logger) *ErrorHandler {
	return &ErrorHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *ErrorHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// 错误规则管理
		api.POST("/rules", h.AddErrorRule)
		api.GET("/rules/:id", h.GetErrorRule)
		api.PUT("/rules/:id", h.UpdateErrorRule)
		api.DELETE("/rules/:id", h.RemoveErrorRule)
		api.GET("/rules", h.ListErrorRules)

		// 错误注入控制
		api.POST("/inject/:service/:operation", h.CheckErrorInjection)

		// 统计信息
		api.GET("/stats", h.GetErrorStats)
		api.POST("/stats/reset", h.ResetErrorStats)
		api.GET("/events", h.GetErrorEvents)

		// 规则控制
		api.POST("/rules/:id/enable", h.EnableRule)
		api.POST("/rules/:id/disable", h.DisableRule)
	}
}

// AddErrorRuleRequest 添加错误规则请求
type AddErrorRuleRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Description string                  `json:"description"`
	Service     string                  `json:"service"`
	Operation   string                  `json:"operation"`
	Conditions  []models.ErrorCondition `json:"conditions"`
	Action      models.ErrorAction      `json:"action" binding:"required"`
	Enabled     bool                    `json:"enabled"`
	Priority    int                     `json:"priority"`
	MaxTriggers int                     `json:"max_triggers"`
	Schedule    *models.ErrorSchedule   `json:"schedule,omitempty"`
	Metadata    map[string]string       `json:"metadata,omitempty"`
}

// AddErrorRule 添加错误规则
func (h *ErrorHandler) AddErrorRule(c *gin.Context) {
	var req AddErrorRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	rule := &models.ErrorRule{
		Name:        req.Name,
		Description: req.Description,
		Service:     req.Service,
		Operation:   req.Operation,
		Conditions:  req.Conditions,
		Action:      req.Action,
		Enabled:     req.Enabled,
		Priority:    req.Priority,
		MaxTriggers: req.MaxTriggers,
		Schedule:    req.Schedule,
		Metadata:    req.Metadata,
		Triggered:   0,
	}

	if err := h.service.AddErrorRule(c.Request.Context(), rule); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to add error rule", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add error rule",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"rule_id": rule.ID,
		"message": "Error rule added successfully",
	})
}

// GetErrorRule 获取错误规则
func (h *ErrorHandler) GetErrorRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Rule ID is required",
		})
		return
	}

	rule, err := h.service.GetErrorRule(c.Request.Context(), ruleID)
	if err != nil {
		h.logger.WarnContext(c.Request.Context(), "Rule not found", "rule_id", ruleID)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Rule not found",
		})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateErrorRule 更新错误规则
func (h *ErrorHandler) UpdateErrorRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Rule ID is required",
		})
		return
	}

	var req AddErrorRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WarnContext(c.Request.Context(), "Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	rule := &models.ErrorRule{
		ID:          ruleID,
		Name:        req.Name,
		Description: req.Description,
		Service:     req.Service,
		Operation:   req.Operation,
		Conditions:  req.Conditions,
		Action:      req.Action,
		Enabled:     req.Enabled,
		Priority:    req.Priority,
		MaxTriggers: req.MaxTriggers,
		Schedule:    req.Schedule,
		Metadata:    req.Metadata,
	}

	if err := h.service.UpdateErrorRule(c.Request.Context(), rule); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to update error rule", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update error rule",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Error rule updated successfully",
	})
}

// RemoveErrorRule 删除错误规则
func (h *ErrorHandler) RemoveErrorRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Rule ID is required",
		})
		return
	}

	if err := h.service.RemoveErrorRule(c.Request.Context(), ruleID); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to remove error rule", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove error rule",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Error rule removed successfully",
	})
}

// ListErrorRules 列出错误规则
func (h *ErrorHandler) ListErrorRules(c *gin.Context) {
	rules, err := h.service.ListErrorRules(c.Request.Context())
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list error rules", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list error rules",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
		"count": len(rules),
	})
}

// CheckErrorInjectionRequest 检查错误注入请求
type CheckErrorInjectionRequest struct {
	Metadata map[string]string `json:"metadata"`
}

// CheckErrorInjection 检查错误注入
func (h *ErrorHandler) CheckErrorInjection(c *gin.Context) {
	service := c.Param("service")
	operation := c.Param("operation")

	if service == "" || operation == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Service and operation are required",
		})
		return
	}

	var req CheckErrorInjectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有body，使用空metadata
		req.Metadata = make(map[string]string)
	}

	action, shouldInject := h.service.ShouldInjectError(c.Request.Context(), service, operation)

	response := gin.H{
		"should_inject": shouldInject,
		"service":       service,
		"operation":     operation,
	}

	if shouldInject && action != nil {
		response["action"] = action
	}

	c.JSON(http.StatusOK, response)
}

// GetErrorStats 获取错误统计
func (h *ErrorHandler) GetErrorStats(c *gin.Context) {
	stats, err := h.service.GetErrorStats(c.Request.Context())
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to get error stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get error statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ResetErrorStats 重置错误统计
func (h *ErrorHandler) ResetErrorStats(c *gin.Context) {
	if err := h.service.ResetErrorStats(c.Request.Context()); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to reset error stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reset error statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Error statistics reset successfully",
	})
}

// GetErrorEvents 获取错误事件
func (h *ErrorHandler) GetErrorEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	// 这里需要在statsRepo中实现GetEvents方法
	// events, err := h.statsRepo.GetEvents(c.Request.Context(), limit)
	// 目前返回空列表，限制数量为 limit
	events := make([]*models.ErrorEvent, 0, limit)

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}

// EnableRule 启用规则
func (h *ErrorHandler) EnableRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Rule ID is required",
		})
		return
	}

	// 获取规则
	rule, err := h.service.GetErrorRule(c.Request.Context(), ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Rule not found",
		})
		return
	}

	// 启用规则
	rule.Enabled = true
	rule.UpdatedAt = time.Now()

	if err := h.service.UpdateErrorRule(c.Request.Context(), rule); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to enable rule", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to enable rule",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Rule enabled successfully",
	})
}

// DisableRule 禁用规则
func (h *ErrorHandler) DisableRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Rule ID is required",
		})
		return
	}

	// 获取规则
	rule, err := h.service.GetErrorRule(c.Request.Context(), ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Rule not found",
		})
		return
	}

	// 禁用规则
	rule.Enabled = false
	rule.UpdatedAt = time.Now()

	if err := h.service.UpdateErrorRule(c.Request.Context(), rule); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to disable rule", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to disable rule",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Rule disabled successfully",
	})
}

package handler

import (
	"net/http"
	"strconv"
	"time"

	"chaos-engineering/internal/injector"
	"chaos-engineering/internal/rules"

	"github.com/gin-gonic/gin"
)

type ChaosHandler struct {
	ruleManager   *rules.RuleManager
	chaosInjector *injector.ChaosInjector
}

func NewChaosHandler(ruleManager *rules.RuleManager, chaosInjector *injector.ChaosInjector) *ChaosHandler {
	return &ChaosHandler{
		ruleManager:   ruleManager,
		chaosInjector: chaosInjector,
	}
}

// HealthCheck 健康检查
func (h *ChaosHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "chaos-engineering",
		"timestamp": time.Now(),
	})
}

// GetRules 获取所有规则
func (h *ChaosHandler) GetRules(c *gin.Context) {
	rules := h.ruleManager.ListRules()

	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
		"count": len(rules),
	})
}

// CreateRule 创建新规则
func (h *ChaosHandler) CreateRule(c *gin.Context) {
	var rule rules.ChaosRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	err := h.ruleManager.AddRule(&rule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"rule_id": rule.ID,
		"message": "规则创建成功",
		"rule":    rule,
	})
}

// GetRule 获取指定规则
func (h *ChaosHandler) GetRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ruleId 参数不能为空",
		})
		return
	}

	rule, err := h.ruleManager.GetRule(ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "规则不存在: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateRule 更新规则
func (h *ChaosHandler) UpdateRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ruleId 参数不能为空",
		})
		return
	}

	var updates rules.ChaosRule
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	err := h.ruleManager.UpdateRule(ruleID, &updates)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "更新规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "规则更新成功",
	})
}

// DeleteRule 删除规则
func (h *ChaosHandler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ruleId 参数不能为空",
		})
		return
	}

	err := h.ruleManager.DeleteRule(ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "删除规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "规则删除成功",
	})
}

// EnableRule 启用规则
func (h *ChaosHandler) EnableRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ruleId 参数不能为空",
		})
		return
	}

	err := h.ruleManager.EnableRule(ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "启用规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "规则已启用",
	})
}

// DisableRule 禁用规则
func (h *ChaosHandler) DisableRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ruleId 参数不能为空",
		})
		return
	}

	err := h.ruleManager.DisableRule(ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "禁用规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "规则已禁用",
	})
}

// StartChaos 启动混沌注入
func (h *ChaosHandler) StartChaos(c *gin.Context) {
	err := h.chaosInjector.Start()
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "启动混沌注入失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "混沌注入已启动",
	})
}

// StopChaos 停止混沌注入
func (h *ChaosHandler) StopChaos(c *gin.Context) {
	h.chaosInjector.Stop()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "混沌注入已停止",
	})
}

// GetChaosStatus 获取混沌注入状态
func (h *ChaosHandler) GetChaosStatus(c *gin.Context) {
	status := h.chaosInjector.GetStatus()

	c.JSON(http.StatusOK, status)
}

// GetStats 获取统计信息
func (h *ChaosHandler) GetStats(c *gin.Context) {
	ruleStats := h.ruleManager.GetStats()
	injectorStatus := h.chaosInjector.GetStatus()

	c.JSON(http.StatusOK, gin.H{
		"rule_stats":     ruleStats,
		"injector_stats": injectorStatus,
		"timestamp":      time.Now(),
	})
}

// GetLogs 获取执行日志
func (h *ChaosHandler) GetLogs(c *gin.Context) {
	limit := 50 // 默认返回最近50条
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	logs := h.chaosInjector.GetExecutionLog(limit)

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"count": len(logs),
		"limit": limit,
	})
}

// ExecuteRule 强制执行指定规则
func (h *ChaosHandler) ExecuteRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ruleId 参数不能为空",
		})
		return
	}

	err := h.chaosInjector.ForceExecuteRule(ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "执行规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "规则执行已启动",
	})
}

// CancelRule 取消正在执行的规则
func (h *ChaosHandler) CancelRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ruleId 参数不能为空",
		})
		return
	}

	err := h.chaosInjector.CancelRule(ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "取消规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "规则执行已取消",
	})
}

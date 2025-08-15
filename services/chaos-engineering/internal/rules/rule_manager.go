package rules

import (
	"encoding/json"
	"fmt"
	"github.com/mocks3/shared/logger"
	"github.com/mocks3/shared/utils"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ChaosRule 混沌工程规则
type ChaosRule struct {
	ID           string         `json:"id" yaml:"id"`
	Name         string         `json:"name" yaml:"name"`
	Description  string         `json:"description,omitempty" yaml:"description,omitempty"`
	Service      string         `json:"service" yaml:"service"`
	Endpoint     string         `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	FailureType  string         `json:"failure_type" yaml:"failure_type"`
	FailureRate  float64        `json:"failure_rate" yaml:"failure_rate"`
	Duration     string         `json:"duration,omitempty" yaml:"duration,omitempty"`
	Schedule     string         `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	Enabled      bool           `json:"enabled" yaml:"enabled"`
	Config       map[string]any `json:"config,omitempty" yaml:"config,omitempty"`
	CreatedAt    time.Time      `json:"created_at" yaml:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" yaml:"updated_at"`
	LastExecuted time.Time      `json:"last_executed,omitempty" yaml:"last_executed,omitempty"`
	ExecuteCount int64          `json:"execute_count" yaml:"execute_count"`
}

// RuleManager 规则管理器
type RuleManager struct {
	rules map[string]*ChaosRule
	mu    sync.RWMutex
}

// NewRuleManager 创建规则管理器
func NewRuleManager() *RuleManager {
	return &RuleManager{
		rules: make(map[string]*ChaosRule),
	}
}

// AddRule 添加规则
func (rm *RuleManager) AddRule(rule *ChaosRule) error {
	if rule.ID == "" {
		rule.ID = utils.GenerateID()
	}

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.rules[rule.ID] = rule
	logger.Infof("Added chaos rule: %s (%s)", rule.Name, rule.ID)

	return nil
}

// GetRule 获取规则
func (rm *RuleManager) GetRule(id string) (*ChaosRule, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rule, exists := rm.rules[id]
	if !exists {
		return nil, fmt.Errorf("rule not found: %s", id)
	}

	return rule, nil
}

// UpdateRule 更新规则
func (rm *RuleManager) UpdateRule(id string, updates *ChaosRule) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, exists := rm.rules[id]
	if !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	// 更新字段
	if updates.Name != "" {
		rule.Name = updates.Name
	}
	if updates.Description != "" {
		rule.Description = updates.Description
	}
	if updates.Service != "" {
		rule.Service = updates.Service
	}
	if updates.Endpoint != "" {
		rule.Endpoint = updates.Endpoint
	}
	if updates.FailureType != "" {
		rule.FailureType = updates.FailureType
	}
	if updates.FailureRate > 0 {
		rule.FailureRate = updates.FailureRate
	}
	if updates.Duration != "" {
		rule.Duration = updates.Duration
	}
	if updates.Schedule != "" {
		rule.Schedule = updates.Schedule
	}
	if updates.Config != nil {
		rule.Config = updates.Config
	}

	rule.UpdatedAt = time.Now()
	logger.Infof("Updated chaos rule: %s", rule.Name)

	return nil
}

// DeleteRule 删除规则
func (rm *RuleManager) DeleteRule(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, exists := rm.rules[id]
	if !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	delete(rm.rules, id)
	logger.Infof("Deleted chaos rule: %s", rule.Name)

	return nil
}

// ListRules 列出所有规则
func (rm *RuleManager) ListRules() []*ChaosRule {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rules := make([]*ChaosRule, 0, len(rm.rules))
	for _, rule := range rm.rules {
		rules = append(rules, rule)
	}

	return rules
}

// EnableRule 启用规则
func (rm *RuleManager) EnableRule(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, exists := rm.rules[id]
	if !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	rule.Enabled = true
	rule.UpdatedAt = time.Now()
	logger.Infof("Enabled chaos rule: %s", rule.Name)

	return nil
}

// DisableRule 禁用规则
func (rm *RuleManager) DisableRule(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, exists := rm.rules[id]
	if !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	rule.Enabled = false
	rule.UpdatedAt = time.Now()
	logger.Infof("Disabled chaos rule: %s", rule.Name)

	return nil
}

// GetActiveRules 获取活跃的规则
func (rm *RuleManager) GetActiveRules() []*ChaosRule {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var activeRules []*ChaosRule
	for _, rule := range rm.rules {
		if rule.Enabled {
			activeRules = append(activeRules, rule)
		}
	}

	return activeRules
}

// MarkRuleExecuted 标记规则已执行
func (rm *RuleManager) MarkRuleExecuted(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, exists := rm.rules[id]
	if !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	rule.LastExecuted = time.Now()
	rule.ExecuteCount++

	return nil
}

// LoadDefaultRules 加载默认规则
// 返回: 错误信息，成功时为nil
func (rm *RuleManager) LoadDefaultRules() error {
	// 获取所有默认规则配置
	defaultRules := rm.getDefaultRuleConfigs()

	// 批量添加规则
	for _, rule := range defaultRules {
		err := rm.AddRule(rule)
		if err != nil {
			return fmt.Errorf("failed to add default rule %s: %w", rule.Name, err)
		}
	}

	logger.Infof("Loaded %d default chaos rules", len(defaultRules))
	return nil
}

// getDefaultRuleConfigs 获取默认规则配置列表（私有方法）
// 返回: 默认规则数组
func (rm *RuleManager) getDefaultRuleConfigs() []*ChaosRule {
	return []*ChaosRule{
		rm.createNetworkTimeoutRule(),
		rm.createMemoryLeakRule(),
		rm.createCPUSpikeRule(),
		rm.createDatabaseErrorRule(),
		rm.createDiskFullRule(),
		rm.createSlowResponseRule(),
	}
}

// createNetworkTimeoutRule 创建网络超时规则
func (rm *RuleManager) createNetworkTimeoutRule() *ChaosRule {
	return &ChaosRule{
		Name:        "网络超时故障",
		Description: "模拟网络请求超时",
		Service:     "*",
		FailureType: "network_timeout",
		FailureRate: 0.05, // 5% 概率
		Duration:    "30s",
		Enabled:     false,
		Config: map[string]any{
			"timeout_ms": 5000,
			"delay_ms":   1000,
		},
	}
}

// createMemoryLeakRule 创建内存泄漏规则
func (rm *RuleManager) createMemoryLeakRule() *ChaosRule {
	return &ChaosRule{
		Name:        "内存泄漏故障",
		Description: "模拟内存逐渐泄漏",
		Service:     "storage",
		FailureType: "memory_leak",
		FailureRate: 0.1, // 10% 概率
		Duration:    "5m",
		Enabled:     false,
		Config: map[string]any{
			"start_memory_mb": 100,
			"increment_mb":    10,
			"interval_sec":    30,
			"max_memory_mb":   500,
		},
	}
}

// createCPUSpikeRule 创建CPU高负载规则
func (rm *RuleManager) createCPUSpikeRule() *ChaosRule {
	return &ChaosRule{
		Name:        "CPU 高负载",
		Description: "模拟 CPU 使用率飙升",
		Service:     "metadata",
		FailureType: "cpu_spike",
		FailureRate: 0.08, // 8% 概率
		Duration:    "2m",
		Enabled:     false,
		Config: map[string]any{
			"cpu_percent": 90,
			"threads":     4,
		},
	}
}

// createDatabaseErrorRule 创建数据库错误规则
func (rm *RuleManager) createDatabaseErrorRule() *ChaosRule {
	return &ChaosRule{
		Name:        "数据库连接故障",
		Description: "模拟数据库连接异常",
		Service:     "metadata",
		FailureType: "database_error",
		FailureRate: 0.03, // 3% 概率
		Duration:    "1m",
		Enabled:     false,
		Config: map[string]any{
			"error_types": []string{"connection_timeout", "deadlock", "connection_refused"},
		},
	}
}

// createDiskFullRule 创建磁盘空间不足规则
func (rm *RuleManager) createDiskFullRule() *ChaosRule {
	return &ChaosRule{
		Name:        "磁盘空间不足",
		Description: "模拟磁盘空间耗尽",
		Service:     "storage",
		FailureType: "disk_full",
		FailureRate: 0.02, // 2% 概率
		Duration:    "3m",
		Enabled:     false,
		Config: map[string]any{
			"free_space_mb": 10,
			"fill_speed":    "fast",
		},
	}
}

// createSlowResponseRule 创建响应延迟规则
func (rm *RuleManager) createSlowResponseRule() *ChaosRule {
	return &ChaosRule{
		Name:        "服务响应延迟",
		Description: "模拟服务响应变慢",
		Service:     "s3-api",
		FailureType: "slow_response",
		FailureRate: 0.15, // 15% 概率
		Duration:    "2m",
		Enabled:     false,
		Config: map[string]any{
			"min_delay_ms": 500,
			"max_delay_ms": 3000,
		},
	}
}

// SaveRulesToFile 保存规则到文件
func (rm *RuleManager) SaveRulesToFile(filePath string) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rules := make([]*ChaosRule, 0, len(rm.rules))
	for _, rule := range rm.rules {
		rules = append(rules, rule)
	}

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// LoadRulesFromFile 从文件加载规则
func (rm *RuleManager) LoadRulesFromFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // 文件不存在，跳过
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var rules []*ChaosRule
	err = json.Unmarshal(data, &rules)
	if err != nil {
		return err
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	for _, rule := range rules {
		rm.rules[rule.ID] = rule
	}

	logger.Infof("Loaded %d chaos rules from file: %s", len(rules), filePath)
	return nil
}

// LoadRulesFromDir 从目录加载规则文件
func (rm *RuleManager) LoadRulesFromDir(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil // 目录不存在，跳过
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (filepath.Ext(path) == ".json" || filepath.Ext(path) == ".yaml") {
			return rm.LoadRulesFromFile(path)
		}

		return nil
	})

	return err
}

// GetStats 获取规则统计信息
func (rm *RuleManager) GetStats() map[string]any {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var totalRules int = len(rm.rules)
	var enabledRules int = 0
	var totalExecutions int64 = 0

	serviceStats := make(map[string]int)
	typeStats := make(map[string]int)

	for _, rule := range rm.rules {
		if rule.Enabled {
			enabledRules++
		}
		totalExecutions += rule.ExecuteCount

		serviceStats[rule.Service]++
		typeStats[rule.FailureType]++
	}

	return map[string]any{
		"total_rules":      totalRules,
		"enabled_rules":    enabledRules,
		"disabled_rules":   totalRules - enabledRules,
		"total_executions": totalExecutions,
		"service_stats":    serviceStats,
		"type_stats":       typeStats,
	}
}

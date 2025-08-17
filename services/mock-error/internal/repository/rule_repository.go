package repository

import (
	"context"
	"fmt"
	"mocks3/shared/models"
	"sort"
	"sync"
	"time"
)

// RuleRepository 错误规则仓库
type RuleRepository struct {
	rules map[string]*models.ErrorRule
	mu    sync.RWMutex
}

// NewRuleRepository 创建错误规则仓库
func NewRuleRepository() *RuleRepository {
	return &RuleRepository{
		rules: make(map[string]*models.ErrorRule),
	}
}

// Add 添加规则
func (r *RuleRepository) Add(ctx context.Context, rule *models.ErrorRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rule.ID == "" {
		rule.ID = generateRuleID()
	}

	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	r.rules[rule.ID] = rule
	return nil
}

// Update 更新规则
func (r *RuleRepository) Update(ctx context.Context, rule *models.ErrorRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rules[rule.ID]; !exists {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	rule.UpdatedAt = time.Now()
	r.rules[rule.ID] = rule
	return nil
}

// Delete 删除规则
func (r *RuleRepository) Delete(ctx context.Context, ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rules[ruleID]; !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	delete(r.rules, ruleID)
	return nil
}

// Get 获取规则
func (r *RuleRepository) Get(ctx context.Context, ruleID string) (*models.ErrorRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rule, exists := r.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}

	// 返回副本
	ruleCopy := *rule
	return &ruleCopy, nil
}

// List 列出所有规则
func (r *RuleRepository) List(ctx context.Context) ([]*models.ErrorRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := make([]*models.ErrorRule, 0, len(r.rules))
	for _, rule := range r.rules {
		ruleCopy := *rule
		rules = append(rules, &ruleCopy)
	}

	// 按优先级和创建时间排序
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority < rules[j].Priority
		}
		return rules[i].CreatedAt.Before(rules[j].CreatedAt)
	})

	return rules, nil
}

// ListActive 列出活跃规则
func (r *RuleRepository) ListActive(ctx context.Context) ([]*models.ErrorRule, error) {
	rules, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	activeRules := make([]*models.ErrorRule, 0)
	for _, rule := range rules {
		if rule.Enabled && !r.isRuleExpired(rule) && !r.isRuleExhausted(rule) {
			activeRules = append(activeRules, rule)
		}
	}

	return activeRules, nil
}

// FindByService 按服务查找规则
func (r *RuleRepository) FindByService(ctx context.Context, service string) ([]*models.ErrorRule, error) {
	rules, err := r.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	serviceRules := make([]*models.ErrorRule, 0)
	for _, rule := range rules {
		if rule.Service == "" || rule.Service == service {
			serviceRules = append(serviceRules, rule)
		}
	}

	return serviceRules, nil
}

// FindByServiceAndOperation 按服务和操作查找规则
func (r *RuleRepository) FindByServiceAndOperation(ctx context.Context, service, operation string) ([]*models.ErrorRule, error) {
	serviceRules, err := r.FindByService(ctx, service)
	if err != nil {
		return nil, err
	}

	operationRules := make([]*models.ErrorRule, 0)
	for _, rule := range serviceRules {
		if rule.Operation == "" || rule.Operation == operation {
			operationRules = append(operationRules, rule)
		}
	}

	return operationRules, nil
}

// IncrementTriggerCount 增加触发次数
func (r *RuleRepository) IncrementTriggerCount(ctx context.Context, ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rule, exists := r.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	rule.Triggered++
	rule.UpdatedAt = time.Now()
	return nil
}

// EnableRule 启用规则
func (r *RuleRepository) EnableRule(ctx context.Context, ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rule, exists := r.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	rule.Enabled = true
	rule.UpdatedAt = time.Now()
	return nil
}

// DisableRule 禁用规则
func (r *RuleRepository) DisableRule(ctx context.Context, ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rule, exists := r.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	rule.Enabled = false
	rule.UpdatedAt = time.Now()
	return nil
}

// Count 获取规则数量
func (r *RuleRepository) Count(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.rules), nil
}

// CountActive 获取活跃规则数量
func (r *RuleRepository) CountActive(ctx context.Context) (int, error) {
	rules, err := r.ListActive(ctx)
	if err != nil {
		return 0, err
	}
	return len(rules), nil
}

// isRuleExpired 检查规则是否已过期
func (r *RuleRepository) isRuleExpired(rule *models.ErrorRule) bool {
	if rule.Schedule == nil || rule.Schedule.EndTime == nil {
		return false
	}
	return time.Now().After(*rule.Schedule.EndTime)
}

// isRuleExhausted 检查规则是否已用尽
func (r *RuleRepository) isRuleExhausted(rule *models.ErrorRule) bool {
	if rule.MaxTriggers <= 0 {
		return false
	}
	return rule.Triggered >= rule.MaxTriggers
}

// generateRuleID 生成规则ID
func generateRuleID() string {
	return fmt.Sprintf("rule_%d", time.Now().UnixNano())
}

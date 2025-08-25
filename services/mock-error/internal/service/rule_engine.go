package service

import (
	"context"
	"fmt"
	"math/rand"
	"mocks3/shared/interfaces"
	"mocks3/shared/models"
	"mocks3/shared/observability"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RuleEngine 错误规则引擎实现
type RuleEngine struct {
	rules  map[string]*models.ErrorRule
	logger *observability.Logger
	rand   *rand.Rand
}

// NewRuleEngine 创建错误规则引擎
func NewRuleEngine(logger *observability.Logger) *RuleEngine {
	return &RuleEngine{
		rules:  make(map[string]*models.ErrorRule),
		logger: logger,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// EvaluateRules 评估规则
func (e *RuleEngine) EvaluateRules(ctx context.Context, service, operation string, metadata map[string]string) (*models.ErrorAction, bool) {
	// 按优先级获取匹配的规则
	matchedRules := e.getMatchingRules(service, operation)

	for _, rule := range matchedRules {
		// 检查规则是否活跃
		if !e.isRuleActive(rule) {
			continue
		}

		// 评估条件
		if e.evaluateConditions(rule.Conditions, metadata) {
			e.logger.Debug(ctx, "Rule matched",
				observability.String("rule_id", rule.ID),
				observability.String("rule_name", rule.Name),
				observability.String("service", service),
				observability.String("operation", operation))

			return &rule.Action, true
		}
	}

	return nil, false
}

// AddRule 添加规则
func (e *RuleEngine) AddRule(rule *models.ErrorRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	e.rules[rule.ID] = rule
	e.logger.Debug(context.Background(), "Rule added", 
		observability.String("rule_id", rule.ID), 
		observability.String("rule_name", rule.Name))
	return nil
}

// RemoveRule 移除规则
func (e *RuleEngine) RemoveRule(ruleID string) error {
	if _, exists := e.rules[ruleID]; !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	delete(e.rules, ruleID)
	e.logger.Debug(context.Background(), "Rule removed", 
		observability.String("rule_id", ruleID))
	return nil
}

// UpdateRule 更新规则
func (e *RuleEngine) UpdateRule(rule *models.ErrorRule) error {
	if _, exists := e.rules[rule.ID]; !exists {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	e.rules[rule.ID] = rule
	e.logger.Debug(context.Background(), "Rule updated", 
		observability.String("rule_id", rule.ID), 
		observability.String("rule_name", rule.Name))
	return nil
}

// GetRule 获取规则
func (e *RuleEngine) GetRule(ruleID string) (*models.ErrorRule, error) {
	rule, exists := e.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}

	// 返回副本
	ruleCopy := *rule
	return &ruleCopy, nil
}

// ListRules 列出所有规则
func (e *RuleEngine) ListRules() []*models.ErrorRule {
	rules := make([]*models.ErrorRule, 0, len(e.rules))
	for _, rule := range e.rules {
		ruleCopy := *rule
		rules = append(rules, &ruleCopy)
	}

	return rules
}

// getMatchingRules 获取匹配的规则
func (e *RuleEngine) getMatchingRules(service, operation string) []*models.ErrorRule {
	var matched []*models.ErrorRule

	for _, rule := range e.rules {
		if e.isRuleMatching(rule, service, operation) {
			matched = append(matched, rule)
		}
	}

	// 按优先级排序
	for i := 0; i < len(matched)-1; i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[i].Priority > matched[j].Priority {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}

	return matched
}

// isRuleMatching 检查规则是否匹配服务和操作
func (e *RuleEngine) isRuleMatching(rule *models.ErrorRule, service, operation string) bool {
	// 检查服务匹配
	if rule.Service != "" && rule.Service != service {
		return false
	}

	// 检查操作匹配
	if rule.Operation != "" && rule.Operation != operation {
		return false
	}

	return true
}

// isRuleActive 检查规则是否活跃
func (e *RuleEngine) isRuleActive(rule *models.ErrorRule) bool {
	// 检查是否启用
	if !rule.Enabled {
		return false
	}

	// 检查触发次数限制
	if rule.MaxTriggers > 0 && rule.Triggered >= rule.MaxTriggers {
		return false
	}

	// 检查时间调度
	if rule.Schedule != nil {
		if !e.isScheduleActive(rule.Schedule) {
			return false
		}
	}

	return true
}

// isScheduleActive 检查调度是否活跃
func (e *RuleEngine) isScheduleActive(schedule *models.ErrorSchedule) bool {
	now := time.Now()

	// 检查时区
	if schedule.Timezone != "" {
		loc, err := time.LoadLocation(schedule.Timezone)
		if err == nil {
			now = now.In(loc)
		}
	}

	// 检查开始时间
	if schedule.StartTime != nil && now.Before(*schedule.StartTime) {
		return false
	}

	// 检查结束时间
	if schedule.EndTime != nil && now.After(*schedule.EndTime) {
		return false
	}

	// 检查日期
	if len(schedule.Days) > 0 {
		dayName := strings.ToLower(now.Weekday().String())
		dayMatched := false
		for _, day := range schedule.Days {
			if strings.ToLower(day) == dayName {
				dayMatched = true
				break
			}
		}
		if !dayMatched {
			return false
		}
	}

	// 检查小时
	if len(schedule.Hours) > 0 {
		hour := now.Hour()
		hourMatched := false
		for _, h := range schedule.Hours {
			if h == hour {
				hourMatched = true
				break
			}
		}
		if !hourMatched {
			return false
		}
	}

	return true
}

// evaluateConditions 评估条件
func (e *RuleEngine) evaluateConditions(conditions []models.ErrorCondition, metadata map[string]string) bool {
	if len(conditions) == 0 {
		return true
	}

	// 所有条件都必须满足（AND 逻辑）
	for _, condition := range conditions {
		if !e.evaluateCondition(condition, metadata) {
			return false
		}
	}

	return true
}

// evaluateCondition 评估单个条件
func (e *RuleEngine) evaluateCondition(condition models.ErrorCondition, metadata map[string]string) bool {
	switch condition.Type {
	case models.ErrorConditionTypeProbability:
		return e.evaluateProbabilityCondition(condition)
	case models.ErrorConditionTypeHeader:
		return e.evaluateHeaderCondition(condition, metadata)
	case models.ErrorConditionTypeParam:
		return e.evaluateParamCondition(condition, metadata)
	case models.ErrorConditionTypeTime:
		return e.evaluateTimeCondition(condition)
	case models.ErrorConditionTypeUserAgent:
		return e.evaluateUserAgentCondition(condition, metadata)
	case models.ErrorConditionTypeIP:
		return e.evaluateIPCondition(condition, metadata)
	case models.ErrorConditionTypeCount:
		return e.evaluateCountCondition(condition, metadata)
	default:
		e.logger.Warn(context.Background(), "Unknown condition type", 
			observability.String("type", condition.Type))
		return false
	}
}

// evaluateProbabilityCondition 评估概率条件
func (e *RuleEngine) evaluateProbabilityCondition(condition models.ErrorCondition) bool {
	probability, ok := condition.Value.(float64)
	if !ok {
		// 尝试从字符串解析
		if str, ok := condition.Value.(string); ok {
			if p, err := strconv.ParseFloat(str, 64); err == nil {
				probability = p
			} else {
				return false
			}
		} else {
			return false
		}
	}

	if probability <= 0 {
		return false
	}
	if probability >= 1 {
		return true
	}

	random := e.rand.Float64()
	return random < probability
}

// evaluateHeaderCondition 评估请求头条件
func (e *RuleEngine) evaluateHeaderCondition(condition models.ErrorCondition, metadata map[string]string) bool {
	headerValue, exists := metadata["header_"+condition.Field]
	if !exists {
		return false
	}

	expectedValue := fmt.Sprintf("%v", condition.Value)
	return e.compareValues(headerValue, expectedValue, condition.Operator)
}

// evaluateParamCondition 评估参数条件
func (e *RuleEngine) evaluateParamCondition(condition models.ErrorCondition, metadata map[string]string) bool {
	paramValue, exists := metadata["param_"+condition.Field]
	if !exists {
		return false
	}

	expectedValue := fmt.Sprintf("%v", condition.Value)
	return e.compareValues(paramValue, expectedValue, condition.Operator)
}

// evaluateTimeCondition 评估时间条件
func (e *RuleEngine) evaluateTimeCondition(condition models.ErrorCondition) bool {
	now := time.Now()
	expectedTime, err := time.Parse(time.RFC3339, fmt.Sprintf("%v", condition.Value))
	if err != nil {
		return false
	}

	switch condition.Operator {
	case "eq":
		return now.Equal(expectedTime)
	case "ne":
		return !now.Equal(expectedTime)
	case "gt":
		return now.After(expectedTime)
	case "lt":
		return now.Before(expectedTime)
	case "gte":
		return now.After(expectedTime) || now.Equal(expectedTime)
	case "lte":
		return now.Before(expectedTime) || now.Equal(expectedTime)
	default:
		return false
	}
}

// evaluateUserAgentCondition 评估User-Agent条件
func (e *RuleEngine) evaluateUserAgentCondition(condition models.ErrorCondition, metadata map[string]string) bool {
	userAgent, exists := metadata["user_agent"]
	if !exists {
		return false
	}

	expectedValue := fmt.Sprintf("%v", condition.Value)
	return e.compareValues(userAgent, expectedValue, condition.Operator)
}

// evaluateIPCondition 评估IP地址条件
func (e *RuleEngine) evaluateIPCondition(condition models.ErrorCondition, metadata map[string]string) bool {
	clientIP, exists := metadata["remote_addr"]
	if !exists {
		return false
	}

	expectedValue := fmt.Sprintf("%v", condition.Value)

	// 支持CIDR匹配
	if strings.Contains(expectedValue, "/") {
		_, network, err := net.ParseCIDR(expectedValue)
		if err != nil {
			return false
		}

		ip := net.ParseIP(clientIP)
		if ip == nil {
			return false
		}

		return network.Contains(ip)
	}

	return e.compareValues(clientIP, expectedValue, condition.Operator)
}

// evaluateCountCondition 评估计数条件
func (e *RuleEngine) evaluateCountCondition(condition models.ErrorCondition, metadata map[string]string) bool {
	countStr, exists := metadata["request_count"]
	if !exists {
		return false
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return false
	}

	expectedCount, err := strconv.Atoi(fmt.Sprintf("%v", condition.Value))
	if err != nil {
		return false
	}

	switch condition.Operator {
	case "eq":
		return count == expectedCount
	case "ne":
		return count != expectedCount
	case "gt":
		return count > expectedCount
	case "lt":
		return count < expectedCount
	case "gte":
		return count >= expectedCount
	case "lte":
		return count <= expectedCount
	default:
		return false
	}
}

// compareValues 比较值
func (e *RuleEngine) compareValues(actual, expected, operator string) bool {
	switch operator {
	case "eq":
		return actual == expected
	case "ne":
		return actual != expected
	case "contains":
		return strings.Contains(actual, expected)
	case "not_contains":
		return !strings.Contains(actual, expected)
	case "starts_with":
		return strings.HasPrefix(actual, expected)
	case "ends_with":
		return strings.HasSuffix(actual, expected)
	case "regex":
		matched, err := regexp.MatchString(expected, actual)
		return err == nil && matched
	case "gt":
		return actual > expected
	case "lt":
		return actual < expected
	case "gte":
		return actual >= expected
	case "lte":
		return actual <= expected
	default:
		e.logger.Warn(context.Background(), "Unknown operator", 
			observability.String("operator", operator))
		return false
	}
}

// 确保实现了接口
var _ interfaces.ErrorRuleEngine = (*RuleEngine)(nil)

package service

import (
	"context"
	"fmt"
	"mocks3/services/mock-error/internal/config"
	"mocks3/services/mock-error/internal/repository"
	"mocks3/shared/interfaces"
	"mocks3/shared/models"
	"mocks3/shared/observability/log"
	"time"

	"github.com/google/uuid"
)

// ErrorInjectorService 错误注入服务实现
type ErrorInjectorService struct {
	config     *config.Config
	ruleRepo   *repository.RuleRepository
	statsRepo  *repository.StatsRepository
	ruleEngine interfaces.ErrorRuleEngine
	logger     *log.Logger
}

// NewErrorInjectorService 创建错误注入服务
func NewErrorInjectorService(
	cfg *config.Config,
	ruleRepo *repository.RuleRepository,
	statsRepo *repository.StatsRepository,
	ruleEngine interfaces.ErrorRuleEngine,
	logger *log.Logger,
) *ErrorInjectorService {
	return &ErrorInjectorService{
		config:     cfg,
		ruleRepo:   ruleRepo,
		statsRepo:  statsRepo,
		ruleEngine: ruleEngine,
		logger:     logger,
	}
}

// AddErrorRule 添加错误规则
func (s *ErrorInjectorService) AddErrorRule(ctx context.Context, rule *models.ErrorRule) error {
	s.logger.InfoContext(ctx, "Adding error rule", "rule_name", rule.Name, "service", rule.Service)

	// 验证规则
	if err := s.validateRule(rule); err != nil {
		s.logger.WarnContext(ctx, "Invalid rule", "error", err)
		return fmt.Errorf("invalid rule: %w", err)
	}

	// 检查规则数量限制
	count, err := s.ruleRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count rules: %w", err)
	}

	if count >= s.config.ErrorEngine.MaxRules {
		return fmt.Errorf("maximum number of rules reached: %d", s.config.ErrorEngine.MaxRules)
	}

	// 生成ID
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	// 添加到仓库
	if err := s.ruleRepo.Add(ctx, rule); err != nil {
		s.logger.ErrorContext(ctx, "Failed to add rule to repository", "error", err)
		return fmt.Errorf("failed to add rule: %w", err)
	}

	// 添加到规则引擎
	if err := s.ruleEngine.AddRule(rule); err != nil {
		s.logger.ErrorContext(ctx, "Failed to add rule to engine", "error", err)
		// 回滚仓库操作
		s.ruleRepo.Delete(ctx, rule.ID)
		return fmt.Errorf("failed to add rule to engine: %w", err)
	}

	// 更新统计
	s.updateRuleCounts(ctx)

	s.logger.InfoContext(ctx, "Error rule added successfully", "rule_id", rule.ID, "rule_name", rule.Name)
	return nil
}

// RemoveErrorRule 移除错误规则
func (s *ErrorInjectorService) RemoveErrorRule(ctx context.Context, ruleID string) error {
	s.logger.InfoContext(ctx, "Removing error rule", "rule_id", ruleID)

	// 从仓库删除
	if err := s.ruleRepo.Delete(ctx, ruleID); err != nil {
		s.logger.WarnContext(ctx, "Failed to remove rule from repository", "rule_id", ruleID, "error", err)
		return fmt.Errorf("failed to remove rule: %w", err)
	}

	// 从规则引擎删除
	if err := s.ruleEngine.RemoveRule(ruleID); err != nil {
		s.logger.WarnContext(ctx, "Failed to remove rule from engine", "rule_id", ruleID, "error", err)
	}

	// 更新统计
	s.updateRuleCounts(ctx)

	s.logger.InfoContext(ctx, "Error rule removed successfully", "rule_id", ruleID)
	return nil
}

// UpdateErrorRule 更新错误规则
func (s *ErrorInjectorService) UpdateErrorRule(ctx context.Context, rule *models.ErrorRule) error {
	s.logger.InfoContext(ctx, "Updating error rule", "rule_id", rule.ID, "rule_name", rule.Name)

	// 验证规则
	if err := s.validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// 更新仓库
	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		s.logger.ErrorContext(ctx, "Failed to update rule in repository", "error", err)
		return fmt.Errorf("failed to update rule: %w", err)
	}

	// 更新规则引擎
	if err := s.ruleEngine.UpdateRule(rule); err != nil {
		s.logger.ErrorContext(ctx, "Failed to update rule in engine", "error", err)
		return fmt.Errorf("failed to update rule in engine: %w", err)
	}

	s.logger.InfoContext(ctx, "Error rule updated successfully", "rule_id", rule.ID)
	return nil
}

// GetErrorRule 获取错误规则
func (s *ErrorInjectorService) GetErrorRule(ctx context.Context, ruleID string) (*models.ErrorRule, error) {
	s.logger.DebugContext(ctx, "Getting error rule", "rule_id", ruleID)

	rule, err := s.ruleRepo.Get(ctx, ruleID)
	if err != nil {
		s.logger.WarnContext(ctx, "Rule not found", "rule_id", ruleID)
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	return rule, nil
}

// ListErrorRules 列出错误规则
func (s *ErrorInjectorService) ListErrorRules(ctx context.Context) ([]*models.ErrorRule, error) {
	s.logger.DebugContext(ctx, "Listing error rules")

	rules, err := s.ruleRepo.List(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list rules", "error", err)
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}

	s.logger.DebugContext(ctx, "Listed error rules", "count", len(rules))
	return rules, nil
}

// ShouldInjectError 检查是否应该注入错误
func (s *ErrorInjectorService) ShouldInjectError(ctx context.Context, service, operation string) (*models.ErrorAction, bool) {
	// 检查全局概率
	if s.config.Injection.GlobalProbability < 1.0 {
		// TODO: 实现全局概率检查
	}

	// 从请求上下文中提取元数据
	metadata := s.extractMetadata(ctx)

	// 使用规则引擎评估
	action, shouldInject := s.ruleEngine.EvaluateRules(ctx, service, operation, metadata)

	if shouldInject {
		s.logger.DebugContext(ctx, "Error injection triggered",
			"service", service,
			"operation", operation,
			"action_type", action.Type)

		// 记录事件
		event := &models.ErrorEvent{
			ID:        uuid.New().String(),
			Service:   service,
			Operation: operation,
			Action:    *action,
			Timestamp: time.Now(),
			Success:   true,
		}

		// 异步记录统计
		go func() {
			if err := s.statsRepo.RecordEvent(context.Background(), event); err != nil {
				s.logger.Warn("Failed to record error event", "error", err)
			}
		}()
	}

	return action, shouldInject
}

// InjectError 执行错误注入
func (s *ErrorInjectorService) InjectError(ctx context.Context, action *models.ErrorAction) error {
	s.logger.DebugContext(ctx, "Injecting error", "action_type", action.Type)

	switch action.Type {
	case models.ErrorActionTypeDelay:
		return s.injectDelay(ctx, action)
	case models.ErrorActionTypeHTTPError:
		// HTTP错误由中间件处理
		return nil
	case models.ErrorActionTypeNetworkError:
		return s.injectNetworkError(ctx, action)
	case models.ErrorActionTypeDatabaseError:
		return s.injectDatabaseError(ctx, action)
	case models.ErrorActionTypeStorageError:
		return s.injectStorageError(ctx, action)
	default:
		return fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

// GetErrorStats 获取错误统计
func (s *ErrorInjectorService) GetErrorStats(ctx context.Context) (*models.ErrorStats, error) {
	s.logger.DebugContext(ctx, "Getting error statistics")

	// 更新规则计数
	s.updateRuleCounts(ctx)

	stats, err := s.statsRepo.GetStats(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get statistics", "error", err)
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return stats, nil
}

// ResetErrorStats 重置错误统计
func (s *ErrorInjectorService) ResetErrorStats(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Resetting error statistics")

	if err := s.statsRepo.ResetStats(ctx); err != nil {
		s.logger.ErrorContext(ctx, "Failed to reset statistics", "error", err)
		return fmt.Errorf("failed to reset statistics: %w", err)
	}

	s.logger.InfoContext(ctx, "Error statistics reset successfully")
	return nil
}

// HealthCheck 健康检查
func (s *ErrorInjectorService) HealthCheck(ctx context.Context) error {
	s.logger.DebugContext(ctx, "Performing health check")

	// 检查规则数量
	count, err := s.ruleRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count rules: %w", err)
	}

	s.logger.DebugContext(ctx, "Health check passed", "rule_count", count)
	return nil
}

// validateRule 验证规则
func (s *ErrorInjectorService) validateRule(rule *models.ErrorRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	if rule.Action.Type == "" {
		return fmt.Errorf("action type is required")
	}

	// 验证动作类型
	validActionTypes := map[string]bool{
		models.ErrorActionTypeHTTPError:     true,
		models.ErrorActionTypeNetworkError:  true,
		models.ErrorActionTypeTimeout:       true,
		models.ErrorActionTypeDelay:         true,
		models.ErrorActionTypeCorruption:    true,
		models.ErrorActionTypeDisconnect:    true,
		models.ErrorActionTypeDatabaseError: true,
		models.ErrorActionTypeStorageError:  true,
	}

	if !validActionTypes[rule.Action.Type] {
		return fmt.Errorf("invalid action type: %s", rule.Action.Type)
	}

	// 验证HTTP错误码
	if rule.Action.Type == models.ErrorActionTypeHTTPError {
		if rule.Action.HTTPCode < 400 || rule.Action.HTTPCode >= 600 {
			return fmt.Errorf("invalid HTTP code: %d", rule.Action.HTTPCode)
		}
	}

	// 验证延迟时间
	if rule.Action.Delay != nil {
		maxDelay := time.Duration(s.config.Injection.MaxDelayMs) * time.Millisecond
		if *rule.Action.Delay > maxDelay {
			return fmt.Errorf("delay exceeds maximum allowed: %v", maxDelay)
		}
	}

	return nil
}

// extractMetadata 从上下文提取元数据
func (s *ErrorInjectorService) extractMetadata(ctx context.Context) map[string]string {
	metadata := make(map[string]string)

	// 从上下文中提取信息（根据实际需要实现）
	// 这里是示例实现
	if userAgent := ctx.Value("user_agent"); userAgent != nil {
		metadata["user_agent"] = fmt.Sprintf("%v", userAgent)
	}

	if remoteAddr := ctx.Value("remote_addr"); remoteAddr != nil {
		metadata["remote_addr"] = fmt.Sprintf("%v", remoteAddr)
	}

	return metadata
}

// updateRuleCounts 更新规则计数统计
func (s *ErrorInjectorService) updateRuleCounts(ctx context.Context) {
	totalRules, _ := s.ruleRepo.Count(ctx)
	activeRules, _ := s.ruleRepo.CountActive(ctx)

	go func() {
		if err := s.statsRepo.UpdateRuleCounts(context.Background(), totalRules, activeRules); err != nil {
			s.logger.Warn("Failed to update rule counts", "error", err)
		}
	}()
}

// injectDelay 注入延迟
func (s *ErrorInjectorService) injectDelay(ctx context.Context, action *models.ErrorAction) error {
	if action.Delay == nil {
		return nil
	}

	s.logger.DebugContext(ctx, "Injecting delay", "duration", *action.Delay)

	select {
	case <-time.After(*action.Delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// injectNetworkError 注入网络错误
func (s *ErrorInjectorService) injectNetworkError(ctx context.Context, action *models.ErrorAction) error {
	if !s.config.Injection.EnableNetworkErrors {
		return nil
	}

	s.logger.DebugContext(ctx, "Injecting network error")
	return fmt.Errorf("network error injected: %s", action.Message)
}

// injectDatabaseError 注入数据库错误
func (s *ErrorInjectorService) injectDatabaseError(ctx context.Context, action *models.ErrorAction) error {
	if !s.config.Injection.EnableDatabaseErrors {
		return nil
	}

	s.logger.DebugContext(ctx, "Injecting database error")
	return fmt.Errorf("database error injected: %s", action.Message)
}

// injectStorageError 注入存储错误
func (s *ErrorInjectorService) injectStorageError(ctx context.Context, action *models.ErrorAction) error {
	if !s.config.Injection.EnableStorageErrors {
		return nil
	}

	s.logger.DebugContext(ctx, "Injecting storage error")
	return fmt.Errorf("storage error injected: %s", action.Message)
}

// 确保实现了接口
var _ interfaces.ErrorInjectorService = (*ErrorInjectorService)(nil)

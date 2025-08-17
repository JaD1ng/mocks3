package interfaces

import (
	"context"
	"mocks3/shared/models"
	"net/http"
)

// ErrorInjectorService 错误注入服务接口
type ErrorInjectorService interface {
	// 错误规则管理
	AddErrorRule(ctx context.Context, rule *models.ErrorRule) error
	RemoveErrorRule(ctx context.Context, ruleID string) error
	UpdateErrorRule(ctx context.Context, rule *models.ErrorRule) error
	GetErrorRule(ctx context.Context, ruleID string) (*models.ErrorRule, error)
	ListErrorRules(ctx context.Context) ([]*models.ErrorRule, error)

	// 错误注入执行
	ShouldInjectError(ctx context.Context, service, operation string) (*models.ErrorAction, bool)
	InjectError(ctx context.Context, action *models.ErrorAction) error

	// 统计信息
	GetErrorStats(ctx context.Context) (*models.ErrorStats, error)
	ResetErrorStats(ctx context.Context) error

	// 健康检查
	HealthCheck(ctx context.Context) error
}

// ErrorInjector 错误注入器接口
type ErrorInjector interface {
	// HTTP 中间件
	HTTPMiddleware() func(http.Handler) http.Handler

	// 错误注入方法
	InjectHTTPError(w http.ResponseWriter, r *http.Request, action *models.ErrorAction) bool
	InjectNetworkError(ctx context.Context, action *models.ErrorAction) error
	InjectDatabaseError(ctx context.Context, action *models.ErrorAction) error
	InjectStorageError(ctx context.Context, action *models.ErrorAction) error
}

// ErrorRuleEngine 错误规则引擎接口
type ErrorRuleEngine interface {
	EvaluateRules(ctx context.Context, service, operation string, metadata map[string]string) (*models.ErrorAction, bool)
	AddRule(rule *models.ErrorRule) error
	RemoveRule(ruleID string) error
	UpdateRule(rule *models.ErrorRule) error
	GetRule(ruleID string) (*models.ErrorRule, error)
	ListRules() []*models.ErrorRule
}

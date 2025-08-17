package repository

import (
	"context"
	"mocks3/shared/models"
	"sync"
	"time"
)

// StatsRepository 统计仓库
type StatsRepository struct {
	stats          *models.ErrorStats
	events         []*models.ErrorEvent
	maxEvents      int
	mu             sync.RWMutex
	retentionHours int
}

// NewStatsRepository 创建统计仓库
func NewStatsRepository(maxEvents int, retentionHours int) *StatsRepository {
	now := time.Now()
	return &StatsRepository{
		stats: &models.ErrorStats{
			TotalRules:     0,
			ActiveRules:    0,
			TotalTriggers:  0,
			RuleStats:      make(map[string]*models.RuleStat),
			ServiceStats:   make(map[string]*models.ServiceStat),
			ErrorTypeStats: make(map[string]int64),
			LastReset:      now,
			LastUpdate:     now,
		},
		events:         make([]*models.ErrorEvent, 0),
		maxEvents:      maxEvents,
		retentionHours: retentionHours,
	}
}

// RecordEvent 记录错误事件
func (r *StatsRepository) RecordEvent(ctx context.Context, event *models.ErrorEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 添加事件
	r.events = append(r.events, event)

	// 保持事件数量限制
	if len(r.events) > r.maxEvents {
		r.events = r.events[len(r.events)-r.maxEvents:]
	}

	// 更新统计
	r.updateStats(event)

	return nil
}

// GetStats 获取统计信息
func (r *StatsRepository) GetStats(ctx context.Context) (*models.ErrorStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 清理过期数据
	r.cleanupExpiredData()

	// 返回统计副本
	statsCopy := r.copyStats()
	return statsCopy, nil
}

// ResetStats 重置统计
func (r *StatsRepository) ResetStats(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.stats = &models.ErrorStats{
		TotalRules:     r.stats.TotalRules,
		ActiveRules:    r.stats.ActiveRules,
		TotalTriggers:  0,
		RuleStats:      make(map[string]*models.RuleStat),
		ServiceStats:   make(map[string]*models.ServiceStat),
		ErrorTypeStats: make(map[string]int64),
		LastReset:      now,
		LastUpdate:     now,
	}

	r.events = make([]*models.ErrorEvent, 0)

	return nil
}

// UpdateRuleCounts 更新规则计数
func (r *StatsRepository) UpdateRuleCounts(ctx context.Context, totalRules, activeRules int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stats.TotalRules = totalRules
	r.stats.ActiveRules = activeRules
	r.stats.LastUpdate = time.Now()

	return nil
}

// GetEvents 获取错误事件
func (r *StatsRepository) GetEvents(ctx context.Context, limit int) ([]*models.ErrorEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 清理过期事件
	r.cleanupExpiredEvents()

	if limit <= 0 || limit > len(r.events) {
		limit = len(r.events)
	}

	// 返回最新的事件
	start := len(r.events) - limit
	if start < 0 {
		start = 0
	}

	events := make([]*models.ErrorEvent, limit)
	copy(events, r.events[start:])

	return events, nil
}

// GetServiceStats 获取服务统计
func (r *StatsRepository) GetServiceStats(ctx context.Context, service string) (*models.ServiceStat, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stat, exists := r.stats.ServiceStats[service]
	if !exists {
		return &models.ServiceStat{
			ServiceName:    service,
			TotalRequests:  0,
			ErrorRequests:  0,
			ErrorRate:      0,
			OperationStats: make(map[string]*models.OpStat),
		}, nil
	}

	// 返回副本
	statCopy := *stat
	statCopy.OperationStats = make(map[string]*models.OpStat)
	for k, v := range stat.OperationStats {
		opStatCopy := *v
		statCopy.OperationStats[k] = &opStatCopy
	}

	return &statCopy, nil
}

// updateStats 更新统计信息
func (r *StatsRepository) updateStats(event *models.ErrorEvent) {
	now := time.Now()
	r.stats.TotalTriggers++
	r.stats.LastUpdate = now

	// 计算小时和日期范围
	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	// 更新小时和今日触发数
	if event.Timestamp.After(oneHourAgo) {
		r.stats.TriggersLastHour++
	}
	if event.Timestamp.After(oneDayAgo) {
		r.stats.TriggersToday++
	}

	// 更新规则统计
	if event.RuleID != "" {
		ruleStat, exists := r.stats.RuleStats[event.RuleID]
		if !exists {
			ruleStat = &models.RuleStat{
				RuleID:        event.RuleID,
				RuleName:      event.RuleName,
				TotalTriggers: 0,
				ErrorCounts:   make(map[string]int64),
			}
			r.stats.RuleStats[event.RuleID] = ruleStat
		}

		ruleStat.TotalTriggers++
		ruleStat.LastTriggered = event.Timestamp

		if event.Action.Type != "" {
			ruleStat.ErrorCounts[event.Action.Type]++
		}
	}

	// 更新服务统计
	if event.Service != "" {
		serviceStat, exists := r.stats.ServiceStats[event.Service]
		if !exists {
			serviceStat = &models.ServiceStat{
				ServiceName:    event.Service,
				TotalRequests:  0,
				ErrorRequests:  0,
				ErrorRate:      0,
				OperationStats: make(map[string]*models.OpStat),
			}
			r.stats.ServiceStats[event.Service] = serviceStat
		}

		serviceStat.TotalRequests++
		if event.Success {
			serviceStat.ErrorRequests++
		}

		// 更新错误率
		if serviceStat.TotalRequests > 0 {
			serviceStat.ErrorRate = float64(serviceStat.ErrorRequests) / float64(serviceStat.TotalRequests)
		}

		// 更新操作统计
		if event.Operation != "" {
			opStat, exists := serviceStat.OperationStats[event.Operation]
			if !exists {
				opStat = &models.OpStat{
					OperationName: event.Operation,
					TotalRequests: 0,
					ErrorRequests: 0,
					ErrorRate:     0,
				}
				serviceStat.OperationStats[event.Operation] = opStat
			}

			opStat.TotalRequests++
			if event.Success {
				opStat.ErrorRequests++
			}

			// 更新操作错误率
			if opStat.TotalRequests > 0 {
				opStat.ErrorRate = float64(opStat.ErrorRequests) / float64(opStat.TotalRequests)
			}
		}
	}

	// 更新错误类型统计
	if event.Action.Type != "" {
		r.stats.ErrorTypeStats[event.Action.Type]++
	}
}

// cleanupExpiredData 清理过期数据
func (r *StatsRepository) cleanupExpiredData() {
	now := time.Now()
	cutoff := now.Add(-time.Duration(r.retentionHours) * time.Hour)

	// 重新计算时间相关的统计
	r.stats.TriggersLastHour = 0
	r.stats.TriggersToday = 0

	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	for _, event := range r.events {
		if event.Timestamp.After(cutoff) {
			if event.Timestamp.After(oneHourAgo) {
				r.stats.TriggersLastHour++
			}
			if event.Timestamp.After(oneDayAgo) {
				r.stats.TriggersToday++
			}
		}
	}
}

// cleanupExpiredEvents 清理过期事件
func (r *StatsRepository) cleanupExpiredEvents() {
	if r.retentionHours <= 0 {
		return
	}

	cutoff := time.Now().Add(-time.Duration(r.retentionHours) * time.Hour)

	// 找到第一个未过期的事件
	startIndex := 0
	for i, event := range r.events {
		if event.Timestamp.After(cutoff) {
			startIndex = i
			break
		}
	}

	// 移除过期事件
	if startIndex > 0 {
		r.events = r.events[startIndex:]
	}
}

// copyStats 复制统计信息
func (r *StatsRepository) copyStats() *models.ErrorStats {
	statsCopy := *r.stats

	// 深拷贝映射
	statsCopy.RuleStats = make(map[string]*models.RuleStat)
	for k, v := range r.stats.RuleStats {
		ruleStat := *v
		ruleStat.ErrorCounts = make(map[string]int64)
		for ek, ev := range v.ErrorCounts {
			ruleStat.ErrorCounts[ek] = ev
		}
		statsCopy.RuleStats[k] = &ruleStat
	}

	statsCopy.ServiceStats = make(map[string]*models.ServiceStat)
	for k, v := range r.stats.ServiceStats {
		serviceStat := *v
		serviceStat.OperationStats = make(map[string]*models.OpStat)
		for ok, ov := range v.OperationStats {
			opStat := *ov
			serviceStat.OperationStats[ok] = &opStat
		}
		statsCopy.ServiceStats[k] = &serviceStat
	}

	statsCopy.ErrorTypeStats = make(map[string]int64)
	for k, v := range r.stats.ErrorTypeStats {
		statsCopy.ErrorTypeStats[k] = v
	}

	return &statsCopy
}

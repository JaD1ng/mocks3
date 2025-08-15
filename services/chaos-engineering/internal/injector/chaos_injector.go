package injector

import (
	"chaos-engineering/internal/rules"
	"context"
	"fmt"
	"math/rand"
	"github.com/mocks3/shared/logger"
	"os"
	"runtime"
	"sync"
	"time"
)

// ChaosInjector 混沌注入器
type ChaosInjector struct {
	ruleManager     *rules.RuleManager
	running         bool
	stopChan        chan struct{}
	activeInjectors map[string]context.CancelFunc
	mu              sync.RWMutex
	executionLog    []ExecutionRecord
	maxLogSize      int
}

// ExecutionRecord 执行记录
type ExecutionRecord struct {
	RuleID      string         `json:"rule_id"`
	RuleName    string         `json:"rule_name"`
	FailureType string         `json:"failure_type"`
	Service     string         `json:"service"`
	Success     bool           `json:"success"`
	Error       string         `json:"error,omitempty"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
	Duration    time.Duration  `json:"duration"`
	Config      map[string]any `json:"config,omitempty"`
}

// NewChaosInjector 创建混沌注入器
func NewChaosInjector(ruleManager *rules.RuleManager) *ChaosInjector {
	return &ChaosInjector{
		ruleManager:     ruleManager,
		stopChan:        make(chan struct{}),
		activeInjectors: make(map[string]context.CancelFunc),
		executionLog:    make([]ExecutionRecord, 0),
		maxLogSize:      1000,
	}
}

// Start 启动混沌注入器
func (ci *ChaosInjector) Start() error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	if ci.running {
		return fmt.Errorf("chaos injector already running")
	}

	ci.running = true
	go ci.runInjectionLoop()

	logger.Info("Chaos injector started")
	return nil
}

// Stop 停止混沌注入器
func (ci *ChaosInjector) Stop() {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	if !ci.running {
		return
	}

	ci.running = false
	close(ci.stopChan)

	// 取消所有活跃的注入器
	for ruleID, cancel := range ci.activeInjectors {
		cancel()
		logger.Infof("Cancelled active injector for rule: %s", ruleID)
	}
	ci.activeInjectors = make(map[string]context.CancelFunc)

	logger.Info("Chaos injector stopped")
}

// runInjectionLoop 运行注入循环
func (ci *ChaosInjector) runInjectionLoop() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ci.stopChan:
			return
		case <-ticker.C:
			ci.evaluateRules()
		}
	}
}

// evaluateRules 评估规则并执行注入
func (ci *ChaosInjector) evaluateRules() {
	activeRules := ci.ruleManager.GetActiveRules()

	for _, rule := range activeRules {
		// 检查是否应该执行这个规则
		if ci.shouldExecuteRule(rule) {
			go ci.executeRule(rule)
		}
	}
}

// shouldExecuteRule 判断是否应该执行规则
func (ci *ChaosInjector) shouldExecuteRule(rule *rules.ChaosRule) bool {
	// 检查规则是否已在执行
	ci.mu.RLock()
	_, isActive := ci.activeInjectors[rule.ID]
	ci.mu.RUnlock()

	if isActive {
		return false // 规则已在执行
	}

	// 基于故障率判断是否执行
	return rand.Float64() < rule.FailureRate
}

// executeRule 执行规则
func (ci *ChaosInjector) executeRule(rule *rules.ChaosRule) {
	logger.Infof("Executing chaos rule: %s (type: %s)", rule.Name, rule.FailureType)

	// 创建执行上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 添加到活跃注入器列表
	ci.mu.Lock()
	ci.activeInjectors[rule.ID] = cancel
	ci.mu.Unlock()

	// 执行记录
	record := ExecutionRecord{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		FailureType: rule.FailureType,
		Service:     rule.Service,
		StartTime:   time.Now(),
		Config:      rule.Config,
	}

	// 解析持续时间
	duration, err := time.ParseDuration(rule.Duration)
	if err != nil {
		duration = 30 * time.Second // 默认30秒
	}

	// 设置超时
	ctx, timeoutCancel := context.WithTimeout(ctx, duration)
	defer timeoutCancel()

	// 执行具体的故障注入
	err = ci.injectFailure(ctx, rule)

	// 记录执行结果
	record.EndTime = time.Now()
	record.Duration = record.EndTime.Sub(record.StartTime)
	record.Success = (err == nil)
	if err != nil {
		record.Error = err.Error()
		logger.Errorf("Failed to execute chaos rule %s: %v", rule.Name, err)
	} else {
		logger.Infof("Successfully executed chaos rule %s for %v", rule.Name, record.Duration)
	}

	// 添加到执行日志
	ci.addExecutionRecord(record)

	// 标记规则已执行
	ci.ruleManager.MarkRuleExecuted(rule.ID)

	// 从活跃注入器列表中移除
	ci.mu.Lock()
	delete(ci.activeInjectors, rule.ID)
	ci.mu.Unlock()
}

// injectFailure 注入具体的故障
func (ci *ChaosInjector) injectFailure(ctx context.Context, rule *rules.ChaosRule) error {
	switch rule.FailureType {
	case "network_timeout":
		return ci.injectNetworkTimeout(ctx, rule)
	case "memory_leak":
		return ci.injectMemoryLeak(ctx, rule)
	case "cpu_spike":
		return ci.injectCPUSpike(ctx, rule)
	case "database_error":
		return ci.injectDatabaseError(ctx, rule)
	case "disk_full":
		return ci.injectDiskFull(ctx, rule)
	case "slow_response":
		return ci.injectSlowResponse(ctx, rule)
	default:
		return fmt.Errorf("unknown failure type: %s", rule.FailureType)
	}
}

// injectNetworkTimeout 注入网络超时
func (ci *ChaosInjector) injectNetworkTimeout(ctx context.Context, rule *rules.ChaosRule) error {
	// 模拟网络延迟/超时
	timeoutMs := 5000.0
	if val, ok := rule.Config["timeout_ms"]; ok {
		if ms, ok := val.(float64); ok {
			timeoutMs = ms
		}
	}

	delayMs := 1000.0
	if val, ok := rule.Config["delay_ms"]; ok {
		if ms, ok := val.(float64); ok {
			delayMs = ms
		}
	}

	logger.Infof("[CHAOS] Network timeout injection: delay=%vms, timeout=%vms", delayMs, timeoutMs)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(delayMs) * time.Millisecond):
		// 模拟网络延迟
		return nil
	}
}

// injectMemoryLeak 注入内存泄漏
func (ci *ChaosInjector) injectMemoryLeak(ctx context.Context, rule *rules.ChaosRule) error {
	startMemoryMB := 100
	incrementMB := 10
	intervalSec := 30
	maxMemoryMB := 500

	if val, ok := rule.Config["start_memory_mb"]; ok {
		if mb, ok := val.(float64); ok {
			startMemoryMB = int(mb)
		}
	}
	if val, ok := rule.Config["increment_mb"]; ok {
		if mb, ok := val.(float64); ok {
			incrementMB = int(mb)
		}
	}
	if val, ok := rule.Config["interval_sec"]; ok {
		if sec, ok := val.(float64); ok {
			intervalSec = int(sec)
		}
	}
	if val, ok := rule.Config["max_memory_mb"]; ok {
		if mb, ok := val.(float64); ok {
			maxMemoryMB = int(mb)
		}
	}

	logger.Infof("[CHAOS] Memory leak injection: start=%dMB, increment=%dMB, interval=%ds, max=%dMB",
		startMemoryMB, incrementMB, intervalSec, maxMemoryMB)

	// 分配内存块（模拟内存泄漏）
	var memoryBlocks [][]byte
	currentMemory := startMemoryMB

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 清理内存
			memoryBlocks = nil
			runtime.GC()
			return ctx.Err()
		case <-ticker.C:
			if currentMemory < maxMemoryMB {
				// 分配内存块
				block := make([]byte, incrementMB*1024*1024)
				memoryBlocks = append(memoryBlocks, block)
				currentMemory += incrementMB
				logger.Infof("[CHAOS] Allocated %dMB, total: %dMB", incrementMB, currentMemory)
			}
		}
	}
}

// injectCPUSpike 注入CPU峰值
func (ci *ChaosInjector) injectCPUSpike(ctx context.Context, rule *rules.ChaosRule) error {
	cpuPercent := 90.0
	threads := 4

	if val, ok := rule.Config["cpu_percent"]; ok {
		if percent, ok := val.(float64); ok {
			cpuPercent = percent
		}
	}
	if val, ok := rule.Config["threads"]; ok {
		if t, ok := val.(float64); ok {
			threads = int(t)
		}
	}

	logger.Infof("[CHAOS] CPU spike injection: %v%% using %d threads", cpuPercent, threads)

	// 启动CPU密集型任务
	for i := 0; i < threads; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// CPU密集型计算
					for j := 0; j < 1000000; j++ {
						_ = j * j
					}
					// 短暂休息以控制CPU使用率
					time.Sleep(time.Microsecond * time.Duration(1000-cpuPercent*10))
				}
			}
		}()
	}

	<-ctx.Done()
	return ctx.Err()
}

// injectDatabaseError 注入数据库错误
func (ci *ChaosInjector) injectDatabaseError(ctx context.Context, rule *rules.ChaosRule) error {
	errorTypes := []string{"connection_timeout", "deadlock", "connection_refused"}
	if val, ok := rule.Config["error_types"]; ok {
		if types, ok := val.([]any); ok {
			errorTypes = make([]string, len(types))
			for i, t := range types {
				if s, ok := t.(string); ok {
					errorTypes[i] = s
				}
			}
		}
	}

	selectedError := errorTypes[rand.Intn(len(errorTypes))]
	logger.Infof("[CHAOS] Database error injection: %s", selectedError)

	// 这里只是模拟，实际环境中可以:
	// 1. 修改数据库连接配置
	// 2. 阻塞数据库连接
	// 3. 返回模拟错误

	<-ctx.Done()
	return ctx.Err()
}

// injectDiskFull 注入磁盘满错误
func (ci *ChaosInjector) injectDiskFull(ctx context.Context, rule *rules.ChaosRule) error {
	freeSpaceMB := 10
	if val, ok := rule.Config["free_space_mb"]; ok {
		if mb, ok := val.(float64); ok {
			freeSpaceMB = int(mb)
		}
	}

	logger.Infof("[CHAOS] Disk full injection: leaving %dMB free space", freeSpaceMB)

	// 创建临时文件占用磁盘空间
	tempDir := os.TempDir()
	tempFile := fmt.Sprintf("%s/chaos-disk-fill-%d.tmp", tempDir, time.Now().Unix())

	// 这里简化实现，只是创建一个较小的文件
	file, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
		os.Remove(tempFile)
	}()

	// 写入一些数据模拟占用空间
	data := make([]byte, 1024*1024) // 1MB
	for i := 0; i < 10; i++ {       // 写入10MB
		_, err := file.Write(data)
		if err != nil {
			return err
		}
	}

	<-ctx.Done()
	return ctx.Err()
}

// injectSlowResponse 注入响应延迟
func (ci *ChaosInjector) injectSlowResponse(ctx context.Context, rule *rules.ChaosRule) error {
	minDelayMs := 500.0
	maxDelayMs := 3000.0

	if val, ok := rule.Config["min_delay_ms"]; ok {
		if ms, ok := val.(float64); ok {
			minDelayMs = ms
		}
	}
	if val, ok := rule.Config["max_delay_ms"]; ok {
		if ms, ok := val.(float64); ok {
			maxDelayMs = ms
		}
	}

	// 随机延迟
	delay := minDelayMs + rand.Float64()*(maxDelayMs-minDelayMs)
	logger.Infof("[CHAOS] Slow response injection: delay=%vms", delay)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(delay) * time.Millisecond):
		return nil
	}
}

// addExecutionRecord 添加执行记录
func (ci *ChaosInjector) addExecutionRecord(record ExecutionRecord) {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	ci.executionLog = append(ci.executionLog, record)

	// 限制日志大小
	if len(ci.executionLog) > ci.maxLogSize {
		ci.executionLog = ci.executionLog[len(ci.executionLog)-ci.maxLogSize:]
	}
}

// GetExecutionLog 获取执行日志
func (ci *ChaosInjector) GetExecutionLog(limit int) []ExecutionRecord {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	if limit <= 0 || limit > len(ci.executionLog) {
		limit = len(ci.executionLog)
	}

	// 返回最新的记录
	start := len(ci.executionLog) - limit
	if start < 0 {
		start = 0
	}

	return ci.executionLog[start:]
}

// GetStatus 获取注入器状态
func (ci *ChaosInjector) GetStatus() map[string]any {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	return map[string]any{
		"running":          ci.running,
		"active_injectors": len(ci.activeInjectors),
		"total_executions": len(ci.executionLog),
	}
}

// ForceExecuteRule 强制执行指定规则
func (ci *ChaosInjector) ForceExecuteRule(ruleID string) error {
	rule, err := ci.ruleManager.GetRule(ruleID)
	if err != nil {
		return err
	}

	go ci.executeRule(rule)
	return nil
}

// CancelRule 取消正在执行的规则
func (ci *ChaosInjector) CancelRule(ruleID string) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	cancel, exists := ci.activeInjectors[ruleID]
	if !exists {
		return fmt.Errorf("rule %s is not currently executing", ruleID)
	}

	cancel()
	delete(ci.activeInjectors, ruleID)
	logger.Infof("Cancelled execution of rule: %s", ruleID)

	return nil
}

package models

import (
	"time"
)

// ErrorRule 错误注入规则
type ErrorRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Service     string            `json:"service"`     // 目标服务
	Operation   string            `json:"operation"`   // 目标操作
	Conditions  []ErrorCondition  `json:"conditions"`  // 触发条件
	Action      ErrorAction       `json:"action"`      // 错误动作
	Enabled     bool              `json:"enabled"`
	Priority    int               `json:"priority"`    // 规则优先级
	MaxTriggers int               `json:"max_triggers"` // 最大触发次数，0表示无限制
	Triggered   int               `json:"triggered"`   // 已触发次数
	Schedule    *ErrorSchedule    `json:"schedule,omitempty"` // 调度配置
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CreatedBy   string            `json:"created_by"`
}

// ErrorCondition 错误触发条件
type ErrorCondition struct {
	Type     string      `json:"type"`     // 条件类型：probability, header, param, time, etc.
	Operator string      `json:"operator"` // 操作符：eq, ne, gt, lt, contains, etc.
	Field    string      `json:"field"`    // 字段名
	Value    interface{} `json:"value"`    // 期望值
}

// ErrorConditionType 条件类型
const (
	ErrorConditionTypeProbability = "probability" // 概率触发
	ErrorConditionTypeHeader      = "header"      // HTTP 头
	ErrorConditionTypeParam       = "param"       // 请求参数
	ErrorConditionTypeTime        = "time"        // 时间条件
	ErrorConditionTypeUserAgent   = "user_agent"  // User-Agent
	ErrorConditionTypeIP          = "ip"          // IP 地址
	ErrorConditionTypeCount       = "count"       // 请求计数
)

// ErrorAction 错误动作
type ErrorAction struct {
	Type      string                 `json:"type"`      // 动作类型
	Delay     *time.Duration         `json:"delay,omitempty"`     // 延迟时间
	HTTPCode  int                    `json:"http_code,omitempty"` // HTTP 状态码
	Message   string                 `json:"message,omitempty"`   // 错误消息
	Headers   map[string]string      `json:"headers,omitempty"`   // 响应头
	Body      string                 `json:"body,omitempty"`      // 响应体
	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // 额外数据
}

// ErrorActionType 错误动作类型
const (
	ErrorActionTypeHTTPError     = "http_error"     // HTTP 错误响应
	ErrorActionTypeNetworkError  = "network_error"  // 网络错误
	ErrorActionTypeTimeout       = "timeout"        // 超时
	ErrorActionTypeDelay         = "delay"          // 延迟
	ErrorActionTypeCorruption    = "corruption"     // 数据损坏
	ErrorActionTypeDisconnect    = "disconnect"     // 连接断开
	ErrorActionTypeDatabaseError = "database_error" // 数据库错误
	ErrorActionTypeStorageError  = "storage_error"  // 存储错误
)

// ErrorSchedule 错误调度配置
type ErrorSchedule struct {
	StartTime *time.Time `json:"start_time,omitempty"` // 开始时间
	EndTime   *time.Time `json:"end_time,omitempty"`   // 结束时间
	Days      []string   `json:"days,omitempty"`       // 生效日期 (monday, tuesday, etc.)
	Hours     []int      `json:"hours,omitempty"`      // 生效小时 (0-23)
	Timezone  string     `json:"timezone,omitempty"`   // 时区
}

// ErrorStats 错误统计
type ErrorStats struct {
	TotalRules        int                    `json:"total_rules"`
	ActiveRules       int                    `json:"active_rules"`
	TotalTriggers     int64                  `json:"total_triggers"`
	TriggersLastHour  int64                  `json:"triggers_last_hour"`
	TriggersToday     int64                  `json:"triggers_today"`
	RuleStats         map[string]*RuleStat   `json:"rule_stats"`
	ServiceStats      map[string]*ServiceStat `json:"service_stats"`
	ErrorTypeStats    map[string]int64       `json:"error_type_stats"`
	LastReset         time.Time              `json:"last_reset"`
	LastUpdate        time.Time              `json:"last_update"`
}

// RuleStat 规则统计
type RuleStat struct {
	RuleID       string    `json:"rule_id"`
	RuleName     string    `json:"rule_name"`
	TotalTriggers int64     `json:"total_triggers"`
	LastTriggered time.Time `json:"last_triggered"`
	ErrorCounts   map[string]int64 `json:"error_counts"` // error_type -> count
}

// ServiceStat 服务统计
type ServiceStat struct {
	ServiceName   string               `json:"service_name"`
	TotalRequests int64                `json:"total_requests"`
	ErrorRequests int64                `json:"error_requests"`
	ErrorRate     float64              `json:"error_rate"`
	OperationStats map[string]*OpStat  `json:"operation_stats"`
}

// OpStat 操作统计
type OpStat struct {
	OperationName string  `json:"operation_name"`
	TotalRequests int64   `json:"total_requests"`
	ErrorRequests int64   `json:"error_requests"`
	ErrorRate     float64 `json:"error_rate"`
}

// ErrorEvent 错误事件（用于记录和分析）
type ErrorEvent struct {
	ID         string                 `json:"id"`
	RuleID     string                 `json:"rule_id"`
	RuleName   string                 `json:"rule_name"`
	Service    string                 `json:"service"`
	Operation  string                 `json:"operation"`
	Action     ErrorAction            `json:"action"`
	RequestID  string                 `json:"request_id,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	RemoteAddr string                 `json:"remote_addr,omitempty"`
	Headers    map[string]string      `json:"headers,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Success    bool                   `json:"success"` // 是否成功注入错误
	Error      string                 `json:"error,omitempty"`
}
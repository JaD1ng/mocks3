# Mock Error Service

Mock Error Service是MockS3微服务架构中的错误注入服务，专门用于混沌工程测试，通过可配置的错误注入来测试系统的容错性和韧性。

## 功能特性

### 🎯 **错误注入类型**
- **HTTP错误**: 返回指定的HTTP错误状态码
- **网络错误**: 模拟网络超时、连接断开等
- **延迟注入**: 为请求添加人工延迟
- **数据库错误**: 模拟数据库操作失败
- **存储错误**: 模拟文件存储操作失败

### 📋 **灵活的规则引擎**
- **多条件支持**: 概率、请求头、参数、时间、IP等
- **优先级调度**: 支持规则优先级排序
- **时间调度**: 支持按时间段和日期调度
- **触发次数限制**: 支持最大触发次数控制

### 📊 **完整的统计监控**
- **实时统计**: 错误注入次数、成功率等
- **规则统计**: 每个规则的触发情况
- **服务统计**: 各服务的错误率分析
- **事件记录**: 详细的错误注入事件日志

### 🔧 **便捷的管理接口**
- **规则管理**: 增删改查错误注入规则
- **动态控制**: 实时启用/禁用规则
- **统计查询**: 获取详细的统计信息
- **事件追踪**: 查看错误注入历史

## API接口

### 规则管理
```
POST   /api/v1/rules           # 添加错误规则
GET    /api/v1/rules/:id       # 获取规则详情
PUT    /api/v1/rules/:id       # 更新错误规则
DELETE /api/v1/rules/:id       # 删除错误规则
GET    /api/v1/rules           # 列出所有规则
```

### 规则控制
```
POST   /api/v1/rules/:id/enable    # 启用规则
POST   /api/v1/rules/:id/disable   # 禁用规则
```

### 错误注入
```
POST   /api/v1/inject/:service/:operation  # 检查是否注入错误
```

### 统计监控
```
GET    /api/v1/stats           # 获取统计信息
POST   /api/v1/stats/reset     # 重置统计信息
GET    /api/v1/events          # 获取错误事件
GET    /health                 # 健康检查
```

## 配置说明

### 环境变量
- `SERVER_PORT`: 服务端口 (默认: 8085)
- `ERROR_MAX_RULES`: 最大规则数量 (默认: 1000)
- `ERROR_ENABLE_SCHEDULING`: 启用时间调度 (默认: true)
- `ERROR_DEFAULT_PROBABILITY`: 默认触发概率 (默认: 0.1)
- `ERROR_ENABLE_STATISTICS`: 启用统计 (默认: true)
- `INJECTION_GLOBAL_PROBABILITY`: 全局触发概率 (默认: 1.0)
- `INJECTION_MAX_DELAY_MS`: 最大延迟毫秒数 (默认: 10000)

### 错误类型配置
```bash
# 启用各种错误类型
INJECTION_ENABLE_HTTP_ERRORS=true
INJECTION_ENABLE_NETWORK_ERRORS=true
INJECTION_ENABLE_DATABASE_ERRORS=true
INJECTION_ENABLE_STORAGE_ERRORS=true
```

## 使用示例

### 添加简单的随机错误规则
```bash
curl -X POST http://localhost:8085/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Storage Random Error",
    "description": "10% chance of 500 error in storage service",
    "service": "storage-service",
    "enabled": true,
    "priority": 1,
    "conditions": [
      {
        "type": "probability",
        "operator": "eq",
        "value": 0.1
      }
    ],
    "action": {
      "type": "http_error",
      "http_code": 500,
      "message": "Internal server error injected"
    }
  }'
```

### 添加延迟注入规则
```bash
curl -X POST http://localhost:8085/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Metadata Service Delay",
    "description": "Add 2s delay to metadata operations",
    "service": "metadata-service",
    "operation": "GetMetadata",
    "enabled": true,
    "conditions": [
      {
        "type": "probability",
        "value": 0.2
      }
    ],
    "action": {
      "type": "delay",
      "delay": "2s"
    }
  }'
```

### 添加条件性错误规则
```bash
curl -X POST http://localhost:8085/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "User Agent Based Error",
    "description": "Inject errors for specific user agents",
    "enabled": true,
    "conditions": [
      {
        "type": "header",
        "field": "User-Agent",
        "operator": "contains",
        "value": "test-client"
      },
      {
        "type": "probability",
        "value": 0.5
      }
    ],
    "action": {
      "type": "http_error",
      "http_code": 503,
      "message": "Service unavailable for test clients"
    }
  }'
```

### 检查错误注入
```bash
curl -X POST http://localhost:8085/api/v1/inject/storage-service/WriteObject \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "user_agent": "test-client",
      "remote_addr": "192.168.1.100"
    }
  }'
```

### 获取统计信息
```bash
curl http://localhost:8085/api/v1/stats
```

## 条件类型详解

### 1. 概率条件 (probability)
```json
{
  "type": "probability",
  "operator": "eq",
  "value": 0.1
}
```

### 2. 请求头条件 (header)
```json
{
  "type": "header",
  "field": "User-Agent",
  "operator": "contains",
  "value": "Chrome"
}
```

### 3. 参数条件 (param)
```json
{
  "type": "param",
  "field": "bucket",
  "operator": "eq",
  "value": "test-bucket"
}
```

### 4. 时间条件 (time)
```json
{
  "type": "time",
  "operator": "gt",
  "value": "2024-01-01T00:00:00Z"
}
```

### 5. IP地址条件 (ip)
```json
{
  "type": "ip",
  "operator": "eq",
  "value": "192.168.1.0/24"
}
```

## 支持的操作符

- `eq`: 等于
- `ne`: 不等于
- `gt`: 大于
- `lt`: 小于
- `gte`: 大于等于
- `lte`: 小于等于
- `contains`: 包含
- `not_contains`: 不包含
- `starts_with`: 以...开始
- `ends_with`: 以...结束
- `regex`: 正则表达式匹配

## 错误动作类型

### HTTP错误
```json
{
  "type": "http_error",
  "http_code": 500,
  "message": "Internal server error",
  "headers": {
    "X-Error-Injected": "true"
  }
}
```

### 延迟注入
```json
{
  "type": "delay",
  "delay": "2s"
}
```

### 网络错误
```json
{
  "type": "network_error",
  "message": "Connection timeout"
}
```

### 数据库错误
```json
{
  "type": "database_error",
  "message": "Database connection failed"
}
```

### 存储错误
```json
{
  "type": "storage_error",
  "message": "Disk full"
}
```

## 时间调度

支持按时间段和日期调度错误注入：

```json
{
  "schedule": {
    "start_time": "2024-01-01T09:00:00Z",
    "end_time": "2024-01-01T17:00:00Z",
    "days": ["monday", "tuesday", "wednesday", "thursday", "friday"],
    "hours": [9, 10, 11, 14, 15, 16],
    "timezone": "Asia/Shanghai"
  }
}
```

## 运行方式

### 直接运行
```bash
cd services/mock-error
go run cmd/server/main.go
```

### Docker运行
```bash
cd services/mock-error
docker-compose up -d
```

## 集成到其他服务

其他服务可以通过HTTP API查询是否需要注入错误：

```go
// 在服务中集成错误注入检查
func (s *Service) SomeOperation(ctx context.Context) error {
    // 检查是否需要注入错误
    resp, err := http.Post("http://mock-error-service:8085/api/v1/inject/my-service/SomeOperation", 
        "application/json", 
        strings.NewReader(`{"metadata":{}}`))
    
    if err == nil && resp.StatusCode == 200 {
        var result map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&result)
        
        if shouldInject, ok := result["should_inject"].(bool); ok && shouldInject {
            if action, ok := result["action"].(map[string]interface{}); ok {
                return handleErrorInjection(action)
            }
        }
    }
    
    // 正常业务逻辑
    return s.normalOperation(ctx)
}
```

## 混沌工程测试场景

### 1. 服务可用性测试
- 随机返回500错误测试服务降级
- 模拟服务超时测试重试机制
- 模拟网络分区测试故障转移

### 2. 性能测试
- 注入延迟测试系统响应
- 模拟高负载下的错误率
- 测试缓存失效场景

### 3. 数据一致性测试
- 模拟存储失败测试回滚
- 模拟网络抖动测试重复请求
- 测试分布式事务处理

## 目录结构
```
services/mock-error/
├── cmd/server/           # 应用入口
├── internal/
│   ├── config/          # 配置管理
│   ├── handler/         # HTTP处理器
│   ├── service/         # 错误注入服务和规则引擎
│   └── repository/      # 规则存储和统计数据
├── Dockerfile           # Docker构建
└── docker-compose.yml   # 本地运行配置
```
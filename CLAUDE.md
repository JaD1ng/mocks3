# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

MockS3 是基于原 s3-storage 单体应用拆分而来的微服务架构，提供 S3 兼容的对象存储服务，并具备全面的错误模拟能力。主要用于测试分布式系统的可靠性和容错性。

## 技术栈

- **容器化**: Docker
- **服务通信**: HTTP REST API
- **服务发现**: Consul
- **API网关**: Nginx
- **数据库**: PostgreSQL + Redis缓存  
- **配置管理**: Consul KV
- **消息队列**: Redis streams
- **监控**: Prometheus
- **可观测性**: OpenTelemetry（统一日志、指标、链路追踪）

## 微服务架构

### 服务拆分设计

1. **Nginx Gateway** (端口: 8080)
   - S3 协议适配和路由
   - 负载均衡和反向代理
   - 认证、授权、限流

2. **Metadata Service** (端口: 8081)
   - 对象元数据 CRUD 操作
   - PostgreSQL 数据库管理
   - 搜索和统计功能

3. **Storage Service** (端口: 8082)
   - 文件实际存储操作
   - 多存储节点管理
   - 数据冗余和恢复

4. **Queue Service** (端口: 8083)
   - 异步任务管理
   - 基于 Redis 的队列实现
   - 工作节点调度

5. **Third-Party Service** (端口: 8084)
   - 外部数据源集成
   - 数据获取和缓存
   - 回退机制实现

6. **Mock Error Service** (端口: 8085)
   - 可配置的错误注入
   - 故障模拟和统计
   - 测试场景管理

### 关键数据流

**上传流程**:
1. Nginx Gateway 接收 S3 PUT 请求
2. 路由到 Storage Service 存储文件数据
3. Storage Service 调用 Metadata Service 保存元数据

**下载流程**:
1. Nginx Gateway 接收 S3 GET 请求
2. 路由到 Storage Service 处理请求
3. Storage Service 查询 Metadata Service 获取文件信息
4. 如失败，通过 Third-Party Service 获取

**错误注入**:
- Mock Error Service 通过中间件在各服务中注入故障
- 支持网络、存储、数据库、业务逻辑等各层面错误
- 可通过配置文件或 HTTP API 动态控制

### 服务间通信

- **同步通信**: HTTP REST API（JSON 格式）
- **异步通信**: Redis Streams 用于事件通知
- **管理接口**: HTTP REST 用于配置和监控
- **服务发现**: Consul Agent 自动注册和发现
- **负载均衡**: Nginx upstream 配置

## 配置管理

### 环境变量
每个服务支持以下环境变量:
- `SERVICE_PORT`: 服务端口
- `SERVICE_NAME`: 服务名称（用于 Consul 注册）
- `LOG_LEVEL`: 日志级别 (debug/info/warn/error)
- `CONSUL_ADDR`: Consul 地址 (默认: localhost:8500)
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTEL Collector 端点

### Consul KV 配置
所有服务配置存储在 Consul KV 中，支持动态更新:

## 接口化设计与目录结构

```
mocks3/
├── shared/                        # 共享包
│   ├── interfaces/                # 核心接口定义
│   │   ├── storage.go            # 存储服务接口
│   │   ├── metadata.go           # 元数据服务接口
│   │   ├── queue.go              # 队列服务接口
│   │   ├── error_injector.go     # 错误注入接口
│   │   └── gateway.go            # 网关接口
│   ├── models/                    # 共享数据模型
│   │   ├── object.go             # 对象模型
│   │   ├── metadata.go           # 元数据模型
│   │   ├── task.go               # 任务模型
│   │   └── error.go              # 错误模型
│   ├── client/                    # HTTP 客户端
│   │   ├── storage_client.go     # 存储服务客户端
│   │   ├── metadata_client.go    # 元数据服务客户端
│   │   └── queue_client.go       # 队列服务客户端
│   ├── observability/             # 可观测性组件
│   │   ├── metric/               # 指标收集
│   │   │   ├── collector.go      # 指标收集器
│   │   │   ├── registry.go       # 指标注册
│   │   │   └── middleware.go     # HTTP 指标中间件
│   │   ├── log/                  # 日志处理
│   │   │   ├── logger.go         # 结构化日志器
│   │   │   ├── formatter.go      # 日志格式化
│   │   │   └── context.go        # 日志上下文
│   │   └── trace/                # 链路追踪
│   │       ├── tracer.go         # 追踪器初始化
│   │       ├── span.go           # Span 工具
│   │       └── middleware.go     # HTTP 追踪中间件
│   ├── middleware/                # 其他中间件
│   │   ├── error_injection.go    # 错误注入中间件
│   │   ├── consul.go             # Consul 集成
│   │   └── recovery.go           # 异常恢复
│   └── utils/                     # 工具函数
│       ├── config.go             # 配置加载
│       ├── http.go               # HTTP 工具
│       └── retry.go              # 重试工具
├── services/                      # 微服务实现
│   ├── metadata/                  # 元数据服务
│   │   ├── cmd/server/main.go    # 服务入口
│   │   ├── internal/
│   │   │   ├── handler/          # HTTP 处理器
│   │   │   ├── service/          # 业务逻辑实现
│   │   │   ├── repository/       # 数据访问层
│   │   │   └── config/           # 服务配置
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   ├── storage/                   # 存储服务
│   ├── queue/                     # 队列服务
│   ├── third-party/              # 第三方服务
│   └── mock-error/               # 错误注入服务
├── gateway/                       # Nginx 网关配置
│   ├── nginx.conf
│   ├── templates/
│   └── Dockerfile
├── deployments/                   # 部署配置
│   ├── docker-compose.yml        # 完整堆栈
│   ├── consul/
│   ├── elasticsearch/
│   └── prometheus/
└── docs/                         # 文档
    ├── api/                      # API 文档
    └── architecture/             # 架构文档
```

## 开发注意事项

### 错误处理
- 所有服务间调用都应实现重试机制（指数退避）
- 使用熔断器模式防止雪崩效应
- 区分可重试错误和不可重试错误

### 事务处理
- 使用 Saga 模式处理分布式事务
- 每个操作步骤实现对应的补偿操作
- 状态机管理复杂业务流程

### 可观测性
- **OpenTelemetry 统一可观测性**: 一套 SDK 处理所有遥测数据
- **结构化日志**: 使用 OTEL Logs API，自动包含 trace/span context
- **分布式追踪**: 自动生成 trace ID 和 span，跨服务传递 context
- **指标收集**: 使用 OTEL Metrics API 收集业务和系统指标
- **统一导出**: 通过 OTEL Collector 统一处理和路由遥测数据

### 测试策略
- 单元测试覆盖核心业务逻辑
- 集成测试验证服务间协作
- 契约测试确保 API 兼容性
- 混沌工程测试系统韧性

### 容器化部署
- 每个服务独立的 Dockerfile
- 多阶段构建优化镜像大小
- 健康检查端点配置
- 资源限制和环境变量管理

## 关键技术实现

### Consul 集成
每个服务启动时需要:
- 注册到 Consul（服务名、地址、端口、健康检查）
- 从 Consul KV 加载配置
- 监听配置变更并动态更新
- 通过 Consul DNS 或 HTTP API 发现其他服务

### Nginx 网关配置
- upstream 配置从 Consul 动态发现后端服务
- location 规则匹配 S3 API 路径
- 健康检查和故障转移
- 访问日志和错误日志

### OpenTelemetry 集成
每个服务统一使用 OpenTelemetry SDK:
- **核心包**: `go.opentelemetry.io/otel`
- **日志**: `go.opentelemetry.io/otel/log` - 结构化日志，自动关联 trace context
- **追踪**: `go.opentelemetry.io/otel/trace` - 分布式链路追踪
- **指标**: `go.opentelemetry.io/otel/metric` - 业务和系统指标
- **导出**: 统一发送到 OTEL Collector，再路由到各种后端存储
- **中间件**: HTTP/gRPC 中间件自动生成 spans 和记录指标

### OpenTelemetry 数据流
```
应用服务 → OTEL SDK → OTEL Collector → 后端存储
                                    ├── Elasticsearch (traces + logs)
                                    └── Prometheus (metrics)
```

### ES 索引设计
```yaml
# Traces 索引模板
traces-*:
  mappings:
    properties:
      trace_id: keyword
      span_id: keyword  
      parent_span_id: keyword
      service_name: keyword
      operation_name: keyword
      start_time: date
      duration: long
      tags: object
      
# Logs 索引模板  
logs-*:
  mappings:
    properties:
      timestamp: date
      level: keyword
      message: text
      trace_id: keyword  # 关联字段
      span_id: keyword   # 关联字段
      service_name: keyword
      attributes: object
```

### 关联查询示例
```json
# 根据 trace_id 查找所有相关日志和 spans
GET logs-*,traces-*/_search
{
  "query": {
    "term": { "trace_id": "abc123..." }
  }
}
```

### 错误注入中间件
Mock Error Service 通过 HTTP 拦截器实现:
- 基于 Consul 配置的故障注入
- 支持延迟、错误码、连接断开等故障类型
- 实时配置更新
- 通过 OTEL 记录故障注入事件和统计
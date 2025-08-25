# MockS3 配置管理

本目录包含 MockS3 微服务架构的所有配置文件，采用统一的 YAML 格式，支持多环境部署和动态配置管理。

## 目录结构

```
config/
├── README.md                   # 配置文档
├── global.yaml                 # 全局配置
├── docker-compose.yaml         # Docker Compose 配置
├── services/                   # 微服务配置
│   ├── metadata.yaml          # 元数据服务配置
│   ├── storage.yaml           # 存储服务配置
│   ├── queue.yaml             # 队列服务配置
│   ├── third-party.yaml       # 第三方服务配置
│   └── mock-error.yaml        # 错误注入服务配置
└── env/                       # 环境配置
    ├── development.yaml       # 开发环境
    └── production.yaml        # 生产环境
```

## 配置文件说明

### 全局配置 (global.yaml)

包含所有服务共享的全局配置：
- 环境信息和集群配置
- 网络和服务发现配置  
- 可观测性全局配置
- 安全和性能配置
- 数据存储配置

### 服务配置 (services/*.yaml)

每个微服务的专用配置文件，包含：
- **服务基本配置**: 端口、版本、环境等
- **可观测性配置**: OTEL、日志、指标、追踪
- **依赖服务配置**: 其他微服务的连接信息
- **服务特定配置**: 各服务独有的业务配置
- **性能和健康检查配置**: 超时、重试、健康检查等

#### Metadata Service (metadata.yaml)
- 数据库连接配置
- 缓存配置
- 对象元数据管理配置

#### Storage Service (storage.yaml) 
- 存储节点配置
- 文件存储策略
- 多节点冗余配置
- 与 Metadata 和 Third-Party 服务的集成

#### Queue Service (queue.yaml)
- Redis 连接配置
- 队列处理配置
- 任务类型和优先级配置
- 工作节点调度配置

#### Third-Party Service (third-party.yaml)
- 外部数据源配置 (S3, HTTP, FTP)
- 缓存策略配置
- 熔断器和限流配置
- 回退机制配置

#### Mock Error Service (mock-error.yaml)
- 错误注入规则配置
- 故障场景配置
- 统计和监控配置
- 动态规则管理配置

### 环境配置 (env/*.yaml)

不同部署环境的特定配置：

#### Development (development.yaml)
- 本地开发环境设置
- 调试功能启用
- 简化的安全配置
- 详细的日志级别

#### Production (production.yaml)
- 生产环境优化配置
- 环境变量占位符
- 安全强化配置
- 性能优化设置

## 配置管理

### 使用配置管理脚本

项目提供了配置管理脚本 `scripts/config-manager.sh`，支持：

```bash
# 验证所有配置文件
./scripts/config-manager.sh validate

# 生成配置模板
./scripts/config-manager.sh generate

# 部署配置到 Consul KV
./scripts/config-manager.sh deploy -e development

# 备份当前配置
./scripts/config-manager.sh backup

# 恢复配置
./scripts/config-manager.sh restore ./backups/config-20231201-120000

# 比较环境配置差异
./scripts/config-manager.sh diff development production

# 列出所有配置文件
./scripts/config-manager.sh list
```

### Consul KV 集成

配置支持通过 Consul KV 进行动态管理：

- **配置路径规则**: `mocks3/config/{type}/{name}`
- **全局配置**: `mocks3/config/global`
- **环境配置**: `mocks3/config/env/{environment}`  
- **服务配置**: `mocks3/config/services/{service}`

### 环境变量支持

生产环境配置支持环境变量占位符：
- `${VAR_NAME}` - 必需的环境变量
- `${VAR_NAME:default}` - 带默认值的环境变量

## 配置最佳实践

### 1. 配置分层

- **全局配置**: 所有服务共享的基础配置
- **环境配置**: 特定环境的配置覆盖
- **服务配置**: 服务特定的业务配置
- **运行时配置**: 通过环境变量动态设置

### 2. 敏感信息处理

- 开发环境可使用明文配置
- 生产环境必须使用环境变量
- 密钥和证书通过 Consul KV 或 Vault 管理
- 配置文件中使用占位符而非明文

### 3. 配置验证

- 启动前验证配置文件语法
- 验证必需字段和数据类型
- 检查服务依赖和连接配置
- 使用配置管理脚本自动验证

### 4. 版本控制

- 配置文件纳入版本控制
- 配置变更通过 PR 流程
- 标记重大配置变更
- 保留配置变更历史

### 5. 监控和告警

- 监控配置加载状态
- 配置验证失败告警
- 服务启动配置错误告警
- 配置热更新状态跟踪

## Docker Compose 集成

`config/docker-compose.yaml` 提供了完整的容器化部署配置：

- 所有微服务和依赖基础设施
- 统一的网络和数据卷管理
- 健康检查和依赖关系配置
- 可观测性堆栈集成

启动完整环境：
```bash
cd config
docker-compose up -d
```

## OpenTelemetry 配置

所有服务统一使用 OpenTelemetry SDK：

- **统一端点**: OTEL Collector (localhost:4318)
- **数据流**: 服务 → OTEL Collector → 后端存储
- **后端存储**: Elasticsearch (traces/logs) + Prometheus (metrics)
- **可视化**: Grafana Dashboard

配置特点：
- 自动 trace context 传播
- 结构化日志与 trace 关联
- 统一的指标命名规范
- 可配置的采样率

## 故障排查

### 常见问题

1. **配置文件语法错误**
   - 使用 `yq` 验证 YAML 语法
   - 检查缩进和特殊字符

2. **服务无法连接**
   - 检查服务端口配置
   - 验证网络配置
   - 查看服务发现状态

3. **环境变量未设置**
   - 检查必需的环境变量
   - 验证占位符语法
   - 确认默认值设置

4. **Consul KV 同步问题**
   - 检查 Consul 连接状态
   - 验证 KV 路径和权限
   - 查看配置更新日志

### 调试方法

- 启用详细日志 (`log_level: debug`)
- 使用健康检查端点验证服务状态
- 通过 Consul UI 查看服务注册状态
- 使用配置管理脚本验证配置
# MockS3 微服务架构

MockS3 是一个基于微服务架构的 S3 兼容对象存储系统，专门用于测试分布式系统的可靠性和容错性。它提供完整的对象存储功能，并集成了强大的错误注入能力，是进行混沌工程测试的理想平台。

## 🏗️ 架构概览

MockS3 采用现代微服务架构，由以下组件组成：

### 核心服务
- **🌐 Nginx Gateway** (8080) - S3 协议适配和 API 网关
- **📊 Metadata Service** (8081) - 对象元数据管理
- **💾 Storage Service** (8082) - 文件存储和管理
- **📨 Queue Service** (8083) - 异步任务处理
- **🔗 Third-Party Service** (8084) - 外部数据源集成
- **⚡ Mock Error Service** (8085) - 错误注入和混沌工程

### 基础设施
- **🗄️ PostgreSQL** - 元数据持久化存储
- **🔄 Redis** - 缓存和消息队列
- **🎯 Consul** - 服务发现和配置管理

### 监控栈
- **📈 OpenTelemetry Collector** - 统一遥测数据收集
- **🔍 Elasticsearch** - 日志和链路追踪存储
- **📊 Prometheus** - 指标收集和存储
- **📋 Grafana** - 可视化仪表板
- **🔎 Kibana** - 日志分析界面

## 🚀 快速开始

### 前置要求
- Docker 20.10+
- Docker Compose 2.0+
- Make (可选，用于便捷命令)

### 启动完整系统

```bash
# 克隆项目
git clone <repository-url>
cd mocks3

# 启动所有服务
docker-compose up -d

# 或者使用 Make 命令 (推荐)
make up
```

### 验证部署

```bash
# 检查服务状态
make status

# 执行健康检查
make health-check

# 测试 S3 API
make test-api
```

## 📋 服务端点

| 服务 | 端口 | 用途 | 健康检查 |
|------|------|------|----------|
| S3 API Gateway | 8080 | S3 兼容 API | http://localhost:8080/health |
| Metadata Service | 8081 | 元数据管理 | http://localhost:8081/health |
| Storage Service | 8082 | 文件存储 | http://localhost:8082/health |
| Queue Service | 8083 | 任务队列 | http://localhost:8083/health |
| Third-Party Service | 8084 | 外部集成 | http://localhost:8084/health |
| Mock Error Service | 8085 | 错误注入 | http://localhost:8085/health |
| Consul UI | 8500 | 服务发现 | http://localhost:8500 |
| Grafana | 3000 | 监控面板 | http://localhost:3000 (admin/admin) |
| Prometheus | 9090 | 指标查询 | http://localhost:9090 |
| Kibana | 5601 | 日志分析 | http://localhost:5601 |
| Elasticsearch | 9200 | 搜索引擎 | http://localhost:9200 |

## 🧪 S3 API 使用示例

### 基本操作

```bash
# 创建存储桶
curl -X PUT http://localhost:8080/test-bucket/

# 上传文件
curl -X PUT http://localhost:8080/test-bucket/test.txt \
  -H "Content-Type: text/plain" \
  -d "Hello MockS3!"

# 下载文件
curl http://localhost:8080/test-bucket/test.txt

# 列出对象
curl http://localhost:8080/test-bucket/

# 获取对象元数据
curl -I http://localhost:8080/test-bucket/test.txt

# 删除对象
curl -X DELETE http://localhost:8080/test-bucket/test.txt
```

### 使用 AWS CLI

```bash
# 配置 AWS CLI (使用假凭证)
aws configure set aws_access_key_id test
aws configure set aws_secret_access_key test
aws configure set default.region us-east-1
aws configure set default.output json

# 设置端点
export AWS_ENDPOINT_URL=http://localhost:8080

# S3 操作
aws s3 mb s3://my-bucket
aws s3 cp file.txt s3://my-bucket/
aws s3 ls s3://my-bucket/
aws s3 rm s3://my-bucket/file.txt
```

## 🎭 错误注入和混沌工程

MockS3 内置强大的错误注入功能，支持各种故障模拟：

### 添加错误注入规则

```bash
# 添加随机 HTTP 错误
curl -X POST http://localhost:8085/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Storage Random Error",
    "description": "10% chance of 500 error",
    "service": "storage-service",
    "enabled": true,
    "conditions": [
      {
        "type": "probability",
        "value": 0.1
      }
    ],
    "action": {
      "type": "http_error",
      "http_code": 500,
      "message": "Internal server error injected"
    }
  }'

# 添加延迟注入
curl -X POST http://localhost:8085/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Metadata Delay",
    "description": "Add 2s delay to metadata operations",
    "service": "metadata-service",
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

### 查看错误注入统计

```bash
# 获取所有规则
curl http://localhost:8085/api/v1/rules

# 获取统计信息
curl http://localhost:8085/api/v1/stats

# 查看错误事件
curl http://localhost:8085/api/v1/events
```

## 📊 监控和可观测性

### Grafana 仪表板

访问 http://localhost:3000 (admin/admin) 查看预配置的仪表板：

- **MockS3 Overview** - 系统总览
- **Service Metrics** - 各服务指标
- **Infrastructure** - 基础设施监控
- **Error Injection** - 错误注入统计
- **Business Metrics** - 业务指标

### Prometheus 指标

```bash
# 查看可用指标
curl http://localhost:9090/api/v1/label/__name__/values

# 查询示例
curl 'http://localhost:9090/api/v1/query?query=http_requests_total'
```

### 日志查看

```bash
# 查看所有服务日志
make logs

# 查看特定服务日志
make logs-metadata
make logs-storage

# 在 Kibana 中查看结构化日志
# 访问 http://localhost:5601
```

## 🔧 开发指南

### 本地开发环境

```bash
# 安装开发工具
make install-tools

# 设置开发环境
make dev-setup

# 运行测试
make test

# 代码格式化和检查
make fmt
make lint

# 启动基础设施 (用于本地开发)
make up-infra

# 本地运行单个服务
make dev-metadata
make dev-storage
```

### 构建和测试

```bash
# 构建所有服务
make build-all

# 运行集成测试
make test-integration

# 运行性能测试
make benchmark

# 运行负载测试
make load-test
```

### 代码结构

```
mocks3/
├── shared/                 # 共享包
│   ├── interfaces/        # 服务接口定义
│   ├── models/           # 数据模型
│   ├── client/           # HTTP 客户端
│   ├── observability/    # 可观测性组件
│   ├── middleware/       # 中间件
│   └── utils/           # 工具函数
├── services/             # 微服务实现
│   ├── metadata/        # 元数据服务
│   ├── storage/         # 存储服务
│   ├── queue/           # 队列服务
│   ├── third-party/     # 第三方服务
│   └── mock-error/      # 错误注入服务
├── gateway/              # Nginx 网关
├── deployments/          # 部署配置
└── docs/                # 文档
```

## 🚀 部署选项

### Docker Compose (推荐用于开发和测试)

```bash
# 完整部署
docker-compose up -d

# 仅基础设施
make up-infra

# 仅微服务
make up-services
```

### Kubernetes (生产环境)

```bash
# TODO: 添加 Kubernetes 部署文件
# kubectl apply -f deployments/k8s/
```

### 云原生部署

支持部署到：
- AWS EKS
- Google GKE  
- Azure AKS
- 阿里云 ACK

## 🔒 安全考虑

### 认证和授权
- 当前版本使用简化的认证机制
- 生产环境需要集成真实的身份认证系统
- 支持 IAM 策略和 S3 兼容的访问控制

### 网络安全
- 所有服务间通信通过内部网络
- Nginx 网关提供统一入口点
- 支持 TLS 终端和证书管理

### 数据安全
- 数据库连接加密
- 敏感配置使用 Consul KV 加密存储
- 支持对象存储加密

## 📈 性能调优

### 容量规划
- **存储**: 支持 PB 级别对象存储
- **并发**: 支持数千并发连接
- **吞吐**: 优化的多节点存储架构

### 优化建议
```bash
# 数据库连接池调优
DATABASE_MAX_OPEN_CONNS=50
DATABASE_MAX_IDLE_CONNS=10

# Redis 内存配置
REDIS_MAXMEMORY=512mb
REDIS_MAXMEMORY_POLICY=allkeys-lru

# Nginx 工作进程
NGINX_WORKER_PROCESSES=auto
NGINX_WORKER_CONNECTIONS=1024
```

## 🛠️ 故障排除

### 常见问题

1. **服务启动失败**
   ```bash
   # 查看服务日志
   make logs-<service-name>
   
   # 检查依赖服务状态
   make status
   ```

2. **数据库连接问题**
   ```bash
   # 检查 PostgreSQL 状态
   docker-compose exec postgres pg_isready -U mocks3
   
   # 重置数据库
   make reset-data
   ```

3. **Consul 服务发现问题**
   ```bash
   # 查看注册的服务
   make consul-services
   
   # 重启 Consul
   docker-compose restart consul
   ```

### 日志级别配置

```bash
# 设置详细日志
LOG_LEVEL=debug docker-compose up

# 查看特定组件日志
docker-compose logs -f metadata-service
```

## 🤝 贡献指南

### 开发流程
1. Fork 项目
2. 创建功能分支
3. 编写代码和测试
4. 运行质量检查: `make pre-commit`
5. 提交 Pull Request

### 代码规范
- 遵循 Go 标准格式
- 100% 测试覆盖率
- 完整的错误处理
- 结构化日志记录

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🔗 相关链接

- [架构文档](docs/architecture/)
- [API 文档](docs/api/)
- [部署指南](docs/deployment/)
- [故障排除](docs/troubleshooting/)

---

**MockS3** - 为混沌工程而生的 S3 兼容存储系统 🚀
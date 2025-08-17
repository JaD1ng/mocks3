# Queue Service

队列服务是MockS3微服务架构中负责异步任务管理和处理的核心服务，基于Redis Streams实现。

## 功能特性

### 🚀 异步任务管理
- **任务队列**: 基于Redis Streams的可靠消息队列
- **工作节点**: 多工作节点并发处理任务
- **重试机制**: 支持失败任务的自动重试
- **优先级**: 任务优先级调度

### 📊 任务类型
- **文件删除**: `file_deletion` - 异步删除存储文件
- **元数据清理**: `metadata_cleanup` - 清理无效元数据
- **存储优化**: `storage_optimization` - 存储空间优化

### 🔧 管理功能
- **工作节点管理**: 动态启动/停止工作节点
- **任务状态跟踪**: 实时任务状态监控
- **统计信息**: 队列性能和状态统计
- **健康检查**: 服务和Redis连接状态检查

### 🛡️ 可靠性保证
- **消费者组**: Redis消费者组确保任务不丢失
- **故障恢复**: 工作节点故障时任务自动重新分配
- **持久化**: 任务数据持久化存储

## API接口

### 任务管理
```
POST   /api/v1/tasks              # 添加任务
GET    /api/v1/tasks/:id          # 获取任务信息
GET    /api/v1/tasks?status=pending&limit=100  # 列出任务
```

### 工作节点管理
```
POST   /api/v1/workers/:id/start  # 启动工作节点
POST   /api/v1/workers/:id/stop   # 停止工作节点
```

### 监控接口
```
GET    /api/v1/stats              # 获取队列统计信息
GET    /health                    # 健康检查
```

## 配置说明

### 环境变量
- `SERVER_PORT`: 服务端口 (默认: 8083)
- `REDIS_HOST`: Redis主机 (默认: localhost)
- `REDIS_PORT`: Redis端口 (默认: 6379)
- `REDIS_PASSWORD`: Redis密码
- `REDIS_DB`: Redis数据库 (默认: 0)
- `QUEUE_MAX_WORKERS`: 最大工作节点数 (默认: 3)
- `QUEUE_MAX_RETRIES`: 最大重试次数 (默认: 3)
- `QUEUE_STREAM_NAME`: 队列流名称 (默认: mocks3:tasks)

### Redis配置
队列服务依赖Redis作为消息存储后端：
- **Redis Streams**: 主要消息队列
- **消费者组**: 确保消息可靠处理
- **失败队列**: 存储处理失败的任务

## 使用示例

### 添加文件删除任务
```bash
curl -X POST http://localhost:8083/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "file_deletion",
    "priority": 1,
    "data": {
      "bucket": "my-bucket",
      "key": "path/to/file.txt"
    }
  }'
```

### 查看任务状态
```bash
curl http://localhost:8083/api/v1/tasks/task-id-here
```

### 获取队列统计
```bash
curl http://localhost:8083/api/v1/stats
```

### 启动额外工作节点
```bash
curl -X POST http://localhost:8083/api/v1/workers/worker-4/start
```

## 运行方式

### 直接运行
```bash
cd services/queue
go run cmd/server/main.go
```

### Docker运行
```bash
cd services/queue
docker-compose up -d
```

## 任务处理流程

### 1. 任务提交
1. 客户端通过API提交任务
2. 任务被添加到Redis Stream
3. 返回任务ID和流ID

### 2. 任务处理
1. 工作节点从Redis Stream读取任务
2. 更新任务状态为"processing"
3. 根据任务类型执行相应处理逻辑
4. 处理成功则确认消息，失败则重试或标记失败

### 3. 错误处理
1. 处理失败的任务会被重新入队
2. 超过最大重试次数后移入失败队列
3. 失败任务可手动重新处理

## 集成说明

### 与其他服务集成
- **Metadata Service**: 删除任务会调用元数据服务清理记录
- **Storage Service**: 文件删除任务会调用存储服务删除实际文件
- **Consul**: 服务注册和发现
- **OpenTelemetry**: 统一可观测性

### 客户端集成
其他服务可通过以下方式使用队列服务：
1. 直接HTTP API调用
2. 使用shared/client中的QueueClient
3. 通过服务发现自动定位队列服务

## 监控指标

### 任务指标
- 待处理任务数量
- 处理中任务数量  
- 已完成任务数量
- 失败任务数量

### 性能指标
- 任务处理速度
- 平均处理时间
- 工作节点利用率
- 队列积压情况

## 故障排查

### 常见问题
1. **Redis连接失败**: 检查Redis服务状态和网络连接
2. **任务堆积**: 增加工作节点数量或优化处理逻辑
3. **任务失败率高**: 检查下游服务状态和网络稳定性
4. **内存使用过高**: 调整批处理大小和处理超时时间

### 日志级别
- `debug`: 详细的任务处理日志
- `info`: 任务创建、完成等关键事件
- `warn`: 重试和异常情况
- `error`: 严重错误和系统故障

## 目录结构
```
services/queue/
├── cmd/server/           # 应用入口
├── internal/
│   ├── config/          # 配置管理
│   ├── handler/         # HTTP处理器
│   ├── service/         # 业务逻辑和工作节点
│   └── repository/      # Redis数据访问
├── Dockerfile           # Docker构建
└── docker-compose.yml   # 本地运行配置
```
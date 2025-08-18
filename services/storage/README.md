# Storage Service

存储服务是MockS3微服务架构中负责实际文件存储和管理的核心服务。

## 功能特性

### 🗂️ 多节点存储
- **冗余存储**: 支持多个存储节点（stg1, stg2, stg3）
- **故障恢复**: 节点故障时的自动回退机制
- **数据一致性**: 顺序写入确保数据完整性

### 📁 文件操作
- **上传**: PUT /{bucket}/{key} - S3兼容的文件上传
- **下载**: GET /{bucket}/{key} - 文件下载和读取
- **删除**: DELETE /{bucket}/{key} - 文件删除
- **元信息**: HEAD /{bucket}/{key} - 获取文件元信息
- **列表**: GET /{bucket} - 列出bucket中的对象

### 🔗 服务集成
- **元数据同步**: 与Metadata Service集成
- **第三方回退**: 支持第三方数据源
- **服务发现**: Consul注册和发现

### 📊 管理功能
- **API接口**: RESTful管理API
- **统计信息**: 存储使用情况统计
- **健康检查**: 节点和服务健康状态监控

## API接口

### S3兼容接口
```
PUT    /{bucket}/{key}     # 上传对象
GET    /{bucket}/{key}     # 下载对象  
DELETE /{bucket}/{key}     # 删除对象
HEAD   /{bucket}/{key}     # 获取对象元信息
GET    /{bucket}           # 列出对象
```

### 管理API
```
POST   /api/v1/objects           # 创建对象
GET    /api/v1/objects/{bucket}/{key}  # 获取对象信息
DELETE /api/v1/objects/{bucket}/{key}  # 删除对象
GET    /api/v1/objects           # 列出对象
GET    /api/v1/stats             # 获取统计信息
GET    /health                   # 健康检查
```

## 配置说明

### 环境变量
- `SERVICE_PORT`: 服务端口 (默认: 8082)
- `STORAGE_DATA_DIR`: 存储根目录 (默认: ./data/storage)
- `STORAGE_STG1_PATH`: 节点1路径 (默认: ./data/storage/stg1)
- `STORAGE_STG2_PATH`: 节点2路径 (默认: ./data/storage/stg2) 
- `STORAGE_STG3_PATH`: 节点3路径 (默认: ./data/storage/stg3)
- `METADATA_SERVICE_URL`: 元数据服务地址 (默认: http://localhost:8081)

### 存储架构
```
data/storage/
├── stg1/           # 主存储节点
│   ├── bucket1/
│   └── bucket2/
├── stg2/           # 备份节点1
│   ├── bucket1/
│   └── bucket2/
└── stg3/           # 备份节点2
    ├── bucket1/
    └── bucket2/
```

## 运行方式

### 直接运行
```bash
cd services/storage
go run cmd/server/main.go
```

### Docker运行
```bash
cd services/storage
docker-compose up -d
```

## 关键特性

### 🔄 故障恢复机制
1. **读取优先级**: stg1 → stg2 → stg3 → 第三方服务
2. **写入策略**: 顺序写入所有可用节点
3. **自动回退**: 节点故障时的透明切换

### 📈 可观测性
- **OpenTelemetry**: 统一的日志、追踪、指标
- **Prometheus指标**: 存储使用、性能统计
- **结构化日志**: 详细的操作记录

### 🛡️ 错误处理
- **重试机制**: 网络异常时的自动重试
- **熔断器**: 防止级联故障
- **优雅降级**: 部分节点故障时的服务延续

## 目录结构
```
services/storage/
├── cmd/server/           # 应用入口
├── internal/
│   ├── config/          # 配置管理
│   ├── handler/         # HTTP处理器
│   ├── service/         # 业务逻辑
│   └── repository/      # 存储实现
├── Dockerfile           # Docker构建
└── docker-compose.yml   # 本地运行配置
```
# Third-Party Service

第三方服务是MockS3微服务架构中负责外部数据源集成和缓存管理的服务，为系统提供数据回退和备份能力。

## 功能特性

### 🔗 多数据源支持
- **S3兼容**: 支持AWS S3、MinIO等S3兼容存储
- **HTTP API**: 支持REST API数据源
- **优先级**: 按优先级顺序访问数据源
- **故障转移**: 自动切换到备用数据源

### 💾 智能缓存
- **LRU缓存**: 最近最少使用淘汰策略
- **自动过期**: 基于TTL的缓存过期
- **统计信息**: 命中率、淘汰率等性能指标
- **内存管理**: 可配置的最大缓存大小

### 🛡️ 容错机制
- **重试逻辑**: 失败请求的自动重试
- **降级策略**: 数据源不可用时的优雅降级
- **健康检查**: 定期检查数据源健康状态

### 📊 监控统计
- **访问统计**: 请求数量、成功率等
- **缓存统计**: 命中率、内存使用等
- **数据源统计**: 各数据源的使用情况

## API接口

### 对象操作
```
GET    /api/v1/objects/:bucket/:key    # 获取对象
POST   /api/v1/objects                 # 存储对象
DELETE /api/v1/objects/:bucket/:key    # 删除对象
GET    /api/v1/objects?bucket=xxx      # 列出对象
```

### 元数据操作
```
GET    /api/v1/metadata/:bucket/:key   # 获取对象元数据
```

### 数据源管理
```
POST   /api/v1/datasources             # 添加数据源
GET    /api/v1/datasources             # 获取数据源列表
```

### 缓存管理
```
POST   /api/v1/cache                   # 缓存对象
DELETE /api/v1/cache/:bucket/:key      # 清除缓存
```

### 监控接口
```
GET    /api/v1/stats                   # 获取统计信息
GET    /health                         # 健康检查
```

## 配置说明

### 环境变量
- `SERVER_PORT`: 服务端口 (默认: 8084)
- `CACHE_ENABLED`: 启用缓存 (默认: true)
- `CACHE_TTL`: 缓存TTL秒数 (默认: 3600)
- `CACHE_MAX_SIZE`: 最大缓存大小MB (默认: 1024)
- `CACHE_STRATEGY`: 缓存策略 (默认: lru)

### 数据源配置
```bash
# 数据源1 - S3
DATASOURCE_1_NAME=backup-s3
DATASOURCE_1_TYPE=s3
DATASOURCE_1_ENDPOINT=https://s3.amazonaws.com
DATASOURCE_1_ACCESS_KEY=your-access-key
DATASOURCE_1_SECRET_KEY=your-secret-key
DATASOURCE_1_REGION=us-east-1
DATASOURCE_1_BUCKET=backup-bucket
DATASOURCE_1_ENABLED=true

# 数据源2 - HTTP
DATASOURCE_2_NAME=backup-http
DATASOURCE_2_TYPE=http
DATASOURCE_2_ENDPOINT=https://backup.example.com/api
DATASOURCE_2_ENABLED=false
```

## 数据源类型

### S3兼容数据源
支持所有S3兼容的对象存储服务：
- Amazon S3
- MinIO
- Ceph Object Gateway
- 阿里云OSS (S3模式)

### HTTP数据源
支持RESTful API数据源：
- 自定义存储API
- 其他对象存储服务的HTTP接口
- CDN边缘节点

## 使用示例

### 获取对象
```bash
curl http://localhost:8084/api/v1/objects/my-bucket/path/to/file.txt
```

### 获取统计信息
```bash
curl http://localhost:8084/api/v1/stats
```

### 添加数据源
```bash
curl -X POST http://localhost:8084/api/v1/datasources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "backup-storage",
    "config": "{\"endpoint\":\"https://backup.example.com\"}"
  }'
```

### 缓存对象
```bash
curl -X POST http://localhost:8084/api/v1/cache \
  -H "Content-Type: application/json" \
  -d '{
    "bucket": "my-bucket",
    "key": "important-file.txt",
    "size": 1024,
    "content_type": "text/plain",
    "data": "base64encodeddata"
  }'
```

## 运行方式

### 直接运行
```bash
cd services/third-party
go run cmd/server/main.go
```

### Docker运行
```bash
cd services/third-party
docker-compose up -d
```

## 集成说明

### 与Storage Service集成
Storage Service在对象读取失败时会自动调用Third-Party Service：

```go
// Storage Service代码示例
if object, err := storageService.ReadObject(ctx, bucket, key); err != nil {
    // 尝试从第三方服务获取
    object, err = thirdPartyClient.GetObject(ctx, bucket, key)
    if err == nil {
        // 异步缓存到本地
        go storageService.WriteObject(ctx, object)
    }
}
```

### 缓存策略
- **写入时缓存**: 从数据源获取的对象自动缓存
- **读取优先**: 优先从缓存读取，缓存未命中才访问数据源
- **智能淘汰**: 基于LRU算法和内存限制的淘汰策略

### 故障处理
- **数据源故障**: 自动切换到下一优先级数据源
- **网络异常**: 指数退避重试机制
- **缓存故障**: 降级为直接访问数据源

## 性能优化

### 缓存优化
- 根据访问模式调整缓存大小
- 设置合适的TTL值
- 监控缓存命中率

### 网络优化
- 配置合适的连接超时
- 使用连接池复用连接
- 启用HTTP/2支持

### 并发优化
- 异步处理非关键操作
- 并发访问多个数据源
- 使用goroutine池控制并发数

## 监控指标

### 缓存指标
- 缓存命中率
- 缓存大小和使用率
- 淘汰次数和频率

### 数据源指标
- 各数据源访问次数
- 响应时间分布
- 错误率统计

### 系统指标
- 内存使用情况
- CPU使用率
- 网络I/O统计

## 目录结构
```
services/third-party/
├── cmd/server/           # 应用入口
├── internal/
│   ├── config/          # 配置管理
│   ├── handler/         # HTTP处理器
│   ├── service/         # 业务逻辑
│   └── repository/      # 数据访问和缓存
├── Dockerfile           # Docker构建
└── docker-compose.yml   # 本地运行配置
```
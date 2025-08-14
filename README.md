# 微服务部署指南

## 📋 系统概述

本微服务系统基于原 s3-storage 单体项目改造，实现了完整的 S3 兼容存储服务微服务架构，专为 mock 环境设计，支持全面的混沌工程测试。

### 🏗️ 架构组件

#### 微服务 (6个)
1. **API Gateway** - Nginx 反向代理，统一入口
2. **S3 API Service** - S3 兼容接口服务  
3. **Metadata Service** - 元数据管理服务
4. **Storage Service** - 分布式存储服务
5. **Task Processor** - 异步任务处理服务
6. **Chaos Engineering** - 混沌工程/错误注入服务
7. **Admin API** - 统一管理接口服务

#### 基础设施 (6个)
1. **Consul** - 服务发现和配置管理
2. **PostgreSQL** - 主数据库
3. **Redis Cache** - 缓存层
4. **Redis Queue** - 消息队列
5. **Elasticsearch** - 日志存储
6. **Prometheus** - 监控指标

## 🚀 快速部署

### 1. 环境准备

```bash
# 系统要求
- Docker 20.10+
- Docker Compose 2.0+
- 至少 4GB 内存
- 10GB+ 磁盘空间

# 端口检查（确保以下端口未占用）
- 80 (API Gateway)
- 8080-8090 (微服务)
- 5432 (PostgreSQL)
- 6379, 6380 (Redis)
- 8500 (Consul)
- 9090 (Prometheus)
- 9200 (Elasticsearch)
```

### 2. 一键部署

```bash
# 克隆项目后，进入目录
cd micro-s3

# 验证配置完整性
./scripts/validate-config.sh

# 设置开发环境
./scripts/setup-dev.sh

# 启动所有服务
make up

# 验证部署
./scripts/test-services.sh
```

### 3. 服务访问

部署完成后，可以通过以下地址访问：

| 服务 | 访问地址 | 说明 |
|------|----------|------|
| 主服务 | http://localhost | S3 API 访问入口 |
| 管理面板 | http://localhost/api/v1/admin/overview | 系统总览 |
| Consul UI | http://localhost:8500 | 服务发现 |
| Prometheus | http://localhost:9090 | 监控面板 |

## 🔧 常用操作

### 服务管理

```bash
# 查看服务状态
make status

# 查看所有日志
make logs

# 查看特定服务日志
make logs-api
make logs-metadata
make logs-storage

# 重启服务
make restart

# 停止服务
make down

# 清理所有数据
make clean
```

### 基本功能测试

```bash
# 上传文件
curl -X PUT "http://localhost/test-bucket/sample.txt" \
     -H "Content-Type: text/plain" \
     -d "Hello, Micro S3!"

# 下载文件
curl "http://localhost/test-bucket/sample.txt"

# 删除文件
curl -X DELETE "http://localhost/test-bucket/sample.txt"

# 列出对象
curl "http://localhost/test-bucket"
```

### 混沌工程测试

```bash
# 查看当前规则
curl "http://localhost/api/v1/chaos/rules" | jq

# 创建网络延迟规则
curl -X POST "http://localhost/api/v1/chaos/rules" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "Network Delay Test",
       "service": "storage",
       "failure_type": "network_timeout",
       "failure_rate": 0.1,
       "duration": "30s",
       "enabled": true
     }'

# 查看混沌统计
curl "http://localhost/api/v1/chaos/stats" | jq
```

## 🏢 生产部署

### 1. 环境变量配置

复制并编辑生产环境配置：

```bash
cp .env .env.production
```

关键配置项：
- `POSTGRES_PASSWORD` - 数据库密码
- `REDIS_PASSWORD` - Redis 密码  
- `LOG_LEVEL` - 日志级别设为 INFO
- `DEV_MODE` - 设为 false

### 2. 安全加固

```yaml
# docker-compose.override.prod.yml
services:
  consul:
    environment:
      - CONSUL_ACL_ENABLED=true
      - CONSUL_ACL_DEFAULT_POLICY=deny
  
  postgres:
    environment:
      - POSTGRES_PASSWORD=${STRONG_POSTGRES_PASSWORD}
  
  redis-cache:
    command: redis-server --requirepass ${REDIS_PASSWORD}
```

### 3. 资源限制

```yaml
services:
  s3-api:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### 4. 健康检查

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 60s
```

## 🔍 监控和维护

### 关键指标监控

通过 Prometheus 监控以下指标：

```bash
# 服务可用性
up == 1

# HTTP 请求率
rate(http_requests_total[5m])

# 响应时间 95 分位数
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# 存储使用率
storage_usage_percent > 80

# 队列积压
redis_queue_length > 1000
```

### 日志分析

```bash
# 查看错误日志
docker-compose logs | grep -i error

# 搜索特定服务错误
docker-compose logs s3-api | grep -E "error|fail|timeout"

# 按时间范围查看日志
docker-compose logs --since="1h" --until="30m"
```

## 🚨 故障排除

### 常见问题

#### 1. 端口占用
```bash
# 检查端口占用
lsof -i :80
sudo kill -9 <PID>
```

#### 2. 磁盘空间不足
```bash
# 清理 Docker 资源
docker system prune -af
docker volume prune -f
```

#### 3. 服务启动失败
```bash
# 检查服务日志
docker-compose logs <service-name>

# 检查资源使用
docker stats
```

#### 4. 数据库连接问题
```bash
# 测试数据库连接
docker-compose exec postgres psql -U micro_s3 -d micro_s3 -c "SELECT version();"

# 检查 Redis 连接
docker-compose exec redis-cache redis-cli ping
```

### 性能调优

#### 1. 数据库优化
```sql
-- 检查慢查询
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC LIMIT 10;

-- 优化索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_objects_bucket_key 
ON objects(bucket_name, object_key);
```

#### 2. Redis 调优
```bash
# 增加内存限制
redis-server --maxmemory 1gb --maxmemory-policy allkeys-lru

# 监控内存使用
redis-cli INFO memory
```

#### 3. 应用调优
```yaml
# 增加工作进程数
task-processor:
  environment:
    - WORKER_COUNT=10
    - REDIS_POOL_SIZE=20

# 调整数据库连接池
metadata:
  environment:
    - DB_MAX_CONNECTIONS=100
    - DB_IDLE_CONNECTIONS=10
```
## 扩展策略

1. **水平扩展**
   - 增加服务实例数
   - 使用 Docker Swarm 或 Kubernetes

2. **垂直扩展** 
   - 增加单个容器资源限制
   - 优化服务配置参数

3. **数据分片**
   - PostgreSQL 读写分离
   - Redis 集群模式

## 🆘 获取帮助

如遇问题，请按以下步骤：

1. 查看 [故障排除](#-故障排除) 部分
2. 检查服务日志：`make logs`
3. 运行健康检查：`./scripts/test-services.sh`
4. 查看系统状态：`curl http://localhost/api/v1/admin/overview`

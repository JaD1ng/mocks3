# MockS3 部署指南

## 系统组件

- **S3 API Gateway** (8080): 对外接口
- **微服务**: metadata(8081), storage(8082), queue(8083), third-party(8084), mock-error(8085)
- **基础设施**: PostgreSQL(5432), Redis(6379), Consul(8500)
- **监控**: Grafana(3000), Prometheus(9090), Kibana(5601), Elasticsearch(9200)

## 部署命令

### 一键部署
```bash
# 推荐方式：自动构建+启动
make build-all && make up

# 或自动构建+启动
docker-compose up -d

# 手动分步（仅在需要单独构建时使用）
make build-all             # 仅构建镜像
docker-compose up -d --no-build  # 仅启动，不构建
```

### 分步部署
```bash
# 1. 基础设施
docker-compose up -d postgres redis consul

# 2. 构建服务
## 方式1: 使用Makefile构建所有服务
make build-all

## 方式2: 单独构建服务
make metadata
make storage  
make queue
make third-party
make mock-error
```

# 3. 启动微服务
```bash
docker-compose up -d metadata-service storage-service queue-service third-party-service mock-error-service nginx-gateway
```

# 4. 启动监控（可选）
```bash
docker-compose up -d elasticsearch prometheus grafana kibana
```

### 仅核心服务
```
docker-compose up -d postgres redis consul metadata-service storage-service nginx-gateway
```

## 验证部署

```bash
make health-check                           # 完整健康检查
docker-compose ps                           # 容器状态
curl http://localhost:8080/health           # 网关检查
curl http://localhost:8500/v1/catalog/services  # 服务发现
```

## S3 API使用

```bash
# 创建存储桶
curl -X PUT http://localhost:8080/test-bucket/

# 上传文件
curl -X PUT http://localhost:8080/test-bucket/test.txt -d "Hello MockS3"

# 下载文件
curl http://localhost:8080/test-bucket/test.txt

# 删除文件
curl -X DELETE http://localhost:8080/test-bucket/test.txt
```

## 错误注入

```bash
# 查看规则
curl http://localhost:8085/api/v1/rules

# 注入500错误
curl -X POST http://localhost:8085/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{"service": "storage-service", "errorType": "HTTP_500", "probability": 0.1}'
```

## 管理命令

```bash
make down                  # 停止服务
make clean                 # 清理构建
make logs                  # 查看日志
make logs-metadata         # 特定服务日志
make shell-metadata        # 进入容器
```

## 监控地址

- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090
- Consul: http://localhost:8500
- Kibana: http://localhost:5601

## 故障排除

```bash
docker logs mocks3-consul              # 查看日志
docker system prune -f                 # 清理资源
docker-compose restart consul          # 重启服务
docker-compose build --no-cache        # 重新构建
```
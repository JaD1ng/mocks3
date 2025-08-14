# 开发指南

## 快速开始

1. 启动所有服务：
   ```bash
   make up
   ```

2. 查看服务状态：
   ```bash
   make status
   ```

3. 运行测试：
   ```bash
   make test
   ```

## 开发工具

启动开发工具（数据库管理、Redis 管理等）：
```bash
docker-compose --profile dev-tools up -d
```

访问地址：
- Adminer (PostgreSQL): http://localhost:8080
- Redis Commander: http://localhost:8081

## 日志查看

```bash
# 查看所有服务日志
make logs

# 查看特定服务日志
make logs-api
make logs-metadata
make logs-storage
```

## 调试

1. 查看服务状态：
   ```bash
   curl http://localhost/api/v1/admin/overview
   ```

2. 查看存储统计：
   ```bash
   curl http://localhost/api/v1/storage/stats
   ```

3. 查看任务队列：
   ```bash
   curl http://localhost/api/v1/tasks/queue/stats
   ```

## 故障排除

1. 检查服务健康状态
2. 查看日志文件
3. 检查端口占用
4. 验证网络连通性

## 清理

```bash
# 停止所有服务
make down

# 清理所有数据
make clean
```

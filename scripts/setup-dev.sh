#!/bin/bash

# 开发环境设置脚本

set -e

echo "🚀 设置微服务开发环境..."

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 检查依赖
check_dependency() {
    local cmd="$1"
    local name="$2"
    
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo -e "${RED}✗ $name 未安装${NC}"
        echo "请先安装 $name"
        exit 1
    else
        echo -e "${GREEN}✓ $name 已安装${NC}"
    fi
}

echo "📋 检查系统依赖..."

check_dependency "docker" "Docker"
check_dependency "docker-compose" "Docker Compose"
check_dependency "make" "Make"
check_dependency "curl" "cURL"

# 检查 Docker 是否运行
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}✗ Docker 未运行${NC}"
    echo "请启动 Docker 服务"
    exit 1
else
    echo -e "${GREEN}✓ Docker 正在运行${NC}"
fi

echo ""
echo "🔧 创建必要的目录和文件..."

# 创建数据目录（在项目根目录）
mkdir -p ../data/stg1 ../data/stg2 ../data/stg3
mkdir -p ../data/postgres ../data/consul ../data/redis-cache ../data/redis-queue
mkdir -p ../data/elasticsearch ../data/prometheus
mkdir -p ../logs

echo -e "${GREEN}✓ 数据目录已创建${NC}"

# 设置权限
chmod -R 755 ../data/
chmod -R 755 ../logs/

# 创建环境变量文件（在项目根目录）
if [ ! -f ../.env ]; then
    cat > ../.env << 'EOF'
# 微服务环境变量配置

# 基础设置
COMPOSE_PROJECT_NAME=micro-s3
DOCKER_REGISTRY=

# 服务端口
API_GATEWAY_PORT=80
S3_API_PORT=8080
METADATA_PORT=8081
STORAGE_PORT=8082
TASK_PROCESSOR_PORT=8083
CHAOS_ENGINEERING_PORT=8084
ADMIN_API_PORT=8090

# 基础设施端口
CONSUL_PORT=8500
POSTGRES_PORT=5432
REDIS_CACHE_PORT=6379
REDIS_QUEUE_PORT=6380
ELASTICSEARCH_PORT=9200
PROMETHEUS_PORT=9090

# 数据库配置
POSTGRES_DB=micro_s3
POSTGRES_USER=micro_s3
POSTGRES_PASSWORD=micro_s3_password

# Redis 配置
REDIS_PASSWORD=

# 日志级别
LOG_LEVEL=INFO

# 开发模式
DEV_MODE=true
DEBUG=false
EOF
    echo -e "${GREEN}✓ 环境配置文件 .env 已创建${NC}"
else
    echo -e "${YELLOW}! ../.env 文件已存在，跳过${NC}"
fi

echo ""
echo "🐳 拉取必要的 Docker 镜像..."

# 拉取基础镜像
docker pull golang:1.24-alpine
docker pull alpine:latest
docker pull nginx:alpine
docker pull hashicorp/consul:latest
docker pull postgres:15
docker pull redis:alpine
docker pull docker.elastic.co/elasticsearch/elasticsearch:8.8.0
docker pull prom/prometheus:latest

echo -e "${GREEN}✓ Docker 镜像拉取完成${NC}"

echo ""
echo "🏗️ 构建微服务镜像..."

# 构建所有服务
if make build; then
    echo -e "${GREEN}✓ 微服务构建成功${NC}"
else
    echo -e "${RED}✗ 微服务构建失败${NC}"
    echo "请检查构建日志"
    exit 1
fi

echo ""
echo "🔍 验证构建结果..."

# 检查镜像是否构建成功
services=("micro-s3-api-gateway" "micro-s3-s3-api" "micro-s3-metadata" "micro-s3-storage" "micro-s3-task-processor" "micro-s3-chaos-engineering" "micro-s3-admin-api")

for service in "${services[@]}"; do
    if docker images | grep -q "$service"; then
        echo -e "${GREEN}✓ $service 镜像已构建${NC}"
    else
        echo -e "${RED}✗ $service 镜像构建失败${NC}"
    fi
done

echo ""
echo "📝 创建开发配置文件..."

# 创建开发用的 docker-compose override 文件（在项目根目录）
if [ ! -f ../docker-compose.override.yml ]; then
    cat > ../docker-compose.override.yml << 'EOF'
version: '3.8'

# 开发环境覆盖配置

services:
  # 开发模式下的服务配置调整
  s3-api:
    environment:
      - DEBUG=true
      - LOG_LEVEL=DEBUG
    volumes:
      - ./logs:/var/log/app

  metadata:
    environment:
      - DEBUG=true
      - LOG_LEVEL=DEBUG
    volumes:
      - ./logs:/var/log/app

  storage:
    environment:
      - DEBUG=true
      - LOG_LEVEL=DEBUG
    volumes:
      - ./logs:/var/log/app
      - ./data:/data

  task-processor:
    environment:
      - DEBUG=true
      - LOG_LEVEL=DEBUG
      - WORKER_COUNT=2
    volumes:
      - ./logs:/var/log/app

  chaos-engineering:
    environment:
      - DEBUG=true
      - LOG_LEVEL=DEBUG
    volumes:
      - ./logs:/var/log/app

  admin-api:
    environment:
      - DEBUG=true
      - LOG_LEVEL=DEBUG
    volumes:
      - ./logs:/var/log/app

  # 开发工具
  postgres-admin:
    image: adminer
    container_name: postgres-admin
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    networks:
      - micro-s3-net
    profiles:
      - dev-tools

  redis-commander:
    image: rediscommander/redis-commander:latest
    container_name: redis-commander
    environment:
      - REDIS_HOSTS=cache:redis-cache:6379,queue:redis-queue:6379
    ports:
      - "8081:8081"
    depends_on:
      - redis-cache
      - redis-queue
    networks:
      - micro-s3-net
    profiles:
      - dev-tools
EOF
    echo -e "${GREEN}✓ 开发配置文件已创建${NC}"
else
    echo -e "${YELLOW}! ../docker-compose.override.yml 已存在，跳过${NC}"
fi

echo ""
echo "📚 创建开发文档..."

# 创建开发指南（在项目根目录）
if [ ! -f ../DEVELOPMENT.md ]; then
    cat > ../DEVELOPMENT.md << 'EOF'
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
EOF
    echo -e "${GREEN}✓ 开发文档已创建${NC}"
else
    echo -e "${YELLOW}! ../DEVELOPMENT.md 已存在，跳过${NC}"
fi

echo ""
echo "🎉 开发环境设置完成！"
echo ""
echo -e "${GREEN}✅ 所有依赖已检查${NC}"
echo -e "${GREEN}✅ 目录结构已创建${NC}"  
echo -e "${GREEN}✅ 环境配置已生成${NC}"
echo -e "${GREEN}✅ Docker 镜像已构建${NC}"
echo -e "${GREEN}✅ 开发工具已配置${NC}"
echo ""
echo "🚀 下一步："
echo "  1. 启动服务: make up"
echo "  2. 运行测试: make test"
echo "  3. 查看状态: make status"
echo "  4. 查看文档: cat DEVELOPMENT.md"
echo ""
echo "🔗 访问地址："
echo "  - 系统概览: http://localhost/api/v1/admin/overview"
echo "  - Consul UI: http://localhost:8500"
echo "  - Prometheus: http://localhost:9090"
echo ""
echo -e "${YELLOW}💡 提示: 使用 'make help' 查看所有可用命令${NC}"
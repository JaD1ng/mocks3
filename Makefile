.PHONY: build up down logs clean help

# 默认目标
help:
	@echo "可用命令:"
	@echo "  build      - 构建所有服务"
	@echo "  up         - 启动所有服务"
	@echo "  down       - 停止所有服务"
	@echo "  logs       - 查看服务日志"
	@echo "  clean      - 清理资源"
	@echo "  test       - 运行测试"
	@echo "  restart    - 重启服务"

# 构建所有服务
build:
	@echo "构建所有微服务..."
	docker-compose build

# 启动服务
up:
	@echo "启动微服务..."
	docker-compose up -d
	@echo "等待服务启动..."
	sleep 10
	@echo "检查服务状态..."
	docker-compose ps

# 停止服务
down:
	@echo "停止微服务..."
	docker-compose down

# 查看日志
logs:
	docker-compose logs -f

# 查看特定服务日志
logs-api:
	docker-compose logs -f s3-api

logs-metadata:
	docker-compose logs -f metadata

logs-storage:
	docker-compose logs -f storage

logs-gateway:
	docker-compose logs -f api-gateway

# 清理资源
clean:
	@echo "清理 Docker 资源..."
	docker-compose down -v
	docker system prune -f
	docker volume prune -f

# 重启服务
restart: down up

# 运行测试
test:
	@echo "运行基本测试..."
	@echo "测试健康检查..."
	curl -f http://localhost/health || echo "健康检查失败"
	@echo "测试上传文件..."
	echo "Hello, World!" | curl -X PUT -d @- http://localhost/test-bucket/test.txt || echo "上传测试失败"
	@echo "测试下载文件..."
	curl -f http://localhost/test-bucket/test.txt || echo "下载测试失败"

# 开发模式 - 只启动基础设施
dev:
	@echo "启动开发环境（仅基础设施）..."
	docker-compose up -d consul postgres redis-cache redis-queue elasticsearch prometheus

# 监控面板
monitor:
	@echo "打开监控面板..."
	@echo "Consul UI: http://localhost:8500"
	@echo "Prometheus: http://localhost:9090"
	@echo "Elasticsearch: http://localhost:9200"

# 服务状态
status:
	docker-compose ps
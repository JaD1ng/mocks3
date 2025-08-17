# MockS3 微服务架构 Makefile
# 用于构建、测试、部署和管理 MockS3 系统

# 变量定义
DOCKER_REGISTRY ?= mocks3
VERSION ?= latest
ENVIRONMENT ?= development

# 服务列表
SERVICES := metadata storage queue third-party mock-error
IMAGES := $(foreach service,$(SERVICES),$(DOCKER_REGISTRY)/$(service)-service:$(VERSION))

# 默认目标
.PHONY: help
help: ## 显示帮助信息
	@echo "MockS3 微服务架构管理命令:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ======= 开发相关 =======

.PHONY: dev-setup
dev-setup: ## 设置开发环境
	@echo "设置开发环境..."
	@go mod download
	@go mod tidy
	@echo "开发环境设置完成"

.PHONY: fmt
fmt: ## 格式化代码
	@echo "格式化代码..."
	@go fmt ./...
	@echo "代码格式化完成"

.PHONY: lint
lint: ## 运行代码检查
	@echo "运行代码检查..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "请安装 golangci-lint"; exit 1; }
	@golangci-lint run ./...
	@echo "代码检查完成"

.PHONY: test
test: ## 运行测试
	@echo "运行单元测试..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "测试完成，覆盖率报告: coverage.html"

.PHONY: test-integration
test-integration: ## 运行集成测试
	@echo "运行集成测试..."
	@go test -v -tags=integration ./...
	@echo "集成测试完成"

# ======= 构建相关 =======

.PHONY: build-all
build-all: $(SERVICES) gateway ## 构建所有服务

.PHONY: $(SERVICES)
$(SERVICES): ## 构建指定微服务
	@echo "构建 $@ 服务..."
	@docker build -f services/$@/Dockerfile -t $(DOCKER_REGISTRY)/$@-service:$(VERSION) .
	@echo "$@ 服务构建完成"

.PHONY: gateway
gateway: ## 构建网关
	@echo "构建 Nginx Gateway..."
	@docker build -f gateway/Dockerfile -t $(DOCKER_REGISTRY)/gateway:$(VERSION) gateway/
	@echo "Gateway 构建完成"

.PHONY: build-local
build-local: ## 本地构建所有服务二进制文件
	@echo "本地构建服务二进制文件..."
	@for service in $(SERVICES); do \
		echo "构建 $$service..."; \
		CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/$$service-service ./services/$$service/cmd/server; \
	done
	@echo "本地构建完成"

.PHONY: clean-build
clean-build: ## 清理构建产物
	@echo "清理构建产物..."
	@rm -rf bin/
	@docker image prune -f
	@echo "清理完成"

# ======= 部署相关 =======

.PHONY: up
up: ## 启动完整堆栈
	@echo "启动 MockS3 微服务堆栈..."
	@docker-compose up -d
	@echo "等待服务启动..."
	@sleep 30
	@$(MAKE) health-check
	@echo "MockS3 堆栈启动完成"

.PHONY: down
down: ## 停止所有服务
	@echo "停止 MockS3 微服务堆栈..."
	@docker-compose down
	@echo "服务已停止"

.PHONY: restart
restart: down up ## 重启所有服务

.PHONY: up-infra
up-infra: ## 仅启动基础设施服务
	@echo "启动基础设施服务..."
	@docker-compose up -d postgres redis consul elasticsearch prometheus grafana
	@echo "基础设施服务启动完成"

.PHONY: up-services
up-services: ## 仅启动微服务
	@echo "启动微服务..."
	@docker-compose up -d metadata-service storage-service queue-service third-party-service mock-error-service nginx-gateway
	@echo "微服务启动完成"

.PHONY: logs
logs: ## 查看所有服务日志
	@docker-compose logs -f

.PHONY: logs-%
logs-%: ## 查看指定服务日志 (例: make logs-metadata)
	@docker-compose logs -f $*

.PHONY: shell-%
shell-%: ## 进入指定服务容器 (例: make shell-metadata)
	@docker-compose exec $* sh

# ======= 监控和诊断 =======

.PHONY: health-check
health-check: ## 执行健康检查
	@echo "执行健康检查..."
	@./scripts/health-check.sh
	@echo "健康检查完成"

.PHONY: status
status: ## 查看服务状态
	@echo "=== Docker Compose 状态 ==="
	@docker-compose ps
	@echo ""
	@echo "=== 服务端点 ==="
	@echo "S3 API Gateway: http://localhost:8080"
	@echo "Consul UI: http://localhost:8500"
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Kibana: http://localhost:5601"
	@echo "Elasticsearch: http://localhost:9200"

.PHONY: metrics
metrics: ## 获取基本指标
	@echo "=== 服务指标 ==="
	@curl -s http://localhost:8080/health | jq '.'
	@echo ""
	@curl -s http://localhost:8081/health | jq '.'
	@echo ""
	@curl -s http://localhost:8082/health | jq '.'

.PHONY: consul-services
consul-services: ## 查看 Consul 注册的服务
	@echo "=== Consul 注册的服务 ==="
	@curl -s http://localhost:8500/v1/catalog/services | jq '.'

# ======= 测试相关 =======

.PHONY: test-api
test-api: ## 测试 API 功能
	@echo "测试 S3 API 功能..."
	@./scripts/test-api.sh
	@echo "API 测试完成"

.PHONY: test-error-injection
test-error-injection: ## 测试错误注入
	@echo "测试错误注入功能..."
	@./scripts/test-error-injection.sh
	@echo "错误注入测试完成"

.PHONY: benchmark
benchmark: ## 运行性能基准测试
	@echo "运行性能基准测试..."
	@go test -bench=. -benchmem ./...
	@echo "基准测试完成"

.PHONY: load-test
load-test: ## 运行负载测试
	@echo "运行负载测试..."
	@command -v wrk >/dev/null 2>&1 || { echo "请安装 wrk 工具"; exit 1; }
	@wrk -t12 -c400 -d30s --script=scripts/load-test.lua http://localhost:8080/health
	@echo "负载测试完成"

# ======= 数据管理 =======

.PHONY: db-migrate
db-migrate: ## 运行数据库迁移
	@echo "运行数据库迁移..."
	@docker-compose exec postgres psql -U mocks3 -d mocks3 -f /docker-entrypoint-initdb.d/01-init-schema.sql
	@echo "数据库迁移完成"

.PHONY: db-backup
db-backup: ## 备份数据库
	@echo "备份数据库..."
	@mkdir -p backups
	@docker-compose exec postgres pg_dump -U mocks3 mocks3 > backups/mocks3-$(shell date +%Y%m%d_%H%M%S).sql
	@echo "数据库备份完成"

.PHONY: db-restore
db-restore: ## 恢复数据库 (需要指定 BACKUP_FILE)
	@echo "恢复数据库..."
	@test -n "$(BACKUP_FILE)" || { echo "请指定 BACKUP_FILE"; exit 1; }
	@docker-compose exec -T postgres psql -U mocks3 -d mocks3 < $(BACKUP_FILE)
	@echo "数据库恢复完成"

.PHONY: reset-data
reset-data: ## 重置所有数据
	@echo "警告: 这将删除所有数据!"
	@read -p "确认继续? [y/N]: " confirm && [ "$$confirm" = "y" ] || exit 1
	@docker-compose down -v
	@docker volume prune -f
	@echo "数据重置完成"

# ======= 维护相关 =======

.PHONY: update-deps
update-deps: ## 更新依赖
	@echo "更新 Go 依赖..."
	@go get -u ./...
	@go mod tidy
	@echo "依赖更新完成"

.PHONY: security-scan
security-scan: ## 运行安全扫描
	@echo "运行安全扫描..."
	@command -v trivy >/dev/null 2>&1 || { echo "请安装 trivy"; exit 1; }
	@for service in $(SERVICES); do \
		echo "扫描 $$service..."; \
		trivy image $(DOCKER_REGISTRY)/$$service-service:$(VERSION); \
	done
	@echo "安全扫描完成"

.PHONY: docs
docs: ## 生成文档
	@echo "生成 API 文档..."
	@command -v swag >/dev/null 2>&1 || { echo "请安装 swag: go install github.com/swaggo/swag/cmd/swag@latest"; exit 1; }
	@for service in $(SERVICES); do \
		echo "生成 $$service 文档..."; \
		swag init -g ./services/$$service/cmd/server/main.go -o ./docs/$$service; \
	done
	@echo "文档生成完成"

# ======= 发布相关 =======

.PHONY: tag
tag: ## 创建版本标签
	@test -n "$(TAG)" || { echo "请指定 TAG"; exit 1; }
	@git tag -a $(TAG) -m "Release $(TAG)"
	@git push origin $(TAG)
	@echo "标签 $(TAG) 创建完成"

.PHONY: push-images
push-images: ## 推送镜像到仓库
	@echo "推送镜像到仓库..."
	@for image in $(IMAGES); do \
		echo "推送 $$image..."; \
		docker push $$image; \
	done
	@docker push $(DOCKER_REGISTRY)/gateway:$(VERSION)
	@echo "镜像推送完成"

.PHONY: release
release: build-all push-images ## 构建并发布

# ======= 清理相关 =======

.PHONY: clean
clean: clean-build ## 清理所有构建产物和缓存
	@echo "清理 Docker 资源..."
	@docker system prune -f
	@echo "清理完成"

.PHONY: clean-all
clean-all: down clean reset-data ## 完全清理 (包括数据)
	@echo "完全清理完成"

# ======= 开发工具 =======

.PHONY: install-tools
install-tools: ## 安装开发工具
	@echo "安装开发工具..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "开发工具安装完成"

.PHONY: pre-commit
pre-commit: fmt lint test ## 提交前检查
	@echo "提交前检查完成"

# 服务特定目标
.PHONY: dev-metadata
dev-metadata: ## 本地运行元数据服务
	@echo "启动元数据服务 (开发模式)..."
	@cd services/metadata && go run cmd/server/main.go

.PHONY: dev-storage
dev-storage: ## 本地运行存储服务
	@echo "启动存储服务 (开发模式)..."
	@cd services/storage && go run cmd/server/main.go

.PHONY: dev-queue
dev-queue: ## 本地运行队列服务
	@echo "启动队列服务 (开发模式)..."
	@cd services/queue && go run cmd/server/main.go
#!/bin/bash

# 配置文件验证脚本

echo "🔍 验证微服务配置..."

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

validate_file() {
    local file="$1"
    local desc="$2"
    
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓ $desc${NC}"
        return 0
    else
        echo -e "${RED}✗ $desc 缺失${NC}: $file"
        return 1
    fi
}

validate_directory() {
    local dir="$1"
    local desc="$2"
    
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓ $desc${NC}"
        return 0
    else
        echo -e "${RED}✗ $desc 缺失${NC}: $dir"
        return 1
    fi
}

echo "📋 检查核心配置文件..."

# 检查核心配置
validate_file "docker-compose.yml" "Docker Compose 配置"
validate_file "Makefile" "构建脚本"
validate_file "README.md" "项目说明"
validate_file "service.md" "业务流程文档"

echo ""
echo "🏗️ 检查服务目录..."

# 检查服务目录
services=(
    "s3-api"
    "metadata"
    "storage"
    "task-processor"
    "chaos-engineering"
    "admin-api"
)

for service in "${services[@]}"; do
    validate_directory "services/$service" "$service 服务"
    validate_file "services/$service/cmd/main.go" "$service 主程序"
    validate_file "services/$service/Dockerfile" "$service Docker 文件" || echo -e "${YELLOW}  警告: 需要创建 Dockerfile${NC}"
done

echo ""
echo "🔧 检查基础设施配置..."

# 检查基础设施配置
validate_directory "infrastructure" "基础设施配置目录"
validate_directory "api-gateway" "API 网关配置"
validate_file "api-gateway/nginx.conf" "Nginx 配置"
validate_file "infrastructure/postgres/init/init.sql" "PostgreSQL 初始化脚本"
validate_file "infrastructure/consul/config/consul.json" "Consul 配置"
validate_file "infrastructure/elasticsearch/config/elasticsearch.yml" "Elasticsearch 配置"
validate_file "infrastructure/prometheus/config/prometheus.yml" "Prometheus 配置"

echo ""
echo "📜 检查脚本文件..."

# 检查脚本
validate_file "scripts/setup-dev.sh" "开发环境设置脚本"
validate_file "scripts/test-services.sh" "服务测试脚本"

# 设置脚本执行权限
chmod +x scripts/*.sh 2>/dev/null

echo ""
echo "🔍 检查 Go 模块配置..."

# 检查每个服务的 go.mod 文件
for service in "${services[@]}"; do
    if [ ! -f "services/$service/go.mod" ]; then
        echo -e "${YELLOW}! 创建 $service Go 模块...${NC}"
        
        # 创建 go.mod 文件
        cat > "services/$service/go.mod" << EOF
module micro-s3/$service

go 1.24.5

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/go-redis/redis/v8 v8.11.5
    github.com/hashicorp/consul/api v1.25.1
    github.com/lib/pq v1.10.9
    gorm.io/driver/postgres v1.5.2
    gorm.io/gorm v1.25.4
)
EOF
        echo -e "${GREEN}✓ $service go.mod 已创建${NC}"
    else
        echo -e "${GREEN}✓ $service Go 模块配置${NC}"
    fi
done

echo ""
echo "📊 生成配置检查报告..."

total_files=0
missing_files=0

# 统计缺失的关键文件
required_files=(
    "docker-compose.yml"
    "Makefile"
    "api-gateway/nginx.conf"
    "infrastructure/postgres/init/init.sql"
    "infrastructure/consul/config/consul.json"
    "scripts/setup-dev.sh"
    "scripts/test-services.sh"
)

for file in "${required_files[@]}"; do
    total_files=$((total_files + 1))
    if [ ! -f "$file" ]; then
        missing_files=$((missing_files + 1))
    fi
done

# 统计服务文件
for service in "${services[@]}"; do
    total_files=$((total_files + 2))  # main.go + go.mod
    [ ! -f "services/$service/cmd/main.go" ] && missing_files=$((missing_files + 1))
    [ ! -f "services/$service/go.mod" ] && missing_files=$((missing_files + 1))
done

present_files=$((total_files - missing_files))
completion_percent=$((present_files * 100 / total_files))

echo ""
echo -e "${GREEN}✅ 配置验证完成${NC}"
echo "  总计: $total_files 个核心文件"
echo "  存在: $present_files 个文件"
echo "  缺失: $missing_files 个文件"
echo "  完成度: ${completion_percent}%"

if [ $missing_files -eq 0 ]; then
    echo -e "${GREEN}🎉 所有配置文件都已就绪！${NC}"
    echo ""
    echo "🚀 下一步："
    echo "  1. 启动 Docker 服务"
    echo "  2. 运行: ./scripts/setup-dev.sh"
    echo "  3. 构建服务: make build"
    echo "  4. 启动服务: make up"
    echo "  5. 测试服务: ./scripts/test-services.sh"
else
    echo -e "${YELLOW}⚠️  仍有 $missing_files 个文件需要完善${NC}"
fi

echo ""
echo "🔗 重要提示："
echo "  - 确保 Docker 服务正在运行"
echo "  - 检查端口 80, 8080-8090, 5432, 6379-6380, 8500, 9090, 9200 未被占用"
echo "  - 生产环境需要修改默认密码和安全配置"
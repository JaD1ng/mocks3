#!/bin/bash

# MockS3 Gateway 启动脚本
# 用于启动完整的 MockS3 微服务堆栈

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 检查 Docker 和 Docker Compose
check_dependencies() {
    log_step "检查依赖项..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装或不在 PATH 中"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose 未安装或不在 PATH 中"
        exit 1
    fi
    
    log_info "依赖项检查通过"
}

# 清理旧容器和网络
cleanup() {
    log_step "清理旧容器和网络..."
    
    docker-compose -f docker-compose.yml down --remove-orphans || true
    docker network prune -f || true
    docker volume prune -f || true
    
    log_info "清理完成"
}

# 构建服务镜像
build_services() {
    log_step "构建服务镜像..."
    
    # 构建 Nginx Gateway
    log_info "构建 Nginx Gateway..."
    docker build -t mocks3/gateway:latest .
    
    # 构建其他微服务（如果 Dockerfile 存在）
    cd ..
    
    for service in metadata storage queue third-party mock-error; do
        if [ -f "services/${service}/Dockerfile" ]; then
            log_info "构建 ${service} 服务..."
            docker build -f "services/${service}/Dockerfile" -t "mocks3/${service}-service:latest" .
        else
            log_warn "未找到 services/${service}/Dockerfile，跳过构建"
        fi
    done
    
    cd gateway
    log_info "镜像构建完成"
}

# 启动基础设施服务
start_infrastructure() {
    log_step "启动基础设施服务..."
    
    # 启动 PostgreSQL, Redis, Consul
    docker-compose up -d postgres redis consul
    
    # 等待服务启动
    log_info "等待基础设施服务启动..."
    sleep 30
    
    # 检查服务状态
    if ! docker-compose ps postgres | grep -q "Up"; then
        log_error "PostgreSQL 启动失败"
        exit 1
    fi
    
    if ! docker-compose ps redis | grep -q "Up"; then
        log_error "Redis 启动失败"
        exit 1
    fi
    
    if ! docker-compose ps consul | grep -q "Up"; then
        log_error "Consul 启动失败"
        exit 1
    fi
    
    log_info "基础设施服务启动成功"
}

# 启动微服务
start_microservices() {
    log_step "启动微服务..."
    
    # 按依赖顺序启动服务
    log_info "启动 Mock Error Service..."
    docker-compose up -d mock-error-service
    sleep 10
    
    log_info "启动 Queue Service..."
    docker-compose up -d queue-service
    sleep 10
    
    log_info "启动 Metadata Service..."
    docker-compose up -d metadata-service
    sleep 15
    
    log_info "启动 Third-Party Service..."
    docker-compose up -d third-party-service
    sleep 10
    
    log_info "启动 Storage Service..."
    docker-compose up -d storage-service
    sleep 15
    
    log_info "启动 Nginx Gateway..."
    docker-compose up -d nginx-gateway
    sleep 10
    
    log_info "所有微服务启动完成"
}

# 健康检查
health_check() {
    log_step "执行健康检查..."
    
    # 定义服务和对应的健康检查端点
    declare -A services=(
        ["Gateway"]="http://localhost:8080/health"
        ["Metadata"]="http://localhost:8081/health"
        ["Storage"]="http://localhost:8082/health"
        ["Queue"]="http://localhost:8083/health"
        ["Third-Party"]="http://localhost:8084/health"
        ["Mock Error"]="http://localhost:8085/health"
    )
    
    all_healthy=true
    
    for service in "${!services[@]}"; do
        endpoint=${services[$service]}
        log_info "检查 ${service} 服务健康状态..."
        
        # 重试3次
        for i in {1..3}; do
            if curl -f -s "${endpoint}" > /dev/null; then
                log_info "${service} 服务健康"
                break
            else
                if [ $i -eq 3 ]; then
                    log_error "${service} 服务不健康"
                    all_healthy=false
                else
                    log_warn "${service} 服务健康检查失败，重试中... ($i/3)"
                    sleep 5
                fi
            fi
        done
    done
    
    if [ "$all_healthy" = true ]; then
        log_info "所有服务健康检查通过"
        return 0
    else
        log_error "部分服务健康检查失败"
        return 1
    fi
}

# 显示服务状态
show_status() {
    log_step "显示服务状态..."
    
    echo ""
    echo "=== MockS3 微服务堆栈状态 ==="
    docker-compose ps
    
    echo ""
    echo "=== 服务端点 ==="
    echo "• S3 API Gateway: http://localhost:8080"
    echo "• Admin Panel: http://localhost:8081"
    echo "• Consul UI: http://localhost:8500"
    echo "• Metadata Service: http://localhost:8081"
    echo "• Storage Service: http://localhost:8082"
    echo "• Queue Service: http://localhost:8083"
    echo "• Third-Party Service: http://localhost:8084"
    echo "• Mock Error Service: http://localhost:8085"
    
    echo ""
    echo "=== 快速测试 ==="
    echo "# 健康检查"
    echo "curl http://localhost:8080/health"
    echo ""
    echo "# 创建存储桶"
    echo "curl -X PUT http://localhost:8080/test-bucket/"
    echo ""
    echo "# 上传文件"
    echo "curl -X PUT http://localhost:8080/test-bucket/test.txt -d 'Hello MockS3!'"
    echo ""
    echo "# 下载文件"
    echo "curl http://localhost:8080/test-bucket/test.txt"
    echo ""
    echo "# 查看错误注入规则"
    echo "curl http://localhost:8085/api/v1/rules"
}

# 主函数
main() {
    log_info "启动 MockS3 微服务堆栈..."
    
    # 解析命令行参数
    CLEANUP=false
    BUILD=false
    SKIP_HEALTH=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --cleanup)
                CLEANUP=true
                shift
                ;;
            --build)
                BUILD=true
                shift
                ;;
            --skip-health)
                SKIP_HEALTH=true
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --cleanup     清理旧容器和网络"
                echo "  --build       重新构建服务镜像"
                echo "  --skip-health 跳过健康检查"
                echo "  -h, --help    显示帮助信息"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 执行步骤
    check_dependencies
    
    if [ "$CLEANUP" = true ]; then
        cleanup
    fi
    
    if [ "$BUILD" = true ]; then
        build_services
    fi
    
    start_infrastructure
    start_microservices
    
    if [ "$SKIP_HEALTH" = false ]; then
        if health_check; then
            show_status
            log_info "MockS3 微服务堆栈启动成功！"
        else
            log_error "部分服务启动失败，请检查日志"
            docker-compose logs --tail=50
            exit 1
        fi
    else
        show_status
        log_info "MockS3 微服务堆栈启动完成（跳过健康检查）"
    fi
}

# 信号处理
trap 'log_info "收到中断信号，正在停止服务..."; docker-compose down; exit 0' INT TERM

# 执行主函数
main "$@"
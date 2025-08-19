#!/bin/bash

# MockS3 健康检查脚本
# 用于验证所有服务的健康状态

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

# 服务健康检查端点配置
SERVICES=(
    "Gateway http://localhost:8080/health"
    "Metadata http://metadata-service:8081/health"
    "Storage http://storage-service:8082/health"
    "Queue http://queue-service:8083/health"
    "Third-Party http://third-party-service:8084/health"
    "Mock-Error http://mock-error-service:8085/health"
)

# 基础设施健康检查端点
INFRASTRUCTURE=(
    "Consul http://localhost:8500/v1/status/leader"
    "Prometheus http://localhost:9090/-/healthy"
    "Grafana http://localhost:3000/api/health"
    "Kibana http://localhost:5601/api/status"
    "Elasticsearch http://localhost:9200/_cluster/health"
)

# 检查单个端点健康状态
check_endpoint() {
    local name="$1"
    local url="$2"
    local timeout="${3:-10}"
    
    local status_code
    # 如果URL包含非localhost地址，使用docker exec在网络内执行
    if [[ "$url" =~ metadata-service|storage-service|queue-service|third-party-service|mock-error-service ]]; then
        # 在consul容器内执行，因为它有curl并且在同一网络中
        status_code=$(docker exec mocks3-consul curl -o /dev/null -s -w "%{http_code}" --max-time "$timeout" "$url" 2>/dev/null || echo "000")
    else
        status_code=$(curl -o /dev/null -s -w "%{http_code}" --max-time "$timeout" "$url" 2>/dev/null || echo "000")
    fi
    
    if [[ "$status_code" =~ ^(200|201|204)$ ]]; then
        log_info "✅ $name: 健康 (HTTP $status_code)"
        return 0
    else
        log_error "❌ $name: 不健康 (HTTP $status_code)"
        return 1
    fi
}

# 检查 Docker 容器状态
check_containers() {
    log_step "检查 Docker 容器状态..."
    
    local containers
    containers=$(docker-compose ps -q 2>/dev/null || echo "")
    
    if [ -z "$containers" ]; then
        log_error "未找到运行中的容器，请先启动服务"
        return 1
    fi
    
    local healthy=0
    local total=0
    
    while IFS= read -r container_id; do
        if [ -n "$container_id" ]; then
            total=$((total + 1))
            local container_name
            container_name=$(docker inspect --format '{{.Name}}' "$container_id" 2>/dev/null | sed 's/^.//')
            
            local status
            status=$(docker inspect --format '{{.State.Status}}' "$container_id" 2>/dev/null)
            
            if [ "$status" = "running" ]; then
                log_info "✅ 容器 $container_name: 运行中"
                healthy=$((healthy + 1))
            else
                log_error "❌ 容器 $container_name: $status"
            fi
        fi
    done <<< "$containers"
    
    log_info "容器状态: $healthy/$total 健康"
    
    if [ "$healthy" -eq "$total" ] && [ "$total" -gt 0 ]; then
        return 0
    else
        return 1
    fi
}

# 检查微服务健康状态
check_services() {
    log_step "检查微服务健康状态..."
    
    local healthy=0
    local total=${#SERVICES[@]}
    
    for service_entry in "${SERVICES[@]}"; do
        local name=$(echo "$service_entry" | awk '{print $1}')
        local endpoint=$(echo "$service_entry" | awk '{print $2}')
        if check_endpoint "$name Service" "$endpoint" 10; then
            healthy=$((healthy + 1))
        fi
        sleep 1
    done
    
    log_info "微服务状态: $healthy/$total 健康"
    
    if [ "$healthy" -eq "$total" ]; then
        return 0
    else
        return 1
    fi
}

# 检查基础设施健康状态
check_infrastructure() {
    log_step "检查基础设施健康状态..."
    
    local healthy=0
    local total=${#INFRASTRUCTURE[@]}
    
    for infra_entry in "${INFRASTRUCTURE[@]}"; do
        local name=$(echo "$infra_entry" | awk '{print $1}')
        local endpoint=$(echo "$infra_entry" | awk '{print $2}')
        if check_endpoint "$name" "$endpoint" 15; then
            healthy=$((healthy + 1))
        fi
        sleep 1
    done
    
    log_info "基础设施状态: $healthy/$total 健康"
    
    if [ "$healthy" -eq "$total" ]; then
        return 0
    else
        return 1
    fi
}

# 检查服务发现
check_service_discovery() {
    log_step "检查服务发现状态..."
    
    local consul_services
    consul_services=$(curl -s http://localhost:8500/v1/catalog/services 2>/dev/null || echo "{}")
    
    if [ "$consul_services" = "{}" ]; then
        log_error "无法获取 Consul 服务列表"
        return 1
    fi
    
    local service_count
    service_count=$(echo "$consul_services" | jq 'length' 2>/dev/null || echo "0")
    
    if [ "$service_count" -gt 0 ]; then
        log_info "✅ Consul 服务发现: $service_count 个服务已注册"
        
        # 显示注册的服务
        echo "$consul_services" | jq -r 'keys[]' 2>/dev/null | while read -r service; do
            log_info "  - $service"
        done
        
        return 0
    else
        log_error "❌ Consul 服务发现: 无服务注册"
        return 1
    fi
}

# 基本功能测试
test_basic_functionality() {
    log_step "执行基本功能测试..."
    
    # 测试 S3 API
    log_info "测试 S3 API..."
    
    # 创建测试存储桶
    local bucket_status
    bucket_status=$(curl -o /dev/null -s -w "%{http_code}" -X PUT http://localhost:8080/health-check-bucket/ 2>/dev/null || echo "000")
    
    if [[ "$bucket_status" =~ ^(200|201|409)$ ]]; then
        log_info "✅ 存储桶创建: 成功"
    else
        log_error "❌ 存储桶创建: 失败 (HTTP $bucket_status)"
        return 1
    fi
    
    # 上传测试文件
    local upload_status
    upload_status=$(curl -o /dev/null -s -w "%{http_code}" -X PUT http://localhost:8080/health-check-bucket/test.txt -d "health check test" 2>/dev/null || echo "000")
    
    if [[ "$upload_status" =~ ^(200|201)$ ]]; then
        log_info "✅ 文件上传: 成功"
    else
        log_error "❌ 文件上传: 失败 (HTTP $upload_status)"
        return 1
    fi
    
    # 下载测试文件
    local download_content
    download_content=$(curl -s http://localhost:8080/health-check-bucket/test.txt 2>/dev/null || echo "")
    
    if [ "$download_content" = "health check test" ]; then
        log_info "✅ 文件下载: 成功"
    else
        log_error "❌ 文件下载: 失败"
        return 1
    fi
    
    # 清理测试文件
    curl -o /dev/null -s -X DELETE http://localhost:8080/health-check-bucket/test.txt 2>/dev/null || true
    
    return 0
}

# 检查错误注入功能
test_error_injection() {
    log_step "测试错误注入功能..."
    
    # 获取错误注入规则
    local rules_response
    rules_response=$(curl -s http://localhost:8085/api/v1/rules 2>/dev/null || echo "")
    
    if [ -n "$rules_response" ]; then
        local rules_count
        rules_count=$(echo "$rules_response" | jq 'length' 2>/dev/null || echo "0")
        log_info "✅ 错误注入服务: $rules_count 个规则已配置"
    else
        log_error "❌ 错误注入服务: 无法获取规则"
        return 1
    fi
    
    # 获取统计信息
    local stats_response
    stats_response=$(curl -s http://localhost:8085/api/v1/stats 2>/dev/null || echo "")
    
    if [ -n "$stats_response" ]; then
        log_info "✅ 错误注入统计: 可用"
    else
        log_warn "⚠️  错误注入统计: 不可用"
    fi
    
    return 0
}

# 性能检查
check_performance() {
    log_step "执行性能检查..."
    
    # 检查响应时间
    local start_time end_time duration
    start_time=$(date +%s%N)
    
    curl -o /dev/null -s http://localhost:8080/health >/dev/null 2>&1
    
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 )) # 转换为毫秒
    
    if [ "$duration" -lt 1000 ]; then
        log_info "✅ API 响应时间: ${duration}ms (良好)"
    elif [ "$duration" -lt 5000 ]; then
        log_warn "⚠️  API 响应时间: ${duration}ms (一般)"
    else
        log_error "❌ API 响应时间: ${duration}ms (较慢)"
    fi
    
    # 检查内存使用情况
    local memory_usage
    memory_usage=$(docker stats --no-stream --format "table {{.Container}}\t{{.MemUsage}}" 2>/dev/null | grep -v CONTAINER | head -5)
    
    if [ -n "$memory_usage" ]; then
        log_info "✅ 内存使用情况:"
        echo "$memory_usage" | while read -r line; do
            log_info "  $line"
        done
    else
        log_warn "⚠️  无法获取内存使用情况"
    fi
}

# 生成健康检查报告
generate_report() {
    local containers_ok="$1"
    local services_ok="$2"
    local infrastructure_ok="$3"
    local service_discovery_ok="$4"
    local functionality_ok="$5"
    local error_injection_ok="$6"
    
    echo ""
    echo "=========================================="
    echo "           健康检查报告"
    echo "=========================================="
    echo "检查时间: $(date)"
    echo ""
    
    echo "📊 检查结果概览:"
    [ "$containers_ok" -eq 0 ] && echo "✅ Docker 容器: 正常" || echo "❌ Docker 容器: 异常"
    [ "$services_ok" -eq 0 ] && echo "✅ 微服务: 正常" || echo "❌ 微服务: 异常"
    [ "$infrastructure_ok" -eq 0 ] && echo "✅ 基础设施: 正常" || echo "❌ 基础设施: 异常"
    [ "$service_discovery_ok" -eq 0 ] && echo "✅ 服务发现: 正常" || echo "❌ 服务发现: 异常"
    [ "$functionality_ok" -eq 0 ] && echo "✅ 基本功能: 正常" || echo "❌ 基本功能: 异常"
    [ "$error_injection_ok" -eq 0 ] && echo "✅ 错误注入: 正常" || echo "❌ 错误注入: 异常"
    
    echo ""
    echo "🔗 服务端点:"
    echo "  • S3 API: http://localhost:8080"
    echo "  • Consul UI: http://localhost:8500"
    echo "  • Grafana: http://localhost:3000"
    echo "  • Prometheus: http://localhost:9090"
    echo "  • Kibana: http://localhost:5601"
    
    echo ""
    local total_checks=6
    local passed_checks=0
    
    [ "$containers_ok" -eq 0 ] && passed_checks=$((passed_checks + 1))
    [ "$services_ok" -eq 0 ] && passed_checks=$((passed_checks + 1))
    [ "$infrastructure_ok" -eq 0 ] && passed_checks=$((passed_checks + 1))
    [ "$service_discovery_ok" -eq 0 ] && passed_checks=$((passed_checks + 1))
    [ "$functionality_ok" -eq 0 ] && passed_checks=$((passed_checks + 1))
    [ "$error_injection_ok" -eq 0 ] && passed_checks=$((passed_checks + 1))
    
    echo "📈 总体状态: $passed_checks/$total_checks 检查通过"
    
    if [ "$passed_checks" -eq "$total_checks" ]; then
        echo "🎉 系统状态: 完全健康"
        return 0
    elif [ "$passed_checks" -ge 4 ]; then
        echo "⚠️  系统状态: 基本健康"
        return 1
    else
        echo "❌ 系统状态: 需要注意"
        return 2
    fi
}

# 主函数
main() {
    log_info "开始 MockS3 系统健康检查..."
    echo ""
    
    # 执行各项检查
    check_containers; containers_result=$?
    echo ""
    
    check_services; services_result=$?
    echo ""
    
    check_infrastructure; infrastructure_result=$?
    echo ""
    
    check_service_discovery; service_discovery_result=$?
    echo ""
    
    test_basic_functionality; functionality_result=$?
    echo ""
    
    test_error_injection; error_injection_result=$?
    echo ""
    
    check_performance
    echo ""
    
    # 生成报告
    generate_report "$containers_result" "$services_result" "$infrastructure_result" \
                   "$service_discovery_result" "$functionality_result" "$error_injection_result"
    
    local overall_result=$?
    
    if [ "$overall_result" -eq 0 ]; then
        log_info "健康检查完成: 系统运行正常 🎉"
        exit 0
    elif [ "$overall_result" -eq 1 ]; then
        log_warn "健康检查完成: 系统基本正常，建议检查异常项 ⚠️"
        exit 1
    else
        log_error "健康检查完成: 系统存在问题，需要立即处理 ❌"
        exit 2
    fi
}

# 检查依赖工具
check_dependencies() {
    local missing_tools=()
    
    command -v curl >/dev/null 2>&1 || missing_tools+=("curl")
    command -v jq >/dev/null 2>&1 || missing_tools+=("jq")
    command -v docker >/dev/null 2>&1 || missing_tools+=("docker")
    command -v docker-compose >/dev/null 2>&1 || missing_tools+=("docker-compose")
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "缺少必要工具: ${missing_tools[*]}"
        echo "请安装缺少的工具后重试"
        exit 1
    fi
}

# 信号处理
trap 'log_info "健康检查被中断"; exit 130' INT TERM

# 执行检查
check_dependencies
main "$@"
#!/bin/bash

# MockS3 Gateway 停止脚本
# 用于优雅停止 MockS3 微服务堆栈

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

# 优雅停止服务
graceful_stop() {
    log_step "优雅停止 MockS3 微服务堆栈..."
    
    # 先停止网关，避免新请求进入
    log_info "停止 Nginx Gateway..."
    docker-compose stop nginx-gateway || true
    
    # 停止应用服务
    log_info "停止应用服务..."
    docker-compose stop storage-service third-party-service metadata-service queue-service mock-error-service || true
    
    # 最后停止基础设施服务
    log_info "停止基础设施服务..."
    docker-compose stop consul redis postgres || true
    
    log_info "所有服务已停止"
}

# 强制停止服务
force_stop() {
    log_step "强制停止所有服务..."
    docker-compose down --remove-orphans
    log_info "所有服务已强制停止"
}

# 清理资源
cleanup_resources() {
    log_step "清理资源..."
    
    # 清理未使用的网络
    log_info "清理网络..."
    docker network prune -f || true
    
    # 可选：清理未使用的镜像
    if [ "$CLEANUP_IMAGES" = true ]; then
        log_info "清理未使用的镜像..."
        docker image prune -f || true
    fi
    
    # 可选：清理数据卷
    if [ "$CLEANUP_VOLUMES" = true ]; then
        log_warn "清理数据卷（所有数据将丢失）..."
        docker-compose down -v
        docker volume prune -f || true
    fi
    
    log_info "资源清理完成"
}

# 显示停止状态
show_stop_status() {
    log_step "检查停止状态..."
    
    # 检查容器状态
    running_containers=$(docker-compose ps -q | wc -l)
    
    if [ "$running_containers" -eq 0 ]; then
        log_info "所有容器已停止"
    else
        log_warn "仍有 $running_containers 个容器在运行"
        docker-compose ps
    fi
    
    echo ""
    echo "=== 停止完成 ==="
    echo "• 如需重新启动: ./start.sh"
    echo "• 如需完全清理: ./stop.sh --cleanup-all"
    echo "• 查看日志: docker-compose logs [service-name]"
}

# 显示帮助信息
show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --force           强制停止所有服务"
    echo "  --cleanup-images  清理未使用的 Docker 镜像"
    echo "  --cleanup-volumes 清理数据卷（会丢失所有数据）"
    echo "  --cleanup-all     清理所有资源（镜像+数据卷）"
    echo "  -h, --help        显示帮助信息"
    echo ""
    echo "Examples:"
    echo "  $0                    # 优雅停止服务"
    echo "  $0 --force           # 强制停止所有服务"
    echo "  $0 --cleanup-all     # 停止并清理所有资源"
}

# 主函数
main() {
    # 解析命令行参数
    FORCE_STOP=false
    CLEANUP_IMAGES=false
    CLEANUP_VOLUMES=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                FORCE_STOP=true
                shift
                ;;
            --cleanup-images)
                CLEANUP_IMAGES=true
                shift
                ;;
            --cleanup-volumes)
                CLEANUP_VOLUMES=true
                shift
                ;;
            --cleanup-all)
                CLEANUP_IMAGES=true
                CLEANUP_VOLUMES=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    log_info "开始停止 MockS3 微服务堆栈..."
    
    # 检查 docker-compose 文件是否存在
    if [ ! -f "docker-compose.yml" ]; then
        log_error "未找到 docker-compose.yml 文件"
        exit 1
    fi
    
    # 执行停止操作
    if [ "$FORCE_STOP" = true ]; then
        force_stop
    else
        graceful_stop
    fi
    
    # 清理资源
    cleanup_resources
    
    # 显示状态
    show_stop_status
    
    log_info "MockS3 微服务堆栈停止完成"
}

# 信号处理
trap 'log_info "收到中断信号，强制停止..."; docker-compose down; exit 0' INT TERM

# 执行主函数
main "$@"
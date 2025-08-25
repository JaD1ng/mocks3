#!/bin/bash

# MockS3 配置管理脚本
# 用于管理各个微服务的配置文件

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置路径
CONFIG_DIR="./config"
SERVICES_CONFIG_DIR="$CONFIG_DIR/services"
ENV_CONFIG_DIR="$CONFIG_DIR/env"

# 服务列表
SERVICES=("metadata" "storage" "queue" "third-party" "mock-error")

# 显示帮助信息
show_help() {
    echo -e "${BLUE}MockS3 配置管理脚本${NC}"
    echo ""
    echo "用法: $0 [选项] [命令]"
    echo ""
    echo "命令:"
    echo "  validate      验证所有配置文件"
    echo "  generate      生成配置模板"
    echo "  deploy        部署配置到 Consul KV"
    echo "  backup        备份当前配置"
    echo "  restore       恢复配置"
    echo "  diff          比较配置差异"
    echo "  list          列出所有配置文件"
    echo ""
    echo "选项:"
    echo "  -h, --help    显示帮助信息"
    echo "  -e, --env     指定环境 (development, production)"
    echo "  -s, --service 指定服务名称"
    echo "  -v, --verbose 详细输出"
    echo ""
}

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

# 验证配置文件
validate_config() {
    local config_file=$1
    log_info "验证配置文件: $config_file"
    
    if [ ! -f "$config_file" ]; then
        log_error "配置文件不存在: $config_file"
        return 1
    fi
    
    # 使用 yq 验证 YAML 语法
    if command -v yq &> /dev/null; then
        yq eval '.' "$config_file" > /dev/null
        if [ $? -eq 0 ]; then
            log_info "配置文件语法正确"
        else
            log_error "配置文件语法错误"
            return 1
        fi
    else
        log_warn "未安装 yq，跳过 YAML 语法验证"
    fi
    
    return 0
}

# 验证所有配置文件
validate_all_configs() {
    log_info "开始验证所有配置文件..."
    
    local errors=0
    
    # 验证全局配置
    validate_config "$CONFIG_DIR/global.yaml" || ((errors++))
    
    # 验证环境配置
    for env_config in "$ENV_CONFIG_DIR"/*.yaml; do
        if [ -f "$env_config" ]; then
            validate_config "$env_config" || ((errors++))
        fi
    done
    
    # 验证服务配置
    for service_config in "$SERVICES_CONFIG_DIR"/*.yaml; do
        if [ -f "$service_config" ]; then
            validate_config "$service_config" || ((errors++))
        fi
    done
    
    if [ $errors -eq 0 ]; then
        log_info "所有配置文件验证通过"
        return 0
    else
        log_error "发现 $errors 个配置错误"
        return 1
    fi
}

# 生成配置模板
generate_templates() {
    log_info "生成配置模板..."
    
    # 创建必要的目录
    mkdir -p "$SERVICES_CONFIG_DIR" "$ENV_CONFIG_DIR"
    
    # 为每个服务生成基础配置模板
    for service in "${SERVICES[@]}"; do
        local template_file="$SERVICES_CONFIG_DIR/${service}.yaml.template"
        
        if [ ! -f "$template_file" ]; then
            log_info "生成 $service 服务配置模板"
            cat > "$template_file" << EOF
# $service 服务配置模板

server:
  host: "0.0.0.0"
  port: {{SERVICE_PORT}}
  environment: "{{ENVIRONMENT}}"
  version: "{{VERSION}}"

observability:
  service_name: "${service}-service"
  service_version: "{{VERSION}}"
  environment: "{{ENVIRONMENT}}"
  otlp_endpoint: "{{OTLP_ENDPOINT}}"
  log_level: "{{LOG_LEVEL}}"

consul:
  address: "{{CONSUL_ADDRESS}}"
  enabled: true
  register: true

health:
  check_interval: "30s"
  timeout: "5s"
  endpoint: "/health"
EOF
        fi
    done
    
    log_info "配置模板生成完成"
}

# 部署配置到 Consul KV
deploy_to_consul() {
    local environment=${1:-"development"}
    log_info "部署配置到 Consul KV (环境: $environment)"
    
    if ! command -v consul &> /dev/null; then
        log_error "consul 命令不可用，请确保 Consul 已安装并在 PATH 中"
        return 1
    fi
    
    # 检查 Consul 是否可访问
    if ! consul members &> /dev/null; then
        log_error "无法连接到 Consul，请确保 Consul 服务正在运行"
        return 1
    fi
    
    # 上传全局配置
    consul kv put "mocks3/config/global" @"$CONFIG_DIR/global.yaml"
    
    # 上传环境配置
    if [ -f "$ENV_CONFIG_DIR/${environment}.yaml" ]; then
        consul kv put "mocks3/config/env/$environment" @"$ENV_CONFIG_DIR/${environment}.yaml"
    fi
    
    # 上传服务配置
    for service in "${SERVICES[@]}"; do
        local config_file="$SERVICES_CONFIG_DIR/${service}.yaml"
        if [ -f "$config_file" ]; then
            consul kv put "mocks3/config/services/$service" @"$config_file"
            log_info "已上传 $service 服务配置"
        fi
    done
    
    log_info "配置部署完成"
}

# 备份配置
backup_configs() {
    local backup_dir="./backups/config-$(date +%Y%m%d-%H%M%S)"
    log_info "备份配置到: $backup_dir"
    
    mkdir -p "$backup_dir"
    
    # 备份本地配置文件
    cp -r "$CONFIG_DIR" "$backup_dir/"
    
    # 如果 Consul 可用，备份 Consul KV
    if command -v consul &> /dev/null && consul members &> /dev/null; then
        log_info "备份 Consul KV 配置..."
        consul kv export "mocks3/config/" > "$backup_dir/consul-kv-backup.json"
    fi
    
    log_info "配置备份完成: $backup_dir"
}

# 恢复配置
restore_configs() {
    local backup_dir=$1
    
    if [ -z "$backup_dir" ] || [ ! -d "$backup_dir" ]; then
        log_error "请指定有效的备份目录"
        return 1
    fi
    
    log_info "从 $backup_dir 恢复配置..."
    
    # 恢复本地配置文件
    if [ -d "$backup_dir/config" ]; then
        cp -r "$backup_dir/config"/* "$CONFIG_DIR/"
        log_info "本地配置文件恢复完成"
    fi
    
    # 恢复 Consul KV
    if [ -f "$backup_dir/consul-kv-backup.json" ] && command -v consul &> /dev/null; then
        consul kv import @"$backup_dir/consul-kv-backup.json"
        log_info "Consul KV 配置恢复完成"
    fi
    
    log_info "配置恢复完成"
}

# 比较配置差异
diff_configs() {
    local env1=${1:-"development"}
    local env2=${2:-"production"}
    
    log_info "比较环境配置差异: $env1 vs $env2"
    
    local config1="$ENV_CONFIG_DIR/${env1}.yaml"
    local config2="$ENV_CONFIG_DIR/${env2}.yaml"
    
    if [ ! -f "$config1" ] || [ ! -f "$config2" ]; then
        log_error "配置文件不存在"
        return 1
    fi
    
    if command -v diff &> /dev/null; then
        diff -u "$config1" "$config2" || true
    else
        log_error "diff 命令不可用"
        return 1
    fi
}

# 列出所有配置文件
list_configs() {
    log_info "配置文件列表:"
    
    echo -e "\n${BLUE}全局配置:${NC}"
    ls -la "$CONFIG_DIR"/*.yaml 2>/dev/null || echo "  无全局配置文件"
    
    echo -e "\n${BLUE}环境配置:${NC}"
    ls -la "$ENV_CONFIG_DIR"/*.yaml 2>/dev/null || echo "  无环境配置文件"
    
    echo -e "\n${BLUE}服务配置:${NC}"
    ls -la "$SERVICES_CONFIG_DIR"/*.yaml 2>/dev/null || echo "  无服务配置文件"
}

# 主函数
main() {
    local command=""
    local environment="development"
    local service=""
    local verbose=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -e|--env)
                environment="$2"
                shift 2
                ;;
            -s|--service)
                service="$2"
                shift 2
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            validate|generate|deploy|backup|restore|diff|list)
                command="$1"
                shift
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 执行命令
    case $command in
        validate)
            validate_all_configs
            ;;
        generate)
            generate_templates
            ;;
        deploy)
            deploy_to_consul "$environment"
            ;;
        backup)
            backup_configs
            ;;
        restore)
            restore_configs "$2"
            ;;
        diff)
            diff_configs "$2" "$3"
            ;;
        list)
            list_configs
            ;;
        *)
            log_error "请指定命令"
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
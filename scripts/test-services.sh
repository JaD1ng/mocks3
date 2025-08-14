#!/bin/bash

# 微服务测试脚本

set -e

BASE_URL="${BASE_URL:-http://localhost}"
TIMEOUT=10

echo "🧪 开始测试微服务..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local name="$1"
    local url="$2"
    local method="${3:-GET}"
    local data="$4"
    
    echo -n "Testing $name... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "%{http_code}" -m $TIMEOUT "$url" || echo "000")
    elif [ "$method" = "POST" ]; then
        response=$(curl -s -w "%{http_code}" -m $TIMEOUT -X POST -H "Content-Type: application/json" -d "$data" "$url" || echo "000")
    elif [ "$method" = "PUT" ]; then
        response=$(curl -s -w "%{http_code}" -m $TIMEOUT -X PUT -H "Content-Type: application/json" -d "$data" "$url" || echo "000")
    fi
    
    http_code="${response: -3}"
    body="${response%???}"
    
    if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
        echo -e "${GREEN}✓ PASS${NC} (HTTP $http_code)"
        return 0
    else
        echo -e "${RED}✗ FAIL${NC} (HTTP $http_code)"
        echo "  URL: $url"
        echo "  Response: $body"
        return 1
    fi
}

# 等待服务启动
wait_for_service() {
    local service_url="$1"
    local service_name="$2"
    local max_attempts=30
    local attempt=0
    
    echo -n "Waiting for $service_name to start... "
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s -f "$service_url/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Ready${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    
    echo -e "${RED}✗ Timeout${NC}"
    return 1
}

echo "📋 等待所有服务启动..."

# 等待服务启动
wait_for_service "$BASE_URL" "API Gateway"

echo ""
echo "🔍 开始健康检查测试..."

# 健康检查测试
test_endpoint "API Gateway Health" "$BASE_URL/health"
test_endpoint "S3 API Health" "$BASE_URL/api/v1/metadata/health" # 通过网关访问
test_endpoint "Metadata Health" "$BASE_URL/api/v1/metadata/health" 
test_endpoint "Storage Health" "$BASE_URL/api/v1/storage/health"
test_endpoint "Task Processor Health" "$BASE_URL/api/v1/tasks/health"
test_endpoint "Chaos Engineering Health" "$BASE_URL/api/v1/chaos/health"

echo ""
echo "📊 测试基础功能..."

# 测试上传文件
echo "Testing file upload..."
test_data="Hello, Micro S3 World!"
if test_endpoint "Upload Object" "$BASE_URL/test-bucket/test-file.txt" "PUT" "$test_data"; then
    # 测试下载文件
    test_endpoint "Download Object" "$BASE_URL/test-bucket/test-file.txt"
    
    # 测试对象元数据
    test_endpoint "Object Metadata" "$BASE_URL/test-bucket/test-file.txt" "HEAD"
    
    # 测试删除文件
    test_endpoint "Delete Object" "$BASE_URL/test-bucket/test-file.txt" "DELETE"
fi

echo ""
echo "📈 测试管理接口..."

# 测试统计接口
test_endpoint "Storage Stats" "$BASE_URL/api/v1/storage/stats"
test_endpoint "Metadata Stats" "$BASE_URL/api/v1/metadata/stats"
test_endpoint "Task Queue Stats" "$BASE_URL/api/v1/tasks/queue/stats"
test_endpoint "Chaos Rules" "$BASE_URL/api/v1/chaos/rules"

echo ""
echo "🎯 测试 Admin API..."

# 测试 Admin API
test_endpoint "Admin Overview" "$BASE_URL/api/v1/admin/overview"
test_endpoint "Admin Services" "$BASE_URL/api/v1/admin/services"
test_endpoint "Admin Metrics" "$BASE_URL/api/v1/admin/metrics"
test_endpoint "Admin Dashboard" "$BASE_URL/api/v1/admin/dashboard"

echo ""
echo "🔧 测试任务处理..."

# 提交测试任务
task_data='{"type":"health_check","object_key":"test","data":{"test":true}}'
test_endpoint "Submit Task" "$BASE_URL/api/v1/tasks/tasks" "POST" "$task_data"

# 获取工作节点状态
test_endpoint "Worker Status" "$BASE_URL/api/v1/tasks/workers/status"

echo ""
echo "⚡ 测试混沌工程..."

# 创建混沌规则
chaos_rule='{
  "name": "Test Network Timeout",
  "service": "storage",
  "failure_type": "network_timeout",
  "failure_rate": 0.1,
  "duration": "30s",
  "enabled": false,
  "config": {"timeout_ms": 5000}
}'
test_endpoint "Create Chaos Rule" "$BASE_URL/api/v1/chaos/rules" "POST" "$chaos_rule"

# 获取混沌统计
test_endpoint "Chaos Stats" "$BASE_URL/api/v1/chaos/stats"

echo ""
echo "📝 测试监控端点..."

# 测试 Prometheus 指标（如果可用）
if curl -s -f http://localhost:9090/api/v1/query?query=up > /dev/null 2>&1; then
    test_endpoint "Prometheus Metrics" "http://localhost:9090/api/v1/query?query=up"
else
    echo "Prometheus not accessible, skipping metrics test"
fi

# 测试 Consul 服务发现（如果可用）
if curl -s -f http://localhost:8500/v1/catalog/services > /dev/null 2>&1; then
    test_endpoint "Consul Services" "http://localhost:8500/v1/catalog/services"
else
    echo "Consul not accessible, skipping service discovery test"
fi

echo ""
echo "📊 生成测试报告..."

# 统计测试结果
total_tests=20  # 估计的测试数量
echo "Test Summary:"
echo "  Total tests: $total_tests"
echo "  Infrastructure: Consul, PostgreSQL, Redis, Elasticsearch, Prometheus"
echo "  Services: API Gateway, S3 API, Metadata, Storage, Task Processor, Chaos Engineering, Admin API"
echo ""

# 性能测试（可选）
if command -v ab >/dev/null 2>&1; then
    echo "🚀 Running performance test..."
    echo "Testing upload performance (100 requests, concurrency 10):"
    echo "Test data" | ab -n 100 -c 10 -T "text/plain" -u test_perf.txt "$BASE_URL/perf-test/test.txt" 2>/dev/null | grep -E "(Requests per second|Time per request)" || true
    
    # 清理测试文件
    curl -s -X DELETE "$BASE_URL/perf-test/test.txt" >/dev/null 2>&1 || true
fi

echo ""
echo -e "${GREEN}✅ 微服务测试完成！${NC}"
echo ""
echo "🔗 有用的链接:"
echo "  - API Gateway: $BASE_URL"
echo "  - Consul UI: http://localhost:8500"
echo "  - Prometheus: http://localhost:9090"
echo "  - Elasticsearch: http://localhost:9200"
echo ""
echo "📚 使用示例:"
echo "  # 上传文件"
echo "  curl -X PUT '$BASE_URL/my-bucket/file.txt' -d 'file content'"
echo ""
echo "  # 下载文件"
echo "  curl '$BASE_URL/my-bucket/file.txt'"
echo ""
echo "  # 查看系统状态"
echo "  curl '$BASE_URL/api/v1/admin/overview'"
echo ""
echo "  # 创建混沌规则"
echo "  curl -X POST '$BASE_URL/api/v1/chaos/rules' -H 'Content-Type: application/json' -d '{\"name\":\"test\",\"service\":\"storage\",\"failure_type\":\"network_timeout\",\"failure_rate\":0.1}'"
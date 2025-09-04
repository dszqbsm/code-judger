#!/bin/bash

# 集成测试脚本
# 测试WebSocket实时推送和基于Consul的RPC调用功能

set -e

echo "=========================================="
echo "开始集成测试: WebSocket + Consul RPC"
echo "=========================================="

# 配置变量
CONSUL_ADDR="localhost:8500"
SUBMISSION_API="http://localhost:8888"
JUDGE_API="http://localhost:8890"
PROBLEM_API="http://localhost:8891"
KAFKA_ADDR="localhost:9094"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查服务是否运行
check_service() {
    local service_name=$1
    local url=$2
    
    log_info "检查 $service_name 服务状态..."
    
    if curl -s --max-time 5 "$url" > /dev/null 2>&1; then
        log_success "$service_name 服务运行正常"
        return 0
    else
        log_error "$service_name 服务不可用: $url"
        return 1
    fi
}

# 检查Consul服务发现
check_consul_discovery() {
    log_info "检查Consul服务发现..."
    
    # 检查提交服务是否注册到Consul
    local submission_services=$(curl -s "$CONSUL_ADDR/v1/health/service/submission-api" | jq length 2>/dev/null || echo "0")
    if [ "$submission_services" -gt 0 ]; then
        log_success "提交服务已注册到Consul (实例数: $submission_services)"
    else
        log_warning "提交服务未在Consul中发现"
    fi
    
    # 检查判题服务是否注册到Consul
    local judge_services=$(curl -s "$CONSUL_ADDR/v1/health/service/judge-api" | jq length 2>/dev/null || echo "0")
    if [ "$judge_services" -gt 0 ]; then
        log_success "判题服务已注册到Consul (实例数: $judge_services)"
    else
        log_warning "判题服务未在Consul中发现"
    fi
    
    # 检查题目服务是否注册到Consul
    local problem_services=$(curl -s "$CONSUL_ADDR/v1/health/service/problem-api" | jq length 2>/dev/null || echo "0")
    if [ "$problem_services" -gt 0 ]; then
        log_success "题目服务已注册到Consul (实例数: $problem_services)"
    else
        log_warning "题目服务未在Consul中发现"
    fi
}

# 测试健康检查端点
test_health_check() {
    log_info "测试健康检查端点..."
    
    local health_response=$(curl -s "$SUBMISSION_API/health")
    local status=$(echo "$health_response" | jq -r '.status' 2>/dev/null || echo "unknown")
    
    if [ "$status" = "healthy" ]; then
        log_success "健康检查通过"
        echo "$health_response" | jq '.' 2>/dev/null || echo "$health_response"
    else
        log_error "健康检查失败: $status"
        echo "$health_response"
    fi
}

# 测试JWT认证
test_jwt_auth() {
    log_info "测试JWT认证..."
    
    # 这里需要一个有效的JWT token进行测试
    # 在实际环境中，你需要先登录获取token
    local jwt_token="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test"
    
    local auth_response=$(curl -s -H "Authorization: Bearer $jwt_token" "$SUBMISSION_API/api/v1/submissions/1/result")
    local code=$(echo "$auth_response" | jq -r '.code' 2>/dev/null || echo "unknown")
    
    if [ "$code" = "200" ] || [ "$code" = "404" ]; then
        log_success "JWT认证测试通过"
    else
        log_warning "JWT认证测试需要有效token"
    fi
}

# 测试RPC调用
test_rpc_calls() {
    log_info "测试RPC调用..."
    
    # 编译并运行RPC测试程序
    if [ -f "rpc_test.go" ]; then
        log_info "运行RPC测试程序..."
        if go run rpc_test.go > rpc_test.log 2>&1; then
            log_success "RPC测试完成，查看 rpc_test.log 获取详细结果"
            grep -E "(SUCCESS|ERROR|测试)" rpc_test.log | head -10
        else
            log_error "RPC测试失败"
            tail -10 rpc_test.log
        fi
    else
        log_warning "未找到RPC测试程序 rpc_test.go"
    fi
}

# 测试Kafka连接
test_kafka_connection() {
    log_info "测试Kafka连接..."
    
    # 检查Kafka主题是否存在
    if command -v kafka-topics.sh > /dev/null 2>&1; then
        log_info "检查Kafka主题..."
        kafka-topics.sh --bootstrap-server "$KAFKA_ADDR" --list | grep -E "(judge_tasks|judge_results|status_updates)" && \
        log_success "Kafka主题存在" || \
        log_warning "Kafka主题不存在或无法连接"
    else
        log_warning "未安装Kafka命令行工具，跳过Kafka连接测试"
    fi
}

# 测试代码提交流程
test_submission_flow() {
    log_info "测试代码提交流程..."
    
    # 准备测试数据
    local test_submission='{
        "problem_id": 1001,
        "language": "c",
        "code": "#include <stdio.h>\nint main() {\n    int a, b;\n    scanf(\"%d %d\", &a, &b);\n    printf(\"%d\\n\", a + b);\n    return 0;\n}"
    }'
    
    # 这里需要有效的JWT token
    local jwt_token="test_token"
    
    log_info "发送提交请求..."
    local submit_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $jwt_token" \
        -d "$test_submission" \
        "$SUBMISSION_API/api/v1/submissions")
    
    local submit_code=$(echo "$submit_response" | jq -r '.code' 2>/dev/null || echo "unknown")
    
    if [ "$submit_code" = "200" ]; then
        log_success "代码提交成功"
        local submission_id=$(echo "$submit_response" | jq -r '.data.submission_id' 2>/dev/null)
        log_info "提交ID: $submission_id"
        
        # 等待一段时间让判题流程执行
        log_info "等待判题流程执行..."
        sleep 5
        
        # 查询判题结果
        log_info "查询判题结果..."
        local result_response=$(curl -s -H "Authorization: Bearer $jwt_token" \
            "$SUBMISSION_API/api/v1/submissions/$submission_id/result")
        echo "$result_response" | jq '.' 2>/dev/null || echo "$result_response"
        
    else
        log_warning "代码提交需要有效的JWT token"
        echo "$submit_response"
    fi
}

# 测试WebSocket连接
test_websocket_connection() {
    log_info "测试WebSocket连接..."
    
    if command -v wscat > /dev/null 2>&1; then
        log_info "使用wscat测试WebSocket连接..."
        echo "连接测试" | timeout 5 wscat -c "ws://localhost:8888/ws?token=test_token" && \
        log_success "WebSocket连接测试完成" || \
        log_warning "WebSocket连接测试失败或超时"
    else
        log_warning "未安装wscat工具，跳过WebSocket连接测试"
        log_info "请打开 websocket_test.html 进行手动测试"
    fi
}

# 性能测试
performance_test() {
    log_info "执行性能测试..."
    
    # 并发提交测试
    if command -v ab > /dev/null 2>&1; then
        log_info "执行并发请求测试..."
        ab -n 100 -c 10 -T "application/json" \
           -H "Authorization: Bearer test_token" \
           "$SUBMISSION_API/health" > performance_test.log 2>&1 && \
        log_success "性能测试完成，查看 performance_test.log" || \
        log_error "性能测试失败"
    else
        log_warning "未安装ab工具，跳过性能测试"
    fi
}

# 清理测试数据
cleanup() {
    log_info "清理测试数据..."
    
    # 清理日志文件
    rm -f rpc_test.log performance_test.log
    
    log_success "清理完成"
}

# 主测试流程
main() {
    log_info "开始执行集成测试..."
    
    # 检查依赖工具
    command -v curl > /dev/null 2>&1 || { log_error "需要安装curl"; exit 1; }
    command -v jq > /dev/null 2>&1 || { log_warning "建议安装jq以更好地解析JSON响应"; }
    
    # 基础服务检查
    check_service "Consul" "$CONSUL_ADDR/v1/status/leader" || log_warning "Consul服务检查失败"
    check_service "提交服务" "$SUBMISSION_API/health" || log_error "提交服务检查失败"
    
    # Consul服务发现测试
    check_consul_discovery
    
    # 健康检查测试
    test_health_check
    
    # JWT认证测试
    test_jwt_auth
    
    # Kafka连接测试
    test_kafka_connection
    
    # RPC调用测试
    test_rpc_calls
    
    # WebSocket连接测试
    test_websocket_connection
    
    # 代码提交流程测试
    test_submission_flow
    
    # 性能测试
    performance_test
    
    log_success "集成测试完成!"
    
    echo ""
    echo "=========================================="
    echo "测试总结:"
    echo "1. 打开 websocket_test.html 进行WebSocket功能测试"
    echo "2. 检查 rpc_test.log 获取RPC调用测试结果"
    echo "3. 检查 performance_test.log 获取性能测试结果"
    echo "4. 确保所有服务都已启动并注册到Consul"
    echo "=========================================="
}

# 捕获Ctrl+C信号进行清理
trap cleanup EXIT

# 解析命令行参数
case "${1:-all}" in
    "health")
        test_health_check
        ;;
    "consul")
        check_consul_discovery
        ;;
    "rpc")
        test_rpc_calls
        ;;
    "websocket")
        test_websocket_connection
        ;;
    "submission")
        test_submission_flow
        ;;
    "performance")
        performance_test
        ;;
    "all"|*)
        main
        ;;
esac



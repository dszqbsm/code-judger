#!/bin/bash

# 提交服务API测试脚本
# Submission API Testing Script

echo "🧪 开始提交服务API测试"
echo "=================================="

BASE_URL="http://localhost:8889"
API_PREFIX="/api/v1"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    ((PASSED_TESTS++))
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((FAILED_TESTS++))
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 测试函数
test_api() {
    local test_name="$1"
    local method="$2"
    local url="$3"
    local data="$4"
    local expected_status="$5"
    
    ((TOTAL_TESTS++))
    
    log_info "测试: $test_name"
    log_info "请求: $method $url"
    
    if [[ "$method" == "GET" ]]; then
        response=$(curl -s -w "\n%{http_code}" "$url")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$url")
    fi
    
    # 分离响应体和状态码
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | sed '$d')
    
    echo "响应状态码: $http_code"
    echo "响应内容: $response_body"
    
    if [[ "$http_code" == "$expected_status" ]]; then
        log_success "$test_name - 状态码正确 ($http_code)"
    else
        log_error "$test_name - 状态码错误 (期望: $expected_status, 实际: $http_code)"
    fi
    
    echo "--------------------------------"
}

# 1. 测试健康检查
log_info "1. 测试健康检查接口"
test_api "健康检查" "GET" "${BASE_URL}/health" "" "200"

# 2. 测试创建提交接口
log_info "2. 测试创建提交接口"
submit_data='{
    "problem_id": 1001,
    "language": "cpp",
    "code": "#include<iostream>\nusing namespace std;\nint main(){\n    int a, b;\n    cin >> a >> b;\n    cout << a + b << endl;\n    return 0;\n}",
    "is_shared": false
}'
test_api "创建提交" "POST" "${BASE_URL}${API_PREFIX}/submissions" "$submit_data" "200"

# 3. 测试获取提交详情接口
log_info "3. 测试获取提交详情接口"
test_api "获取提交详情" "GET" "${BASE_URL}${API_PREFIX}/submissions/1" "" "200"

# 4. 测试获取提交列表接口
log_info "4. 测试获取提交列表接口"
test_api "获取提交列表" "GET" "${BASE_URL}${API_PREFIX}/submissions" "" "200"

# 5. 测试获取提交列表(带分页参数)
log_info "5. 测试获取提交列表(带分页参数)"
test_api "获取提交列表-分页" "GET" "${BASE_URL}${API_PREFIX}/submissions?page=1&page_size=10" "" "200"

# 6. 测试获取不存在的提交
log_info "6. 测试获取不存在的提交"
test_api "获取不存在的提交" "GET" "${BASE_URL}${API_PREFIX}/submissions/99999" "" "404"

# 7. 测试无效的创建提交请求
log_info "7. 测试无效的创建提交请求"
invalid_data='{
    "problem_id": "",
    "language": "",
    "code": ""
}'
test_api "无效创建提交" "POST" "${BASE_URL}${API_PREFIX}/submissions" "$invalid_data" "400"

# 8. 测试WebSocket连接（基础检查）
log_info "8. 测试WebSocket端点可达性"
test_api "WebSocket端点" "GET" "${BASE_URL}/ws/submissions/1/status" "" "400"

echo "=================================="
echo "📊 测试结果统计"
echo "总测试数: $TOTAL_TESTS"
echo -e "通过: ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败: ${RED}$FAILED_TESTS${NC}"

if [[ $FAILED_TESTS -eq 0 ]]; then
    echo -e "${GREEN}🎉 所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}❌ 有 $FAILED_TESTS 个测试失败${NC}"
    exit 1
fi

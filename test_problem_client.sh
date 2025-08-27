#!/bin/bash

# 题目服务客户端测试脚本
# 用于验证判题服务是否能正确调用题目服务

set -e

echo "=== 题目服务客户端集成测试 ==="

# 定义颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# 配置信息
JUDGE_API_URL="http://localhost:8889"
PROBLEM_API_URL="http://localhost:8888"

echo -e "${YELLOW}测试环境配置:${NC}"
echo "- 判题服务地址: $JUDGE_API_URL"
echo "- 题目服务地址: $PROBLEM_API_URL"
echo ""

# 测试函数
test_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="$3"
    
    echo -n "测试 $name ... "
    
    if response=$(curl -s -w "%{http_code}" -o /dev/null "$url"); then
        if [ "$response" = "$expected_status" ]; then
            echo -e "${GREEN}✓${NC} (状态码: $response)"
            return 0
        else
            echo -e "${RED}✗${NC} (期望: $expected_status, 实际: $response)"
            return 1
        fi
    else
        echo -e "${RED}✗${NC} (连接失败)"
        return 1
    fi
}

# 测试提交判题任务
test_submit_judge() {
    echo -n "测试提交判题任务 ... "
    
    # 测试数据
    local test_data='{
        "submission_id": 1001,
        "problem_id": 101,
        "user_id": 123,
        "language": "cpp",
        "code": "#include <iostream>\nusing namespace std;\n\nint main() {\n    int a, b;\n    cin >> a >> b;\n    cout << a + b << endl;\n    return 0;\n}"
    }'
    
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_data" \
        -w "%{http_code}" \
        -o /tmp/judge_response.json \
        "$JUDGE_API_URL/api/v1/judge/submit"); then
        
        if [ "$response" = "200" ]; then
            echo -e "${GREEN}✓${NC} (状态码: $response)"
            echo "响应内容:"
            cat /tmp/judge_response.json | jq . 2>/dev/null || cat /tmp/judge_response.json
            echo ""
            return 0
        else
            echo -e "${RED}✗${NC} (状态码: $response)"
            echo "错误响应:"
            cat /tmp/judge_response.json
            echo ""
            return 1
        fi
    else
        echo -e "${RED}✗${NC} (连接失败)"
        return 1
    fi
}

# 1. 检查服务可用性
echo -e "${YELLOW}1. 检查服务可用性${NC}"
test_endpoint "判题服务健康检查" "$JUDGE_API_URL/api/v1/judge/health" "200"
test_endpoint "题目服务健康检查" "$PROBLEM_API_URL/api/v1/health" "200"
echo ""

# 2. 检查题目服务API
echo -e "${YELLOW}2. 检查题目服务API${NC}"
test_endpoint "获取题目详情" "$PROBLEM_API_URL/api/v1/problems/101" "200"
echo ""

# 3. 测试判题提交（这会触发题目服务调用）
echo -e "${YELLOW}3. 测试判题提交（验证题目服务集成）${NC}"
test_submit_judge
echo ""

# 4. 检查判题服务其他接口
echo -e "${YELLOW}4. 检查判题服务其他接口${NC}"
test_endpoint "获取支持的编程语言" "$JUDGE_API_URL/api/v1/judge/languages" "200"
test_endpoint "获取判题节点状态" "$JUDGE_API_URL/api/v1/judge/nodes" "200"
test_endpoint "获取判题队列状态" "$JUDGE_API_URL/api/v1/judge/queue" "200"
echo ""

# 5. 测试错误情况
echo -e "${YELLOW}5. 测试错误情况${NC}"

# 测试不存在的题目
echo -n "测试不存在的题目 ... "
test_data_invalid='{
    "submission_id": 2001,
    "problem_id": 999999,
    "user_id": 123,
    "language": "cpp",
    "code": "int main(){return 0;}"
}'

if response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$test_data_invalid" \
    -w "%{http_code}" \
    -o /tmp/judge_error_response.json \
    "$JUDGE_API_URL/api/v1/judge/submit"); then
    
    if [ "$response" = "200" ]; then
        # 检查是否返回了错误信息
        if grep -q "题目不存在\|获取题目信息失败" /tmp/judge_error_response.json; then
            echo -e "${GREEN}✓${NC} (正确处理了不存在的题目)"
        else
            echo -e "${YELLOW}?${NC} (响应需要检查)"
            cat /tmp/judge_error_response.json
        fi
    else
        echo -e "${GREEN}✓${NC} (状态码: $response，符合预期的错误响应)"
    fi
else
    echo -e "${RED}✗${NC} (连接失败)"
fi
echo ""

# 清理临时文件
rm -f /tmp/judge_response.json /tmp/judge_error_response.json

echo -e "${YELLOW}=== 测试完成 ===${NC}"
echo ""
echo -e "${YELLOW}说明:${NC}"
echo "✓ 如果看到绿色的勾号，说明该项测试通过"
echo "✗ 如果看到红色的叉号，说明该项测试失败"
echo "? 如果看到黄色的问号，说明需要人工检查结果"
echo ""
echo -e "${YELLOW}重要提示:${NC}"
echo "1. 确保题目服务 (localhost:8888) 已启动"
echo "2. 确保判题服务 (localhost:8889) 已启动"
echo "3. 确保题目服务中存在ID为101的题目"
echo "4. 检查服务日志以确认题目服务调用是否成功"
echo ""
echo "服务日志查看命令:"
echo "docker logs -f oj-judge-api"   # 如果使用Docker
echo "或查看配置的日志文件路径"


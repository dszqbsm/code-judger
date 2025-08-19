#!/bin/bash

# 判题服务接口测试脚本
BASE_URL="http://localhost:8889/api/v1/judge"

echo "=== 判题服务接口功能测试 ==="
echo

# 1. 健康检查
echo "1. 测试健康检查接口"
echo "GET $BASE_URL/health"
curl -s -X GET "$BASE_URL/health" | jq . 2>/dev/null || curl -s -X GET "$BASE_URL/health"
echo -e "\n"

# 2. 获取支持的语言
echo "2. 测试获取支持语言接口"
echo "GET $BASE_URL/languages"
curl -s -X GET "$BASE_URL/languages" | jq . 2>/dev/null || curl -s -X GET "$BASE_URL/languages"
echo -e "\n"

# 3. 获取判题节点状态
echo "3. 测试获取判题节点状态接口"
echo "GET $BASE_URL/nodes"
curl -s -X GET "$BASE_URL/nodes" | jq . 2>/dev/null || curl -s -X GET "$BASE_URL/nodes"
echo -e "\n"

# 4. 获取判题队列状态
echo "4. 测试获取判题队列状态接口"
echo "GET $BASE_URL/queue"
curl -s -X GET "$BASE_URL/queue" | jq . 2>/dev/null || curl -s -X GET "$BASE_URL/queue"
echo -e "\n"

# 5. 提交判题任务
echo "5. 测试提交判题任务接口"
echo "POST $BASE_URL/submit"
SUBMIT_DATA='{
  "submission_id": 12345,
  "problem_id": 1001,
  "user_id": 101,
  "language": "cpp",
  "code": "#include <iostream>\nusing namespace std;\nint main() {\n    int a, b;\n    cin >> a >> b;\n    cout << a + b << endl;\n    return 0;\n}",
  "time_limit": 1000,
  "memory_limit": 128,
  "test_cases": [
    {
      "case_id": 1,
      "input": "1 2",
      "expected_output": "3"
    },
    {
      "case_id": 2,
      "input": "5 7",
      "expected_output": "12"
    }
  ]
}'
echo "$SUBMIT_DATA" | curl -s -X POST "$BASE_URL/submit" \
  -H "Content-Type: application/json" \
  -d @- | jq . 2>/dev/null || echo "$SUBMIT_DATA" | curl -s -X POST "$BASE_URL/submit" \
  -H "Content-Type: application/json" \
  -d @-
echo -e "\n"

# 6. 查询判题结果
echo "6. 测试查询判题结果接口"
echo "GET $BASE_URL/result/12345"
curl -s -X GET "$BASE_URL/result/12345" | jq . 2>/dev/null || curl -s -X GET "$BASE_URL/result/12345"
echo -e "\n"

# 7. 查询判题状态
echo "7. 测试查询判题状态接口"
echo "GET $BASE_URL/status/12345"
curl -s -X GET "$BASE_URL/status/12345" | jq . 2>/dev/null || curl -s -X GET "$BASE_URL/status/12345"
echo -e "\n"

# 8. 取消判题任务
echo "8. 测试取消判题任务接口"
echo "DELETE $BASE_URL/cancel/12345"
curl -s -X DELETE "$BASE_URL/cancel/12345" | jq . 2>/dev/null || curl -s -X DELETE "$BASE_URL/cancel/12345"
echo -e "\n"

# 9. 重新判题
echo "9. 测试重新判题接口"
echo "POST $BASE_URL/rejudge/12345"
curl -s -X POST "$BASE_URL/rejudge/12345" | jq . 2>/dev/null || curl -s -X POST "$BASE_URL/rejudge/12345"
echo -e "\n"

echo "=== 接口测试完成 ==="

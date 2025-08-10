#!/bin/bash

# 文件名：verify-env.sh
# 用途：在线判题系统开发环境完整性验证脚本，全面检测所有服务的功能状态
# 创建日期：2024-01-15
# 版本：v1.0
# 说明：执行多维度测试验证开发环境是否正常工作，包括服务状态、网络连通性、功能测试等
# 依赖：运行中的Docker Compose服务栈，各服务的健康检查接口
#
# 验证范围：
# 1. 容器运行状态检查
# 2. 端口连通性测试
# 3. HTTP服务健康检查
# 4. 数据库连接和基本操作验证
# 5. 缓存服务读写功能测试
# 6. 消息队列收发功能测试
# 7. 日志系统完整性检查
# 8. 监控系统功能验证
# 9. 服务间网络连通性测试

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 计数器
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 打印函数
print_header() {
    echo -e "${BLUE}==================== $1 ====================${NC}"
}

print_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    PASSED_TESTS=$((PASSED_TESTS + 1))
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    FAILED_TESTS=$((FAILED_TESTS + 1))
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# 验证函数
test_service_running() {
    local service_name=$1
    local container_name=$2
    
    print_test "验证 $service_name 容器运行状态"
    if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
        print_pass "$service_name 容器正在运行"
        return 0
    else
        print_fail "$service_name 容器未运行"
        return 1
    fi
}

test_port_accessible() {
    local service_name=$1
    local port=$2
    
    print_test "验证 $service_name 端口 $port 可访问性"
    if timeout 5 bash -c "</dev/tcp/localhost/$port"; then
        print_pass "$service_name 端口 $port 可访问"
        return 0
    else
        print_fail "$service_name 端口 $port 不可访问"
        return 1
    fi
}

test_http_endpoint() {
    local service_name=$1
    local url=$2
    local expected_code=${3:-200}
    
    print_test "验证 $service_name HTTP 端点: $url"
    local response_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    
    if [ "$response_code" -eq "$expected_code" ]; then
        print_pass "$service_name HTTP 端点正常 (HTTP $response_code)"
        return 0
    else
        print_fail "$service_name HTTP 端点异常 (HTTP $response_code)"
        return 1
    fi
}

# MySQL 验证
verify_mysql() {
    print_header "MySQL 数据库验证"
    
    test_service_running "MySQL" "oj-mysql"
    test_port_accessible "MySQL" "3306"
    
    print_test "验证 MySQL 数据库连接"
    if docker exec oj-mysql mysqladmin ping -h localhost --silent 2>/dev/null; then
        print_pass "MySQL 数据库连接正常"
    else
        print_fail "MySQL 数据库连接失败"
        return 1
    fi
    
    print_test "验证数据库和表结构"
    local table_count=$(docker exec oj-mysql mysql -u oj_user -poj_password oj_system -e "SHOW TABLES;" 2>/dev/null | wc -l)
    if [ "$table_count" -gt 5 ]; then
        print_pass "数据库表结构正常 ($((table_count-1)) 个表)"
    else
        print_fail "数据库表结构异常"
        return 1
    fi
    
    print_test "验证初始数据"
    local user_count=$(docker exec oj-mysql mysql -u oj_user -poj_password oj_system -e "SELECT COUNT(*) FROM users;" 2>/dev/null | tail -1)
    if [ "$user_count" -ge 3 ]; then
        print_pass "初始用户数据正常 ($user_count 个用户)"
    else
        print_fail "初始用户数据异常"
        return 1
    fi
}

# Redis 验证
verify_redis() {
    print_header "Redis 缓存验证"
    
    test_service_running "Redis" "oj-redis"
    test_port_accessible "Redis" "6379"
    
    print_test "验证 Redis 连接"
    if docker exec oj-redis redis-cli ping 2>/dev/null | grep -q "PONG"; then
        print_pass "Redis 连接正常"
    else
        print_fail "Redis 连接失败"
        return 1
    fi
    
    print_test "验证 Redis 读写操作"
    if docker exec oj-redis redis-cli set test_key "test_value" >/dev/null 2>&1 && \
       [ "$(docker exec oj-redis redis-cli get test_key 2>/dev/null)" = "test_value" ]; then
        print_pass "Redis 读写操作正常"
        docker exec oj-redis redis-cli del test_key >/dev/null 2>&1
    else
        print_fail "Redis 读写操作失败"
        return 1
    fi
}

# Kafka 验证
verify_kafka() {
    print_header "Kafka 消息队列验证"
    
    test_service_running "Zookeeper" "oj-zookeeper"
    test_service_running "Kafka" "oj-kafka"
    test_port_accessible "Kafka" "9094"
    
    print_test "验证 Kafka 集群状态"
    if docker exec oj-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 >/dev/null 2>&1; then
        print_pass "Kafka 集群状态正常"
    else
        print_fail "Kafka 集群状态异常"
        return 1
    fi
    
    print_test "验证主题创建和消息收发"
    local test_topic="verify-test-topic"
    local test_message="test-message-$(date +%s)"
    
    # 创建测试主题
    docker exec oj-kafka kafka-topics --create --topic "$test_topic" --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1 >/dev/null 2>&1
    
    # 发送消息
    echo "$test_message" | docker exec -i oj-kafka kafka-console-producer --topic "$test_topic" --bootstrap-server localhost:9092 >/dev/null 2>&1
    
    # 接收消息
    local received_message=$(timeout 5 docker exec oj-kafka kafka-console-consumer --topic "$test_topic" --from-beginning --bootstrap-server localhost:9092 --max-messages 1 2>/dev/null || echo "")
    
    if [ "$received_message" = "$test_message" ]; then
        print_pass "Kafka 消息收发正常"
    else
        print_fail "Kafka 消息收发异常"
    fi
    
    # 清理测试主题
    docker exec oj-kafka kafka-topics --delete --topic "$test_topic" --bootstrap-server localhost:9092 >/dev/null 2>&1
}

# ELK Stack 验证
verify_elk() {
    print_header "ELK Stack 日志系统验证"
    
    test_service_running "Elasticsearch" "oj-elasticsearch"
    test_service_running "Logstash" "oj-logstash"
    test_service_running "Kibana" "oj-kibana"
    
    test_http_endpoint "Elasticsearch" "http://localhost:9200"
    test_http_endpoint "Kibana" "http://localhost:5601"
    
    print_test "验证 Elasticsearch 集群健康状态"
    local cluster_status=$(curl -s "http://localhost:9200/_cluster/health" | python3 -c "import sys, json; print(json.load(sys.stdin)['status'])" 2>/dev/null || echo "unknown")
    
    if [ "$cluster_status" = "green" ] || [ "$cluster_status" = "yellow" ]; then
        print_pass "Elasticsearch 集群状态: $cluster_status"
    else
        print_fail "Elasticsearch 集群状态异常: $cluster_status"
    fi
}

# Consul 验证
verify_consul() {
    print_header "Consul 服务注册中心验证"
    
    test_service_running "Consul" "oj-consul"
    test_http_endpoint "Consul" "http://localhost:8500/v1/status/leader"
    
    print_test "验证 Consul 集群状态"
    if curl -s "http://localhost:8500/v1/status/leader" | grep -q '"'; then
        print_pass "Consul 集群状态正常"
    else
        print_fail "Consul 集群状态异常"
        return 1
    fi
}

# 监控系统验证
verify_monitoring() {
    print_header "监控系统验证"
    
    test_service_running "Prometheus" "oj-prometheus"
    test_service_running "Grafana" "oj-grafana"
    
    test_http_endpoint "Prometheus" "http://localhost:9090/-/healthy"
    test_http_endpoint "Grafana" "http://localhost:3000/api/health"
    
    print_test "验证 Prometheus 目标状态"
    local targets_up=$(curl -s "http://localhost:9090/api/v1/query?query=up" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    result = data['data']['result']
    up_count = sum(1 for r in result if r['value'][1] == '1')
    print(up_count)
except:
    print(0)
" 2>/dev/null || echo "0")
    
    if [ "$targets_up" -gt 0 ]; then
        print_pass "Prometheus 监控目标状态正常 ($targets_up 个目标在线)"
    else
        print_fail "Prometheus 监控目标状态异常"
    fi
}

# 网络连通性验证
verify_network() {
    print_header "网络连通性验证"
    
    print_test "验证容器间网络连通性"
    
    # 测试 MySQL 到 Redis
    if docker exec oj-mysql ping -c 1 oj-redis >/dev/null 2>&1; then
        print_pass "MySQL 到 Redis 网络连通"
    else
        print_fail "MySQL 到 Redis 网络不通"
    fi
    
    # 测试 Kafka 到 Zookeeper
    if docker exec oj-kafka ping -c 1 oj-zookeeper >/dev/null 2>&1; then
        print_pass "Kafka 到 Zookeeper 网络连通"
    else
        print_fail "Kafka 到 Zookeeper 网络不通"
    fi
}

# 生成报告
generate_report() {
    print_header "验证报告"
    
    echo -e "${BLUE}总测试数:${NC} $TOTAL_TESTS"
    echo -e "${GREEN}通过测试:${NC} $PASSED_TESTS"
    echo -e "${RED}失败测试:${NC} $FAILED_TESTS"
    
    local success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo -e "${BLUE}成功率:${NC} ${success_rate}%"
    
    if [ "$FAILED_TESTS" -eq 0 ]; then
        echo -e "${GREEN}🎉 所有测试通过！开发环境完全正常！${NC}"
        return 0
    else
        echo -e "${RED}❌ 有 $FAILED_TESTS 个测试失败，请检查相关服务！${NC}"
        return 1
    fi
}

# 主函数
main() {
    echo -e "${BLUE}==================== 在线判题系统环境验证 ====================${NC}"
    echo
    
    # 检查 Docker Compose 是否运行
    if ! docker-compose ps | grep -q "Up"; then
        echo -e "${RED}[ERROR]${NC} 没有检测到运行中的服务，请先启动开发环境："
        echo "  ./start-dev-env.sh"
        exit 1
    fi
    
    verify_mysql
    echo
    verify_redis
    echo
    verify_kafka
    echo
    verify_elk
    echo
    verify_consul
    echo
    verify_monitoring
    echo
    verify_network
    echo
    generate_report
}

# 脚本参数处理
case "${1:-}" in
    --help|-h)
        echo "用法: $0 [选项]"
        echo
        echo "选项:"
        echo "  --help, -h     显示帮助信息"
        echo "  --quick        快速验证（仅检查服务状态）"
        echo
        exit 0
        ;;
    --quick)
        echo -e "${BLUE}==================== 快速环境验证 ====================${NC}"
        echo
        docker-compose ps
        echo
        echo -e "${GREEN}如需详细验证，请运行: $0${NC}"
        exit 0
        ;;
    "")
        main
        ;;
    *)
        echo -e "${RED}[ERROR]${NC} 未知参数: $1"
        echo "使用 --help 查看帮助信息"
        exit 1
        ;;
esac
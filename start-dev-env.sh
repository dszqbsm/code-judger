#!/bin/bash

# 文件名：start-dev-env.sh
# 用途：在线判题系统开发环境一键启动脚本，自动化处理环境配置和服务启动
# 创建日期：2024-01-15
# 版本：v1.0
# 说明：智能检测系统环境，配置Docker参数，启动所有微服务，并验证服务健康状态
# 依赖：Docker 20.0+, Docker Compose 2.0+, 系统内存4GB+
#
# 主要功能：
# 1. 系统环境检查（Docker、内存、权限等）
# 2. Elasticsearch系统参数配置（vm.max_map_count）
# 3. Docker镜像拉取和服务启动
# 4. 服务健康状态检查和等待
# 5. 显示服务访问地址和管理命令

set -e  # 遇到错误立即退出，确保脚本执行安全性

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查系统要求
check_requirements() {
    print_status "检查系统要求..."
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    # 检查 Docker 是否运行
    if ! docker info &> /dev/null; then
        print_error "Docker 服务未启动，请启动 Docker 服务"
        exit 1
    fi
    
    # 检查可用内存
    if [ "$(uname)" == "Linux" ]; then
        available_memory=$(free -m | awk 'NR==2{printf "%.0f", $7}')
        if [ "$available_memory" -lt 2048 ]; then
            print_warning "可用内存少于 2GB，可能会影响服务启动"
        fi
    fi
    
    print_success "系统要求检查通过"
}

# 设置 vm.max_map_count（Elasticsearch 需要）
setup_elasticsearch() {
    print_status "配置 Elasticsearch 系统参数..."
    
    if [ "$(uname)" == "Linux" ]; then
        current_vm_max_map_count=$(sysctl vm.max_map_count | cut -d' ' -f3)
        if [ "$current_vm_max_map_count" -lt 262144 ]; then
            print_status "设置 vm.max_map_count=262144"
            sudo sysctl -w vm.max_map_count=262144
            
            # 询问是否永久设置
            read -p "是否永久设置 vm.max_map_count? (y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                echo 'vm.max_map_count=262144' | sudo tee -a /etc/sysctl.conf
                print_success "已永久设置 vm.max_map_count"
            fi
        fi
    fi
}

# 创建必要的目录
create_directories() {
    print_status "创建必要的目录..."
    
    directories=(
        "docker/mysql/init"
        "docker/mysql/conf"
        "docker/redis"
        "docker/logstash/config"
        "docker/logstash/pipeline"
        "docker/kibana/config"
        "docker/consul/config"
        "docker/prometheus"
        "docker/grafana/provisioning/datasources"
        "docker/grafana/provisioning/dashboards"
        "docker/grafana/dashboards"
    )
    
    for dir in "${directories[@]}"; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            print_status "创建目录: $dir"
        fi
    done
    
    print_success "目录创建完成"
}

# 拉取镜像
pull_images() {
    print_status "拉取 Docker 镜像..."
    docker-compose pull
    print_success "镜像拉取完成"
}

# 启动服务
start_services() {
    print_status "启动服务..."
    docker-compose up -d
    
    print_status "等待服务启动..."
    sleep 10
    
    # 检查服务状态
    print_status "检查服务状态..."
    docker-compose ps
}

# 等待服务就绪
wait_for_services() {
    print_status "等待关键服务就绪..."
    
    # 等待 MySQL
    print_status "等待 MySQL 启动..."
    while ! docker exec oj-mysql mysqladmin ping -h"localhost" --silent; do
        sleep 2
    done
    print_success "MySQL 已就绪"
    
    # 等待 Redis
    print_status "等待 Redis 启动..."
    while ! docker exec oj-redis redis-cli ping > /dev/null 2>&1; do
        sleep 2
    done
    print_success "Redis 已就绪"
    
    # 等待 Elasticsearch
    print_status "等待 Elasticsearch 启动..."
    while ! curl -s http://localhost:9200/_cluster/health > /dev/null 2>&1; do
        sleep 5
    done
    print_success "Elasticsearch 已就绪"
    
    # 等待 Consul
    print_status "等待 Consul 启动..."
    while ! curl -s http://localhost:8500/v1/status/leader > /dev/null 2>&1; do
        sleep 2
    done
    print_success "Consul 已就绪"
    
    print_success "所有关键服务已就绪"
}

# 显示访问信息
show_access_info() {
    echo
    print_success "==================== 在线判题系统开发环境启动成功 ===================="
    echo
    echo -e "${BLUE}📊 服务访问地址:${NC}"
    echo "  🗄️  MySQL:          localhost:3306 (用户: oj_user, 密码: oj_password)"
    echo "  💾 Redis:           localhost:6379"
    echo "  📨 Kafka:           localhost:9094"
    echo "  🎛️  Kafka UI:        http://localhost:8080"
    echo "  🔍 Elasticsearch:   http://localhost:9200"
    echo "  📈 Kibana:          http://localhost:5601"
    echo "  🏛️  Consul:          http://localhost:8500"
    echo "  📊 Prometheus:      http://localhost:9090"
    echo "  📉 Grafana:         http://localhost:3000 (用户: admin, 密码: oj_grafana_admin)"
    echo
    echo -e "${BLUE}🔧 管理命令:${NC}"
    echo "  查看服务状态:   docker-compose ps"
    echo "  查看服务日志:   docker-compose logs -f [service_name]"
    echo "  停止所有服务:   docker-compose down"
    echo "  重启服务:       docker-compose restart [service_name]"
    echo
    echo -e "${BLUE}📚 文档:${NC}"
    echo "  详细文档: ./DOCKER_SETUP.md"
    echo
    echo -e "${GREEN}🎉 开发环境已准备就绪，祝您开发愉快！${NC}"
    echo "=================================================================="
}

# 主函数
main() {
    echo -e "${BLUE}==================== 在线判题系统开发环境启动 ====================${NC}"
    echo
    
    check_requirements
    setup_elasticsearch
    create_directories
    pull_images
    start_services
    wait_for_services
    show_access_info
}

# 脚本参数处理
case "${1:-}" in
    --help|-h)
        echo "用法: $0 [选项]"
        echo
        echo "选项:"
        echo "  --help, -h     显示帮助信息"
        echo "  --pull-only    仅拉取镜像，不启动服务"
        echo "  --no-wait      启动服务后不等待服务就绪"
        echo
        exit 0
        ;;
    --pull-only)
        check_requirements
        pull_images
        print_success "镜像拉取完成"
        exit 0
        ;;
    --no-wait)
        check_requirements
        setup_elasticsearch
        create_directories
        start_services
        show_access_info
        exit 0
        ;;
    "")
        main
        ;;
    *)
        print_error "未知参数: $1"
        print_status "使用 --help 查看帮助信息"
        exit 1
        ;;
esac
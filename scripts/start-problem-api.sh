#!/bin/bash

# 题目服务启动脚本
# 用途：启动题目管理微服务
# 作者：OJ Team
# 日期：2024-01-15

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
SERVICE_NAME="problem-api"
SERVICE_DIR="services/problem-api"
CONFIG_FILE="etc/problem-api.yaml"
LOG_DIR="logs"
PID_FILE="/tmp/${SERVICE_NAME}.pid"

# 打印带颜色的日志
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

# 检查依赖
check_dependencies() {
    log_info "检查依赖服务..."
    
    # 检查MySQL连接
    if ! mysqladmin ping -h mysql -u oj_user -poj_password --silent; then
        log_error "MySQL连接失败，请确保MySQL服务正在运行"
        exit 1
    fi
    
    # 检查Redis连接
    if ! redis-cli -h redis ping > /dev/null 2>&1; then
        log_error "Redis连接失败，请确保Redis服务正在运行"
        exit 1
    fi
    
    log_success "依赖服务检查通过"
}

# 检查服务是否已运行
check_service_status() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            log_warning "服务已在运行中 (PID: $pid)"
            return 0
        else
            log_info "删除过期的PID文件"
            rm -f "$PID_FILE"
        fi
    fi
    return 1
}

# 停止现有服务
stop_service() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            log_info "停止现有服务 (PID: $pid)..."
            kill -TERM "$pid"
            
            # 等待服务优雅关闭
            local count=0
            while ps -p "$pid" > /dev/null 2>&1 && [[ $count -lt 30 ]]; do
                sleep 1
                ((count++))
            done
            
            # 如果仍未关闭，强制终止
            if ps -p "$pid" > /dev/null 2>&1; then
                log_warning "强制终止服务"
                kill -KILL "$pid"
            fi
            
            rm -f "$PID_FILE"
            log_success "服务已停止"
        fi
    fi
}

# 初始化数据库
init_database() {
    log_info "初始化数据库..."
    
    # 检查数据库是否已存在
    if mysql -h mysql -u oj_user -poj_password -e "USE oj_problems;" > /dev/null 2>&1; then
        log_info "数据库已存在，跳过初始化"
        return 0
    fi
    
    # 执行数据库初始化脚本
    if [[ -f "sql/problems_init.sql" ]]; then
        mysql -h mysql -u root -poj_password < sql/problems_init.sql
        log_success "数据库初始化完成"
    else
        log_error "数据库初始化脚本不存在"
        exit 1
    fi
}

# 构建服务
build_service() {
    log_info "构建${SERVICE_NAME}服务..."
    
    cd "$SERVICE_DIR"
    
    # 检查go.mod文件
    if [[ ! -f "go.mod" ]]; then
        log_error "go.mod文件不存在"
        exit 1
    fi
    
    # 下载依赖
    go mod tidy
    go mod download
    
    # 构建可执行文件
    go build -o bin/${SERVICE_NAME} main.go
    
    if [[ $? -eq 0 ]]; then
        log_success "服务构建成功"
    else
        log_error "服务构建失败"
        exit 1
    fi
    
    cd - > /dev/null
}

# 启动服务
start_service() {
    log_info "启动${SERVICE_NAME}服务..."
    
    cd "$SERVICE_DIR"
    
    # 创建日志目录
    mkdir -p "$LOG_DIR"
    
    # 检查配置文件
    if [[ ! -f "$CONFIG_FILE" ]]; then
        log_error "配置文件不存在: $CONFIG_FILE"
        exit 1
    fi
    
    # 启动服务
    nohup ./bin/${SERVICE_NAME} -f "$CONFIG_FILE" > "$LOG_DIR/${SERVICE_NAME}.log" 2>&1 &
    local pid=$!
    
    # 保存PID
    echo "$pid" > "../../../$PID_FILE"
    
    # 等待服务启动
    sleep 3
    
    # 检查服务是否启动成功
    if ps -p "$pid" > /dev/null 2>&1; then
        log_success "服务启动成功 (PID: $pid)"
        log_info "日志文件: $SERVICE_DIR/$LOG_DIR/${SERVICE_NAME}.log"
        
        # 健康检查
        sleep 2
        if curl -s http://localhost:8889/api/v1/health > /dev/null; then
            log_success "服务健康检查通过"
        else
            log_warning "服务健康检查失败，请检查日志"
        fi
    else
        log_error "服务启动失败"
        exit 1
    fi
    
    cd - > /dev/null
}

# 显示服务状态
show_status() {
    echo
    log_info "=== 服务状态 ==="
    echo "服务名称: $SERVICE_NAME"
    echo "配置文件: $SERVICE_DIR/$CONFIG_FILE"
    echo "日志文件: $SERVICE_DIR/$LOG_DIR/${SERVICE_NAME}.log"
    echo "PID文件: $PID_FILE"
    
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            echo -e "运行状态: ${GREEN}运行中${NC} (PID: $pid)"
        else
            echo -e "运行状态: ${RED}已停止${NC}"
        fi
    else
        echo -e "运行状态: ${RED}未运行${NC}"
    fi
    
    # 显示端口使用情况
    echo "端口信息:"
    netstat -tlnp 2>/dev/null | grep :8889 || echo "  端口8889未被占用"
    
    echo
    log_info "=== 快速测试 ==="
    echo "健康检查: curl http://localhost:8889/api/v1/health"
    echo "服务指标: curl http://localhost:8889/api/v1/metrics"
    echo "题目列表: curl http://localhost:8889/api/v1/problems"
    echo
}

# 主函数
main() {
    log_info "启动${SERVICE_NAME}服务..."
    
    # 解析命令行参数
    local force_restart=false
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force-restart)
                force_restart=true
                shift
                ;;
            --help|-h)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --force-restart  强制重启服务"
                echo "  --help, -h       显示帮助信息"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 切换到项目根目录
    cd "$(dirname "$0")/.."
    
    # 检查服务状态
    if check_service_status && [[ "$force_restart" == "false" ]]; then
        log_info "服务已在运行，使用 --force-restart 强制重启"
        show_status
        exit 0
    fi
    
    # 停止现有服务
    if [[ "$force_restart" == "true" ]]; then
        stop_service
    fi
    
    # 执行启动流程
    check_dependencies
    init_database
    build_service
    start_service
    show_status
    
    log_success "${SERVICE_NAME}服务启动完成！"
}

# 错误处理
trap 'log_error "脚本执行失败，退出码: $?"' ERR

# 执行主函数
main "$@"
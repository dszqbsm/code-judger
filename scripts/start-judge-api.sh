#!/bin/bash

# 判题服务启动脚本
# 使用方法: ./scripts/start-judge-api.sh [环境]
# 环境选项: dev (开发环境) | prod (生产环境)

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
JUDGE_API_DIR="${PROJECT_ROOT}/services/judge-api"

# 默认环境为开发环境
ENVIRONMENT=${1:-dev}

# 颜色输出函数
print_info() {
    echo -e "\033[32m[INFO]\033[0m $1"
}

print_warn() {
    echo -e "\033[33m[WARN]\033[0m $1"
}

print_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

# 检查Go环境
check_go_environment() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Go version: ${GO_VERSION}"
}

# 检查必要的目录和文件
check_prerequisites() {
    if [ ! -d "${JUDGE_API_DIR}" ]; then
        print_error "Judge API directory not found: ${JUDGE_API_DIR}"
        exit 1
    fi
    
    if [ ! -f "${JUDGE_API_DIR}/main.go" ]; then
        print_error "main.go not found in ${JUDGE_API_DIR}"
        exit 1
    fi
    
    if [ ! -f "${JUDGE_API_DIR}/etc/judge-api.yaml" ]; then
        print_error "Configuration file not found: ${JUDGE_API_DIR}/etc/judge-api.yaml"
        exit 1
    fi
}

# 创建必要的目录
create_directories() {
    print_info "Creating necessary directories..."
    
    local dirs=(
        "/tmp/judge"
        "/tmp/judge/temp"
        "/tmp/judge/data"
        "/var/log/judge-api"
    )
    
    for dir in "${dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            sudo mkdir -p "$dir"
            sudo chmod 755 "$dir"
            print_info "Created directory: $dir"
        fi
    done
}

# 检查端口是否被占用
check_port() {
    local port=8889
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        print_warn "Port $port is already in use"
        print_info "Trying to stop existing service..."
        pkill -f "judge-api" || true
        sleep 2
    fi
}

# 构建应用
build_application() {
    print_info "Building judge-api application..."
    
    cd "${JUDGE_API_DIR}"
    
    # 下载依赖
    print_info "Downloading dependencies..."
    go mod tidy
    
    # 构建应用
    print_info "Building application..."
    if [ "$ENVIRONMENT" = "prod" ]; then
        go build -ldflags="-s -w" -o judge-api main.go
    else
        go build -o judge-api main.go
    fi
    
    if [ $? -eq 0 ]; then
        print_info "Build completed successfully"
    else
        print_error "Build failed"
        exit 1
    fi
}

# 启动服务
start_service() {
    print_info "Starting judge-api service..."
    
    cd "${JUDGE_API_DIR}"
    
    # 设置环境变量
    export GOOS=linux
    export CGO_ENABLED=1
    
    if [ "$ENVIRONMENT" = "prod" ]; then
        # 生产环境：后台运行
        print_info "Starting in production mode..."
        nohup ./judge-api -f etc/judge-api.yaml > /var/log/judge-api/judge-api.log 2>&1 &
        local PID=$!
        echo $PID > /var/run/judge-api.pid
        print_info "Judge API started with PID: $PID"
        print_info "Log file: /var/log/judge-api/judge-api.log"
    else
        # 开发环境：前台运行
        print_info "Starting in development mode..."
        print_info "Press Ctrl+C to stop the service"
        ./judge-api -f etc/judge-api.yaml
    fi
}

# 验证服务启动
verify_service() {
    if [ "$ENVIRONMENT" = "prod" ]; then
        print_info "Verifying service startup..."
        sleep 3
        
        # 检查进程是否存在
        if [ -f /var/run/judge-api.pid ]; then
            local PID=$(cat /var/run/judge-api.pid)
            if ps -p $PID > /dev/null; then
                print_info "Service is running with PID: $PID"
                
                # 检查健康接口
                print_info "Checking health endpoint..."
                sleep 2
                if curl -s http://localhost:8889/api/v1/judge/health > /dev/null; then
                    print_info "Health check passed ✓"
                else
                    print_warn "Health check failed, but service is running"
                fi
            else
                print_error "Service failed to start"
                exit 1
            fi
        fi
    fi
}

# 显示使用说明
show_usage() {
    echo "Usage: $0 [environment]"
    echo ""
    echo "Environments:"
    echo "  dev   - Development mode (default, runs in foreground)"
    echo "  prod  - Production mode (runs in background)"
    echo ""
    echo "Examples:"
    echo "  $0        # Start in development mode"
    echo "  $0 dev    # Start in development mode"
    echo "  $0 prod   # Start in production mode"
}

# 主函数
main() {
    print_info "Starting Judge API Service..."
    print_info "Environment: $ENVIRONMENT"
    print_info "Project root: $PROJECT_ROOT"
    
    # 如果是帮助参数
    if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
        show_usage
        exit 0
    fi
    
    # 验证环境参数
    if [ "$ENVIRONMENT" != "dev" ] && [ "$ENVIRONMENT" != "prod" ]; then
        print_error "Invalid environment: $ENVIRONMENT"
        show_usage
        exit 1
    fi
    
    # 执行启动流程
    check_go_environment
    check_prerequisites
    create_directories
    check_port
    build_application
    start_service
    verify_service
    
    print_info "Judge API service startup completed!"
}

# 执行主函数
main "$@"

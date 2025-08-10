#!/bin/bash

# 启动用户API服务脚本
# 用途：快速启动用户服务，用于开发和测试

echo "🚀 启动在线判题系统用户API服务..."

# 检查配置文件是否存在
CONFIG_FILE="services/user-api/etc/user-api.yaml"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ 配置文件不存在: $CONFIG_FILE"
    echo "请确保配置文件存在并配置正确"
    exit 1
fi

# 检查依赖服务
echo "📋 检查依赖服务状态..."

# 检查MySQL
if ! docker ps | grep -q mysql; then
    echo "⚠️  MySQL服务未运行，正在启动..."
    docker-compose up -d mysql
    echo "⏳ 等待MySQL服务启动..."
    sleep 10
fi

# 检查Redis
if ! docker ps | grep -q redis; then
    echo "⚠️  Redis服务未运行，正在启动..."
    docker-compose up -d redis
    echo "⏳ 等待Redis服务启动..."
    sleep 5
fi

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ Go环境未安装，请先安装Go 1.21+"
    exit 1
fi

# 进入用户API服务目录
cd services/user-api

# 下载依赖
echo "📦 下载Go模块依赖..."
go mod tidy

# 构建服务
echo "🔨 构建用户API服务..."
go build -o user-api main.go

if [ $? -ne 0 ]; then
    echo "❌ 构建失败"
    exit 1
fi

# 启动服务
echo "🎯 启动用户API服务..."
echo "服务地址: http://localhost:8888"
echo "API文档: http://localhost:8888/api/v1/docs (待实现)"
echo ""
echo "按 Ctrl+C 停止服务"

./user-api -f etc/user-api.yaml
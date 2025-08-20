#!/bin/bash

# 提交服务启动脚本
# Submission API Start Script

echo "🚀 启动提交服务 (Submission API Service)"
echo "================================================"

# 检查配置文件
CONFIG_FILE="etc/submission-api.yaml"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ 配置文件不存在: $CONFIG_FILE"
    echo "请确保配置文件存在并正确配置"
    exit 1
fi

# 检查依赖服务
echo "🔍 检查依赖服务..."

# 检查MySQL
if ! nc -z localhost 3306 2>/dev/null; then
    echo "⚠️  MySQL服务未启动 (localhost:3306)"
fi

# 检查Redis
if ! nc -z localhost 6379 2>/dev/null; then
    echo "⚠️  Redis服务未启动 (localhost:6379)"
fi

# 检查Kafka
if ! nc -z localhost 9094 2>/dev/null; then
    echo "⚠️  Kafka服务未启动 (localhost:9094)"
fi

echo "✅ 依赖检查完成"

# 构建项目
echo "🔨 构建项目..."
if ! go build -o submission-api .; then
    echo "❌ 构建失败"
    exit 1
fi

echo "✅ 构建成功"

# 启动服务
echo "🚀 启动提交服务..."
echo "服务地址: http://localhost:8889"
echo "健康检查: http://localhost:8889/health"
echo "WebSocket: ws://localhost:8889/ws/submissions"
echo ""
echo "按 Ctrl+C 停止服务"
echo "================================================"

# 启动服务
./submission-api -f "$CONFIG_FILE"


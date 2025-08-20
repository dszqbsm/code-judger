#!/bin/bash

# 用户服务启动脚本
# 功能：编译并启动用户API服务

set -e

SERVICE_NAME="user-api"
CONFIG_FILE="etc/user-api.yaml"
LOG_FILE="logs/user-api.log"

# 创建日志目录
mkdir -p logs

echo "正在编译 $SERVICE_NAME 服务..."
go build -o $SERVICE_NAME .

echo "正在启动 $SERVICE_NAME 服务..."
echo "配置文件: $CONFIG_FILE"
echo "日志文件: $LOG_FILE"

# 检查配置文件是否存在
if [ ! -f "$CONFIG_FILE" ]; then
    echo "错误: 配置文件 $CONFIG_FILE 不存在"
    exit 1
fi

# 启动服务
./$SERVICE_NAME -f $CONFIG_FILE > $LOG_FILE 2>&1 &

# 获取进程ID
PID=$!
echo "服务已启动，进程ID: $PID"
echo "日志文件: $LOG_FILE"

# 等待服务启动
sleep 2

# 检查服务是否正常启动
if ps -p $PID > /dev/null; then
    echo "✅ $SERVICE_NAME 服务启动成功"
    echo "可以使用以下命令查看日志:"
    echo "tail -f $LOG_FILE"
else
    echo "❌ $SERVICE_NAME 服务启动失败，请检查日志文件: $LOG_FILE"
    exit 1
fi

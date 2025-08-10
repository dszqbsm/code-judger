#!/bin/bash

# 镜像列表
images=(
    "mysql:8.0"
    "redis:7-alpine" 
    "confluentinc/cp-zookeeper:7.4.0"
    "confluentinc/cp-kafka:7.4.0"
    "docker.elastic.co/elasticsearch/elasticsearch:8.11.0"
    "docker.elastic.co/logstash/logstash:8.11.0"
    "docker.elastic.co/kibana/kibana:8.11.0"
    "consul:1.16"
    "prom/prometheus:v2.47.0"
    "grafana/grafana:10.1.0"
    "provectuslabs/kafka-ui:latest"
)

echo "开始逐个拉取镜像..."
echo "总共需要拉取 ${#images[@]} 个镜像"
echo "================================"

success_count=0
failed_images=()

for i in "${!images[@]}"; do
    image="${images[$i]}"
    echo "[$((i+1))/${#images[@]}] 正在拉取: $image"
    
    if docker pull "$image"; then
        echo "✅ $image 拉取成功"
        ((success_count++))
    else
        echo "❌ $image 拉取失败"
        failed_images+=("$image")
    fi
    echo "--------------------------------"
done

echo "拉取完成！"
echo "成功: $success_count/${#images[@]}"

if [ ${#failed_images[@]} -gt 0 ]; then
    echo "失败的镜像:"
    for img in "${failed_images[@]}"; do
        echo "  - $img"
    done
    exit 1
else
    echo "🎉 所有镜像拉取成功！"
    exit 0
fi
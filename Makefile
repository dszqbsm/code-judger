# 文件名：Makefile
# 用途：在线判题系统开发环境管理的便捷命令集合
# 创建日期：2024-01-15
# 版本：v1.0
# 说明：提供标准化的开发环境操作命令，简化Docker服务管理、监控、调试等日常操作
# 依赖：Docker, Docker Compose, start-dev-env.sh, verify-env.sh
#
# 主要功能分类：
# - 环境管理：setup, start, stop, restart, clean
# - 状态查看：status, logs, verify
# - 数据操作：db-connect, db-backup, redis-connect
# - 监控调试：monitoring-urls, shell-*, stress-test
# - 开发工具：dev (一站式启动), docs (文档查看)

.PHONY: help setup start stop restart status logs clean verify build

# 默认目标
.DEFAULT_GOAL := help

# 颜色定义
BLUE := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
NC := \033[0m  # No Color

# 帮助信息
help: ## 显示帮助信息
	@echo "$(BLUE)在线判题系统 - 开发环境管理$(NC)"
	@echo ""
	@echo "$(GREEN)可用命令:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(GREEN)快速开始:$(NC)"
	@echo "  1. make setup    # 首次设置环境"
	@echo "  2. make start    # 启动所有服务"
	@echo "  3. make verify   # 验证环境是否正常"
	@echo ""

# 环境设置
setup: ## 设置开发环境（首次运行）
	@echo "$(BLUE)设置开发环境...$(NC)"
	@./start-dev-env.sh --pull-only
	@echo "$(GREEN)环境设置完成$(NC)"

# 启动服务
start: ## 启动所有服务
	@echo "$(BLUE)启动开发环境...$(NC)"
	@./start-dev-env.sh
	@echo "$(GREEN)开发环境启动完成$(NC)"

# 快速启动（不等待服务就绪）
start-quick: ## 快速启动服务（不等待服务就绪）
	@echo "$(BLUE)快速启动开发环境...$(NC)"
	@./start-dev-env.sh --no-wait
	@echo "$(GREEN)服务启动完成（请手动验证服务状态）$(NC)"

# 停止服务
stop: ## 停止所有服务
	@echo "$(BLUE)停止所有服务...$(NC)"
	@docker-compose down
	@echo "$(GREEN)所有服务已停止$(NC)"

# 重启服务
restart: ## 重启所有服务
	@echo "$(BLUE)重启所有服务...$(NC)"
	@docker-compose down
	@docker-compose up -d
	@echo "$(GREEN)所有服务已重启$(NC)"

# 重启特定服务
restart-mysql: ## 重启 MySQL 服务
	@echo "$(BLUE)重启 MySQL 服务...$(NC)"
	@docker-compose restart mysql
	@echo "$(GREEN)MySQL 服务已重启$(NC)"

restart-redis: ## 重启 Redis 服务
	@echo "$(BLUE)重启 Redis 服务...$(NC)"
	@docker-compose restart redis
	@echo "$(GREEN)Redis 服务已重启$(NC)"

restart-kafka: ## 重启 Kafka 服务
	@echo "$(BLUE)重启 Kafka 服务...$(NC)"
	@docker-compose restart kafka
	@echo "$(GREEN)Kafka 服务已重启$(NC)"

restart-elk: ## 重启 ELK 服务
	@echo "$(BLUE)重启 ELK 服务...$(NC)"
	@docker-compose restart elasticsearch logstash kibana
	@echo "$(GREEN)ELK 服务已重启$(NC)"

# 服务状态
status: ## 查看服务状态
	@echo "$(BLUE)服务状态:$(NC)"
	@docker-compose ps

# 查看日志
logs: ## 查看所有服务日志
	@docker-compose logs -f

logs-mysql: ## 查看 MySQL 日志
	@docker-compose logs -f mysql

logs-redis: ## 查看 Redis 日志
	@docker-compose logs -f redis

logs-kafka: ## 查看 Kafka 日志
	@docker-compose logs -f kafka

logs-elk: ## 查看 ELK 日志
	@docker-compose logs -f elasticsearch logstash kibana

logs-consul: ## 查看 Consul 日志
	@docker-compose logs -f consul

logs-monitoring: ## 查看监控服务日志
	@docker-compose logs -f prometheus grafana

# 环境验证
verify: ## 验证开发环境
	@echo "$(BLUE)验证开发环境...$(NC)"
	@./verify-env.sh

verify-quick: ## 快速验证环境
	@echo "$(BLUE)快速验证环境...$(NC)"
	@./verify-env.sh --quick

# 数据库操作
db-connect: ## 连接到 MySQL 数据库
	@echo "$(BLUE)连接到 MySQL 数据库...$(NC)"
	@docker exec -it oj-mysql mysql -u oj_user -p oj_system

db-backup: ## 备份数据库
	@echo "$(BLUE)备份数据库...$(NC)"
	@mkdir -p backups
	@docker exec oj-mysql mysqldump -u oj_user -poj_password oj_system > backups/oj_system_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)数据库备份完成$(NC)"

db-restore: ## 恢复数据库（需要指定备份文件）
	@echo "$(BLUE)恢复数据库...$(NC)"
	@echo "请使用: docker exec -i oj-mysql mysql -u oj_user -poj_password oj_system < backups/your_backup.sql"

# Redis 操作
redis-connect: ## 连接到 Redis
	@echo "$(BLUE)连接到 Redis...$(NC)"
	@docker exec -it oj-redis redis-cli

redis-flushall: ## 清空 Redis 所有数据
	@echo "$(YELLOW)警告: 这将清空所有 Redis 数据!$(NC)"
	@read -p "确认继续? [y/N]: " confirm && [ "$$confirm" = "y" ] || exit 1
	@docker exec oj-redis redis-cli flushall
	@echo "$(GREEN)Redis 数据已清空$(NC)"

# Kafka 操作
kafka-topics: ## 查看 Kafka 主题列表
	@echo "$(BLUE)Kafka 主题列表:$(NC)"
	@docker exec oj-kafka kafka-topics --list --bootstrap-server localhost:9092

kafka-create-topic: ## 创建 Kafka 主题（需要指定主题名）
	@echo "$(BLUE)创建 Kafka 主题...$(NC)"
	@read -p "主题名称: " topic && \
	docker exec oj-kafka kafka-topics --create --topic $$topic --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1
	@echo "$(GREEN)主题创建完成$(NC)"

# 清理操作
clean: ## 清理停止的容器和未使用的镜像
	@echo "$(BLUE)清理 Docker 资源...$(NC)"
	@docker system prune -f
	@echo "$(GREEN)清理完成$(NC)"

clean-all: ## 完全清理（删除所有容器、镜像、卷）
	@echo "$(RED)警告: 这将删除所有数据!$(NC)"
	@read -p "确认继续? [y/N]: " confirm && [ "$$confirm" = "y" ] || exit 1
	@docker-compose down -v --rmi all
	@docker system prune -a -f
	@echo "$(GREEN)完全清理完成$(NC)"

# 更新镜像
update: ## 更新所有镜像到最新版本
	@echo "$(BLUE)更新镜像...$(NC)"
	@docker-compose pull
	@echo "$(GREEN)镜像更新完成$(NC)"

# 开发工具
shell-mysql: ## 进入 MySQL 容器 shell
	@docker exec -it oj-mysql bash

shell-redis: ## 进入 Redis 容器 shell
	@docker exec -it oj-redis sh

shell-kafka: ## 进入 Kafka 容器 shell
	@docker exec -it oj-kafka bash

shell-consul: ## 进入 Consul 容器 shell
	@docker exec -it oj-consul sh

# 监控相关
monitoring-urls: ## 显示监控服务访问地址
	@echo "$(BLUE)监控服务访问地址:$(NC)"
	@echo "  📊 Grafana:         http://localhost:3000 (admin/oj_grafana_admin)"
	@echo "  📈 Prometheus:      http://localhost:9090"
	@echo "  📋 Kibana:          http://localhost:5601"
	@echo "  🏛️  Consul:          http://localhost:8500"
	@echo "  🎛️  Kafka UI:        http://localhost:8080"

# 性能测试
stress-test: ## 运行基础压力测试
	@echo "$(BLUE)运行基础压力测试...$(NC)"
	@echo "测试 MySQL 连接性能..."
	@for i in {1..10}; do docker exec oj-mysql mysql -u oj_user -poj_password oj_system -e "SELECT 1;" >/dev/null 2>&1 && echo "MySQL 连接 $$i: OK" || echo "MySQL 连接 $$i: FAIL"; done
	@echo "测试 Redis 性能..."
	@docker exec oj-redis redis-cli --latency -i 1 -c 10 >/dev/null 2>&1 && echo "Redis 延迟测试: OK" || echo "Redis 延迟测试: FAIL"
	@echo "$(GREEN)压力测试完成$(NC)"

# 开发模式
dev: start verify monitoring-urls ## 完整的开发环境启动（启动 + 验证 + 显示地址）

# 生产部署准备
prod-check: ## 检查生产部署准备情况
	@echo "$(BLUE)检查生产部署准备情况...$(NC)"
	@echo "$(YELLOW)注意: 这是开发环境配置，生产环境需要额外配置:$(NC)"
	@echo "  - 修改默认密码"
	@echo "  - 启用 SSL/TLS"
	@echo "  - 配置防火墙"
	@echo "  - 设置资源限制"
	@echo "  - 配置数据备份"
	@echo "  - 启用监控告警"

# 文档
docs: ## 查看相关文档
	@echo "$(BLUE)相关文档:$(NC)"
	@echo "  📚 项目文档:        ./README.md"
	@echo "  🐳 Docker 配置:     ./DOCKER_SETUP.md"
	@echo "  ⚙️  技术选型:        ./docs/技术选型分析.md"
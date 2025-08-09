# 在线判题系统 - Docker 本地开发环境搭建指南

> **文档信息**
> - 文件名：DOCKER_SETUP.md
> - 用途：Docker容器化开发环境的完整搭建和管理指南
> - 创建日期：2024-01-15
> - 版本：v1.0
> - 适用对象：开发人员、运维工程师、系统管理员
> 
> **指南特色**
> - 🎯 **一键启动**：提供自动化脚本，5分钟完成环境搭建
> - 🏗️ **完整栈**：包含9个微服务的完整开发环境
> - 🔧 **易管理**：提供Makefile便捷命令和健康检查
> - 📊 **可观测**：集成监控、日志、可视化完整解决方案
> - 🛡️ **生产就绪**：开发配置可平滑过渡到生产环境

## 📋 环境要求

在开始之前，请确保您的系统已安装以下软件：

- **Docker**: 版本 20.0+ 
- **Docker Compose**: 版本 2.0+
- **可用内存**: 至少 4GB RAM
- **可用磁盘空间**: 至少 10GB

## 🚀 快速启动

### 1. 克隆项目并进入目录
```bash
git clone <项目地址>
cd code-judger
```

### 2. 启动所有服务
```bash
# 后台启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看服务日志
docker-compose logs -f
```

### 3. 等待服务初始化
首次启动需要等待 3-5 分钟，让所有服务完成初始化。您可以通过以下命令监控启动进度：

```bash
# 监控所有服务日志
docker-compose logs -f

# 监控特定服务日志
docker-compose logs -f mysql
docker-compose logs -f elasticsearch
```

## 🔍 服务访问地址

启动成功后，您可以通过以下地址访问各个服务：

| 服务名称 | 访问地址 | 用户名 | 密码 | 说明 |
|---------|---------|--------|------|------|
| **MySQL** | `localhost:3306` | `oj_user` | `oj_password` | 主数据库 |
| **Redis** | `localhost:6379` | - | - | 缓存服务 |
| **Kafka** | `localhost:9094` | - | - | 消息队列（外部访问） |
| **Kafka UI** | http://localhost:8080 | - | - | Kafka管理界面 |
| **Elasticsearch** | http://localhost:9200 | - | - | 搜索引擎 |
| **Kibana** | http://localhost:5601 | - | - | 日志可视化 |
| **Consul** | http://localhost:8500 | - | - | 服务注册中心 |
| **Prometheus** | http://localhost:9090 | - | - | 监控指标收集 |
| **Grafana** | http://localhost:3000 | `admin` | `oj_grafana_admin` | 监控可视化 |

## ✅ 服务验证步骤

### 1. MySQL 数据库验证
```bash
# 连接数据库
docker exec -it oj-mysql mysql -u oj_user -p oj_system

# 查看表结构
SHOW TABLES;
SELECT COUNT(*) FROM users;
```

预期结果：
- 应该看到 `users`, `problems`, `submissions` 等表
- `users` 表应该有 3 条初始数据（admin, student, teacher）

### 2. Redis 缓存验证
```bash
# 连接 Redis
docker exec -it oj-redis redis-cli

# 测试基本操作
ping
set test_key "test_value"
get test_key
```

预期结果：
- `ping` 命令返回 `PONG`
- 能够正常设置和获取键值

### 3. Kafka 消息队列验证
```bash
# 创建测试主题
docker exec -it oj-kafka kafka-topics --create --topic test-topic --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1

# 查看主题列表
docker exec -it oj-kafka kafka-topics --list --bootstrap-server localhost:9092

# 发送测试消息
docker exec -it oj-kafka kafka-console-producer --topic test-topic --bootstrap-server localhost:9092
# 输入: Hello Kafka

# 消费测试消息
docker exec -it oj-kafka kafka-console-consumer --topic test-topic --from-beginning --bootstrap-server localhost:9092
```

### 4. Elasticsearch 验证
```bash
# 检查集群健康状态
curl -X GET "localhost:9200/_cluster/health?pretty"

# 查看索引
curl -X GET "localhost:9200/_cat/indices?v"
```

预期结果：
- 集群状态应该为 `green` 或 `yellow`
- 应该看到系统自动创建的索引

### 5. Consul 服务注册验证
访问 http://localhost:8500，应该看到：
- Consul Web UI 界面
- 服务列表中显示 consul 服务
- 节点状态为健康

### 6. Prometheus 监控验证
访问 http://localhost:9090，验证：
- Prometheus Web UI 可以正常访问
- Status > Targets 页面显示各个监控目标
- 可以执行简单查询如 `up`

### 7. Grafana 可视化验证
访问 http://localhost:3000：
- 使用 admin/oj_grafana_admin 登录
- 检查数据源配置（Settings > Data Sources）
- Prometheus 数据源应该显示为绿色（Connected）

## 🛠️ 常见问题排查

### 服务启动失败

**症状**: 某个容器无法启动或不断重启
```bash
# 查看容器状态
docker-compose ps

# 查看失败服务的日志
docker-compose logs <service_name>

# 重启特定服务
docker-compose restart <service_name>
```

**常见解决方案**:
1. **端口冲突**: 检查端口是否被占用
2. **内存不足**: 确保系统有足够的可用内存
3. **磁盘空间不足**: 清理 Docker 镜像和容器

### MySQL 连接问题

**症状**: 无法连接到 MySQL 数据库
```bash
# 检查 MySQL 日志
docker-compose logs mysql

# 进入 MySQL 容器
docker exec -it oj-mysql bash

# 检查 MySQL 进程
ps aux | grep mysql
```

**解决方案**:
1. 等待 MySQL 完全启动（首次启动需要更长时间）
2. 检查密码配置是否正确
3. 确认防火墙设置

### Elasticsearch 内存错误

**症状**: Elasticsearch 启动失败，提示内存错误
```bash
# 查看 Elasticsearch 日志
docker-compose logs elasticsearch
```

**解决方案**:
```bash
# 增加虚拟内存限制
sudo sysctl -w vm.max_map_count=262144

# 永久设置
echo 'vm.max_map_count=262144' | sudo tee -a /etc/sysctl.conf
```

### Kafka 连接问题

**症状**: 无法连接到 Kafka 或消息发送失败

**解决方案**:
1. 确认 Zookeeper 已成功启动
2. 检查 Kafka 广播地址配置
3. 验证网络连接性

## 🔧 开发环境配置

### 环境变量配置
创建 `.env` 文件来自定义配置：
```bash
# 数据库配置
MYSQL_ROOT_PASSWORD=your_root_password
MYSQL_PASSWORD=your_user_password

# Redis 配置
REDIS_PASSWORD=your_redis_password

# Grafana 配置
GRAFANA_ADMIN_PASSWORD=your_grafana_password
```

### 数据持久化
所有数据都存储在 Docker volumes 中：
```bash
# 查看数据卷
docker volume ls

# 备份数据卷
docker run --rm -v oj_mysql_data:/data -v $(pwd):/backup alpine tar czf /backup/mysql_backup.tar.gz -C /data .

# 恢复数据卷
docker run --rm -v oj_mysql_data:/data -v $(pwd):/backup alpine tar xzf /backup/mysql_backup.tar.gz -C /data
```

## 🧹 清理和重置

### 停止所有服务
```bash
# 停止服务
docker-compose down

# 停止服务并删除卷（注意：会删除所有数据）
docker-compose down -v
```

### 完全重置环境
```bash
# 停止并删除所有容器、网络、卷
docker-compose down -v --remove-orphans

# 删除相关镜像
docker-compose down --rmi all

# 清理 Docker 系统
docker system prune -a
```

## 📊 监控和调试

### 资源使用监控
```bash
# 查看容器资源使用情况
docker stats

# 查看系统资源
htop
df -h
```

### 性能调优建议

1. **MySQL 性能调优**:
   - 调整 `innodb_buffer_pool_size`
   - 配置慢查询日志
   - 优化索引策略

2. **Redis 性能调优**:
   - 配置合适的 `maxmemory`
   - 选择合适的数据淘汰策略
   - 启用 AOF 持久化

3. **Elasticsearch 性能调优**:
   - 调整 JVM 堆内存设置
   - 配置合适的分片数量
   - 启用索引生命周期管理

## 🎯 下一步

环境搭建完成后，您可以：

1. **开发微服务**: 使用已配置的基础设施开发各个微服务
2. **集成测试**: 在完整环境中进行集成测试
3. **监控调试**: 使用 Grafana 和 Kibana 进行系统监控
4. **性能测试**: 对系统进行压力测试和性能调优

## 📞 技术支持

如果您在环境搭建过程中遇到问题，请：

1. 查看本文档的常见问题排查部分
2. 检查 Docker 和 Docker Compose 版本
3. 查看各服务的详细日志
4. 联系技术团队获取支持

---

**祝您开发愉快！🎉**
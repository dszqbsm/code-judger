# 🚀 在线判题系统 - 快速启动指南

> 5分钟快速搭建完整的在线判题系统开发环境

## 📋 准备工作

确保您的系统已安装：
- **Docker** 20.0+
- **Docker Compose** 2.0+
- **可用内存** 4GB+

## ⚡ 一键启动

```bash
# 1. 克隆项目
git clone <项目地址>
cd code-judger

# 2. 启动开发环境 (自动处理所有依赖)
./start-dev-env.sh
```

等待 3-5 分钟，系统将自动：
- 📥 下载所有必需的 Docker 镜像
- 🔧 配置系统参数（如 Elasticsearch 的 vm.max_map_count）
- 🚀 启动 10+ 个微服务
- ✅ 验证所有服务健康状态
- 📊 显示访问地址

## 🎯 访问服务

启动完成后，您可以立即访问：

| 服务 | 地址 | 用户名 | 密码 |
|------|------|--------|------|
| 🎛️ **Kafka 管理** | http://localhost:8080 | - | - |
| 📈 **监控面板** | http://localhost:3000 | `admin` | `oj_grafana_admin` |
| 🔍 **日志查询** | http://localhost:5601 | - | - |
| 🏛️ **服务注册** | http://localhost:8500 | - | - |
| 📊 **监控指标** | http://localhost:9090 | - | - |

## 🔧 常用命令

### 快速操作
```bash
# 查看所有服务状态
make status

# 查看服务日志
make logs

# 验证环境完整性
make verify

# 停止所有服务
make stop

# 查看所有可用命令
make help
```

### 数据库操作
```bash
# 连接 MySQL 数据库
make db-connect
# 用户名: oj_user, 密码: oj_password, 数据库: oj_system

# 连接 Redis
make redis-connect
```

### 调试和监控
```bash
# 查看监控面板地址
make monitoring-urls

# 查看特定服务日志
make logs-mysql
make logs-kafka
make logs-elk
```

## 📊 内置数据

系统启动后自动包含：

### 测试用户
| 用户名 | 密码 | 角色 | 说明 |
|--------|------|------|------|
| `admin` | `admin123` | 管理员 | 系统管理员账户 |
| `teacher` | `teacher123` | 教师 | 教师账户 |
| `student` | `student123` | 学生 | 学生账户 |

### 示例题目
- **A + B Problem**: 经典入门题目
- **Hello World**: 基础输出题目

## 🔍 验证环境

运行完整的环境验证：
```bash
# 详细验证（推荐）
./verify-env.sh

# 快速验证
./verify-env.sh --quick
```

验证内容包括：
- ✅ 所有服务运行状态
- ✅ 端口连通性测试
- ✅ 数据库连接和数据完整性
- ✅ 缓存读写操作
- ✅ 消息队列收发测试
- ✅ 日志系统功能
- ✅ 监控系统状态

## 🛠️ 开发工作流

### 首次使用
1. **启动**: `./start-dev-env.sh`
2. **验证**: `./verify-env.sh`
3. **浏览**: 访问各服务的 Web 界面

### 日常开发
1. **启动**: `make start`
2. **开发**: 编写代码，服务自动重载
3. **调试**: `make logs` 查看日志
4. **停止**: `make stop`

### 问题排查
1. **检查状态**: `make status`
2. **查看日志**: `make logs-<service>`
3. **重启服务**: `make restart-<service>`
4. **完整验证**: `make verify`

## ⚠️ 常见问题

### 启动失败
```bash
# 检查 Docker 服务
sudo systemctl status docker

# 检查端口占用
netstat -tlnp | grep :3306

# 清理并重新启动
make clean
make start
```

### 内存不足
```bash
# 停止不必要的服务
docker stop $(docker ps -q)

# 清理系统资源
make clean

# 调整服务内存限制（编辑 docker-compose.yml）
```

### 权限问题
```bash
# 添加用户到 docker 组
sudo usermod -aG docker $USER
logout  # 重新登录生效

# 设置 Elasticsearch 参数
sudo sysctl -w vm.max_map_count=262144
```

## 📚 进阶使用

### 自定义配置
1. 复制环境变量文件：`cp env.example .env`
2. 编辑配置：`nano .env`
3. 重启服务：`make restart`

### 性能监控
- **Grafana**: 系统性能监控和告警
- **Prometheus**: 指标收集和查询
- **Kibana**: 日志分析和搜索

### 数据管理
```bash
# 数据备份
make db-backup

# 清空 Redis 缓存
make redis-flushall

# 查看 Kafka 主题
make kafka-topics
```

## 🎯 下一步

环境就绪后，您可以：
1. 📖 查看 [完整文档](README.md)
2. 🔧 开始 [微服务开发](docs/)
3. 🧪 运行 [集成测试](#)
4. 📊 配置 [监控告警](#)

## 🆘 获取帮助

- 📚 **详细文档**: `README.md`
- 🐳 **Docker 配置**: `DOCKER_SETUP.md`
- ⚙️ **技术选型**: `docs/技术选型分析.md`
- 💡 **所有命令**: `make help`

---

**🎉 恭喜！您的在线判题系统开发环境已就绪，开始愉快的开发之旅吧！**
# 项目结构说明

## 📁 目录结构

```
code-judger/                           # 项目根目录
├── README.md                          # 项目说明文档
├── go.mod                             # Go模块定义
├── go.sum                             # Go模块校验和
├── Makefile                           # 构建和部署脚本
├── docker-compose.yml                 # Docker编排配置
├── start-user-api.sh                  # 用户API服务启动脚本
├── PROJECT_STRUCTURE.md               # 项目结构说明(本文件)
│
├── docs/                              # 📚 文档目录
│   ├── API接口文档.md                  # 完整的API接口文档
│   ├── 数据库表设计.md                 # 数据库设计文档
│   └── 技术选型分析.md                 # 技术选型分析文档
│
├── sql/                               # 🗄️ 数据库脚本
│   └── init.sql                       # 数据库初始化脚本
│
├── docker/                            # 🐳 Docker配置
│   ├── mysql/                         # MySQL配置
│   ├── redis/                         # Redis配置
│   ├── consul/                        # Consul配置
│   ├── prometheus/                    # Prometheus配置
│   ├── grafana/                       # Grafana配置
│   ├── kibana/                        # Kibana配置
│   └── logstash/                      # Logstash配置
│
├── common/                            # 🔧 通用组件
│   ├── types/                         # 公共类型定义
│   │   └── user.go                    # 用户相关类型
│   ├── utils/                         # 工具函数
│   │   ├── hash.go                    # 密码哈希工具
│   │   ├── jwt.go                     # JWT工具
│   │   └── response.go                # 响应格式工具
│   └── middleware/                    # 公共中间件
│
└── services/                          # 🚀 微服务目录
    ├── user-api/                      # 用户API服务
    │   ├── main.go                    # 服务入口
    │   ├── user.api                   # API定义文件
    │   ├── etc/                       # 配置文件
    │   │   └── user-api.yaml          # 服务配置
    │   ├── internal/                  # 内部实现
    │   │   ├── config/                # 配置结构
    │   │   │   └── config.go
    │   │   ├── handler/               # HTTP处理器
    │   │   │   ├── handler.go         # 路由注册
    │   │   │   ├── auth/              # 认证相关handler
    │   │   │   │   ├── register_handler.go
    │   │   │   │   └── login_handler.go
    │   │   │   ├── users/             # 用户相关handler
    │   │   │   │   └── user_handler.go
    │   │   │   └── admin/             # 管理员相关handler
    │   │   │       └── admin_handler.go
    │   │   ├── logic/                 # 业务逻辑
    │   │   │   └── auth/              # 认证逻辑
    │   │   │       ├── register_logic.go
    │   │   │       └── login_logic.go
    │   │   ├── middleware/            # 中间件
    │   │   │   ├── auth_middleware.go  # 认证中间件
    │   │   │   └── admin_middleware.go # 管理员中间件
    │   │   ├── svc/                   # 服务上下文
    │   │   │   └── service_context.go
    │   │   └── types/                 # 类型定义
    │   │       └── types.go
    │   └── models/                    # 数据模型
    │       ├── user_model.go          # 用户模型
    │       ├── user_token_model.go    # 用户令牌模型
    │       ├── user_statistics_model.go # 用户统计模型
    │       └── user_login_log_model.go # 登录日志模型
    │
    └── user-rpc/                      # 用户RPC服务(待开发)
```

## 🏗️ 架构说明

### 1. 微服务架构
- **services/user-api**: 用户HTTP API服务，处理用户认证、信息管理等
- **services/user-rpc**: 用户RPC服务(待开发)，提供内部服务调用

### 2. 分层设计
- **Handler层**: 处理HTTP请求，参数验证
- **Logic层**: 业务逻辑处理
- **Model层**: 数据持久化操作
- **Middleware层**: 中间件，如认证、权限控制

### 3. 公共组件
- **common/types**: 跨服务共享的数据类型
- **common/utils**: 通用工具函数
- **common/middleware**: 可复用的中间件

## 🚀 快速启动

### 1. 环境准备
```bash
# 1. 启动基础设施服务
make start

# 2. 初始化数据库
mysql -h localhost -P 3306 -u root -p < sql/init.sql

# 3. 验证服务状态
make status
```

### 2. 启动用户服务
```bash
# 使用启动脚本
./start-user-api.sh

# 或手动启动
cd services/user-api
go run main.go -f etc/user-api.yaml
```

### 3. 测试接口
```bash
# 用户注册
curl -X POST http://localhost:8888/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com", 
    "password": "TestPass123!",
    "confirm_password": "TestPass123!",
    "role": "student"
  }'

# 用户登录
curl -X POST http://localhost:8888/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "TestPass123!"
  }'
```

## 📋 开发规范

### 1. 代码结构规范
- 遵循go-zero项目结构约定
- API定义使用.api文件
- 业务逻辑在logic层实现
- 数据操作在model层实现

### 2. 命名规范
- 文件名使用下划线分隔
- 结构体使用大驼峰命名
- 函数和变量使用小驼峰命名
- 常量使用全大写下划线分隔

### 3. 错误处理
- 使用统一的错误码和错误消息
- 详细的错误日志记录
- 友好的用户错误提示

## 🔧 配置说明

### 1. 服务配置
配置文件位置: `services/user-api/etc/user-api.yaml`

主要配置项:
- 服务端口和地址
- 数据库连接信息
- Redis连接信息
- JWT密钥配置
- 业务配置(密码策略、分页等)

### 2. 数据库配置
- MySQL: 主数据库，存储业务数据
- Redis: 缓存和会话存储
- 连接池配置优化

### 3. 日志配置
- 结构化日志输出
- 不同级别的日志记录
- 日志轮转和清理

## 📊 监控和运维

### 1. 健康检查
- 服务健康状态检查
- 数据库连接状态检查
- 外部依赖服务检查

### 2. 性能监控
- Prometheus指标收集
- Grafana可视化面板
- 关键业务指标监控

### 3. 日志分析
- ELK日志聚合分析
- 错误日志告警
- 性能瓶颈分析

## 🔒 安全措施

### 1. 数据安全
- 密码bcrypt加密存储
- 敏感信息不记录日志
- 数据库连接加密

### 2. 接口安全
- JWT令牌认证
- 请求频率限制
- 参数验证和过滤

### 3. 系统安全
- 容器化部署隔离
- 网络访问控制
- 定期安全更新

## 📝 后续开发计划

### Phase 1: 基础功能完善
- [ ] 完善所有handler实现
- [ ] 添加参数验证
- [ ] 完善错误处理
- [ ] 单元测试覆盖

### Phase 2: 高级功能
- [ ] 邮箱验证功能
- [ ] 第三方登录集成
- [ ] 双因子认证
- [ ] 用户行为分析

### Phase 3: 性能优化
- [ ] 数据库查询优化
- [ ] 缓存策略优化
- [ ] 并发性能提升
- [ ] 负载测试和调优

### Phase 4: 运维增强
- [ ] 监控告警完善
- [ ] 自动化部署
- [ ] 日志分析增强
- [ ] 备份恢复机制

## 🤝 贡献指南

1. Fork项目
2. 创建功能分支
3. 提交代码变更
4. 创建Pull Request
5. 代码审查和合并

## 📞 联系方式

- 项目地址: [GitHub Repository]
- 技术文档: `docs/` 目录
- 问题反馈: [GitHub Issues]
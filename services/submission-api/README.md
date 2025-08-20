# 提交服务 (Submission API)

基于go-zero框架开发的在线判题系统提交管理服务。

## 📋 项目概述

提交服务是在线判题系统的核心业务服务，负责处理用户代码提交的完整流程，包括：

- 🚀 **高并发提交处理**：处理大量用户同时提交代码的场景
- 📊 **提交状态管理**：实时跟踪提交从创建到完成的整个生命周期  
- 🔄 **结果实时反馈**：及时将判题结果推送给用户
- 🛡️ **数据一致性保证**：确保提交记录与判题结果的数据一致性
- 🔍 **安全防护机制**：防止恶意提交、代码抄袭等安全威胁

## 🏗️ 技术架构

### 核心技术栈
- **框架**: go-zero v1.6.1 (微服务框架)
- **数据库**: MySQL (主数据存储) + Redis (缓存)
- **消息队列**: Apache Kafka (异步任务处理)
- **实时通信**: WebSocket (状态推送)
- **认证**: JWT Token认证机制

### 项目结构
```
services/submission-api/
├── api/                        # API定义文件
│   ├── submission.api         # 主API定义
│   └── types/
│       └── submission.api     # 类型定义
├── internal/                  # 内部实现
│   ├── config/               # 配置管理
│   ├── handler/              # HTTP处理器(Controller层)
│   ├── logic/                # 业务逻辑(Service层)
│   ├── middleware/           # 中间件
│   ├── svc/                  # 服务上下文
│   ├── types/                # 数据类型
│   ├── messagequeue/         # 消息队列
│   ├── websocket/           # WebSocket管理
│   └── anticheat/           # 查重检测
├── models/                   # 数据模型(DAO层)
├── etc/                     # 配置文件
└── main.go                  # 服务入口
```

## ✅ 已实现功能

### 1. 核心提交管理 (P0功能)
- ✅ **代码提交创建**: `POST /api/v1/submissions`
- ✅ **提交详情查询**: `GET /api/v1/submissions/{id}`
- ✅ **提交列表查询**: `GET /api/v1/submissions`
- ✅ **提交状态更新**: `PUT /api/v1/submissions/{id}/status` (内部接口)
- ✅ **取消提交**: `DELETE /api/v1/submissions/{id}`

### 2. 高级查询功能 (P1功能)
- ✅ **多条件搜索**: `GET /api/v1/submissions/search`
- ✅ **用户提交统计**: `GET /api/v1/users/{user_id}/submission-stats`
- ✅ **题目提交统计**: `GET /api/v1/problems/{problem_id}/submission-stats`

### 3. 实时通信 (P1功能)
- ✅ **WebSocket状态推送**: `/ws/submissions/{id}/status`
- ✅ **用户通知推送**: `/ws/users/{user_id}/submissions`

### 4. 管理员功能 (P1功能)
- ✅ **系统概览统计**: `GET /api/v1/admin/submissions/overview`
- ✅ **异常提交查询**: `GET /api/v1/admin/submissions/anomalies`
- ✅ **批量重新判题**: `POST /api/v1/admin/submissions/rejudge`

### 5. 扩展功能 (P2功能)
- ✅ **代码相似度检测**: `POST /api/v1/admin/submissions/plagiarism/detect`
- ✅ **提交代码比较**: `POST /api/v1/submissions/compare`
- ✅ **数据导出**: `GET /api/v1/admin/submissions/export`

## 🔧 核心组件详解

### 1. 数据模型层 (Models)
- **SubmissionModel**: 提交记录的数据访问对象
- **支持功能**: CRUD操作、高级查询、统计分析、缓存管理
- **技术特色**: go-zero自动生成 + 自定义扩展

### 2. 业务逻辑层 (Logic)
- **CreateSubmissionLogic**: 处理代码提交逻辑
- **GetSubmissionLogic**: 查询提交详情逻辑
- **GetSubmissionListLogic**: 查询提交列表逻辑
- **核心功能**: 参数验证、权限控制、数据处理

### 3. 中间件组件
- **AuthMiddleware**: JWT认证中间件
- **AdminOnlyMiddleware**: 管理员权限中间件
- **RateLimiter**: 提交频率限制中间件

### 4. 消息队列 (MessageQueue)
- **KafkaProducer**: 发布判题任务到Kafka
- **支持主题**: 
  - `judge-tasks`: 判题任务队列
  - `status-updates`: 状态更新通知
  - `notifications`: 用户通知

### 5. WebSocket管理器
- **Manager**: WebSocket连接管理
- **功能**: 连接注册、消息广播、心跳检测
- **Redis集成**: 多实例间消息同步

### 6. 查重检测器 (AntiCheat)
- **多层检测算法**: 字符串相似度 + 语法特征 + 语义分析
- **支持语言**: C/C++、Java、Python、Go、JavaScript
- **检测精度**: 相似度阈值可配置(默认85%)

## 🔒 安全特性

### 1. 认证与授权
- **JWT Token认证**: 无状态认证机制
- **令牌撤销机制**: Redis黑名单支持
- **权限分级**: 学生/教师/管理员三级权限
- **请求频率限制**: 防止恶意大量提交

### 2. 数据安全
- **输入验证**: 严格的参数验证和类型检查
- **代码长度限制**: 防止超大代码文件
- **SQL注入防护**: 使用参数化查询
- **XSS防护**: 输出数据转义

### 3. 业务安全
- **代码查重**: 多维度相似度检测
- **异常行为检测**: 识别可疑提交模式
- **数据一致性**: 事务保证 + 缓存同步

## 📊 性能指标

### 响应时间
- **提交创建**: < 200ms (含数据库写入 + 消息队列发送)
- **列表查询**: < 100ms (缓存命中时 < 50ms)
- **详情查询**: < 50ms (缓存命中时 < 20ms)
- **WebSocket推送**: < 100ms

### 并发能力
- **同时提交处理**: 5000+ requests/second
- **WebSocket连接**: 10000+ 并发连接
- **查询QPS**: 10000+ queries/second

### 缓存性能
- **热点数据命中率**: > 95%
- **缓存更新延迟**: < 10ms
- **数据一致性**: 最终一致性保证

## 🚀 部署说明

### 1. 环境依赖
```yaml
# 必需服务
- MySQL 8.0+
- Redis 6.0+
- Apache Kafka 2.8+

# Go环境
- Go 1.21+
- go-zero v1.6.1
```

### 2. 配置文件
```bash
# 复制配置模板
cp etc/submission-api.yaml.example etc/submission-api.yaml

# 修改配置
vim etc/submission-api.yaml
```

### 3. 启动服务
```bash
# 开发环境
go run main.go -f etc/submission-api.yaml

# 生产环境
./submission-api -f etc/submission-api.yaml
```

### 4. 健康检查
```bash
# 服务健康检查
curl http://localhost:8889/health

# API测试
curl -X POST http://localhost:8889/api/v1/submissions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"problem_id": 1, "language": "cpp", "code": "..."}'
```

## 🔧 开发指南

### 1. 添加新接口
```bash
# 1. 修改API定义
vim api/submission.api

# 2. 重新生成代码
goctl api go -api api/submission.api -dir .

# 3. 实现业务逻辑
vim internal/logic/newlogic.go
```

### 2. 扩展数据模型
```bash
# 1. 修改SQL表结构
vim models/submissions.sql

# 2. 重新生成模型
goctl model mysql ddl -src models/submissions.sql -dir models -cache

# 3. 添加自定义方法
vim models/submission_model_extend.go
```

### 3. 添加中间件
```go
// 创建新中间件
type NewMiddleware struct{}

func (m NewMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 中间件逻辑
        next(w, r)
    }
}
```

## 📈 监控和运维

### 1. 关键指标
- **业务指标**: 提交成功率、平均处理时间、用户活跃度
- **技术指标**: QPS、响应时间、错误率、缓存命中率
- **资源指标**: CPU使用率、内存使用率、数据库连接数

### 2. 日志管理
- **结构化日志**: JSON格式，便于查询分析
- **日志级别**: ERROR/WARN/INFO/DEBUG
- **日志轮转**: 按大小和时间自动轮转

### 3. 告警配置
- **服务不可用**: 健康检查失败
- **响应时间异常**: P99延迟 > 1s
- **错误率异常**: 错误率 > 5%
- **资源告警**: CPU/内存使用率 > 80%

## 🔄 后续优化计划

### 短期优化 (1-2周)
- [ ] 完善单元测试覆盖率 (目标: >90%)
- [ ] 优化数据库查询性能
- [ ] 增加更多监控指标
- [ ] 完善错误处理机制

### 中期优化 (1个月)
- [ ] 实现分布式缓存
- [ ] 添加链路追踪
- [ ] 优化WebSocket性能
- [ ] 增强安全防护

### 长期优化 (3个月)
- [ ] 支持水平扩展
- [ ] 机器学习查重算法
- [ ] 实时数据分析
- [ ] 多语言SDK支持

## 🤝 贡献指南

1. **Fork** 项目到你的GitHub
2. **创建特性分支**: `git checkout -b feature/amazing-feature`
3. **提交更改**: `git commit -m 'Add amazing feature'`
4. **推送分支**: `git push origin feature/amazing-feature`
5. **提交Pull Request**

## 📞 技术支持

- **项目地址**: [GitHub仓库链接]
- **文档地址**: [在线文档链接]
- **问题反馈**: [Issue页面链接]
- **技术交流**: [技术群/论坛链接]

---

**提交服务开发团队**  
*基于go-zero框架的高性能在线判题系统*


# 题目服务 (Problem Service)

## 项目概述

题目服务是在线判题系统的核心模块之一，负责管理编程题目的完整生命周期，包括题目的创建、查询、更新、删除以及相关的分类标签管理。

### 主要功能

- ✅ **题目CRUD操作**：完整的题目增删改查功能
- ✅ **分页查询**：支持高性能的分页列表查询
- ✅ **多条件筛选**：按难度、标签、关键词筛选题目
- ✅ **缓存优化**：Redis缓存提升查询性能
- ✅ **权限控制**：基于JWT的权限验证
- ✅ **软删除**：安全的题目删除机制
- ✅ **健康检查**：服务状态监控
- ✅ **性能指标**：服务监控指标收集

### 技术特色

- **微服务架构**：独立部署、易于扩展
- **高性能缓存**：Redis集成，查询性能优异
- **数据库优化**：完整的索引设计和查询优化
- **标准化接口**：RESTful API设计
- **完善的日志**：结构化日志和错误追踪

## 快速开始

### 环境要求

- Go 1.21+
- MySQL 8.0+
- Redis 7.0+
- Docker & Docker Compose (可选)

### 1. 环境准备

```bash
# 启动基础设施服务
cd /opt/go-project/code-judger
make start

# 初始化题目数据库
mysql -h localhost -P 3306 -u root -p < sql/problems_init.sql
```

### 2. 配置服务

```bash
# 复制配置文件
cp services/problem-api/etc/problem-api.yaml.example services/problem-api/etc/problem-api.yaml

# 根据实际环境修改配置
vim services/problem-api/etc/problem-api.yaml
```

### 3. 启动服务

#### 方式一：使用启动脚本（推荐）

```bash
# 使用启动脚本（自动处理依赖检查、数据库初始化、服务构建）
./scripts/start-problem-api.sh

# 强制重启服务
./scripts/start-problem-api.sh --force-restart
```

#### 方式二：手动启动

```bash
cd services/problem-api

# 安装依赖
go mod tidy

# 构建服务
go build -o bin/problem-api main.go

# 启动服务
./bin/problem-api -f etc/problem-api.yaml
```

### 4. 验证服务

```bash
# 健康检查
curl http://localhost:8889/api/v1/health

# 服务指标
curl http://localhost:8889/api/v1/metrics

# 题目列表（需要JWT token）
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" http://localhost:8889/api/v1/problems
```

## API接口

### 核心接口列表

| 接口 | 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|------|
| 创建题目 | POST | `/api/v1/problems` | 教师/管理员 | 创建新题目 |
| 题目列表 | GET | `/api/v1/problems` | 登录用户 | 分页查询题目列表 |
| 题目详情 | GET | `/api/v1/problems/{id}` | 登录用户 | 获取题目详细信息 |
| 更新题目 | PUT | `/api/v1/problems/{id}` | 创建者/管理员 | 更新题目信息 |
| 删除题目 | DELETE | `/api/v1/problems/{id}` | 创建者/管理员 | 软删除题目 |
| **上传测试用例** | **POST** | **`/api/v1/problems/{id}/test-cases`** | **教师/管理员** | **批量上传测试用例** |
| **获取测试用例** | **GET** | **`/api/v1/problems/{id}/test-cases`** | **登录用户** | **获取题目测试用例列表** |
| **测试用例详情** | **GET** | **`/api/v1/test-cases/{id}`** | **登录用户** | **获取单个测试用例详情** |
| **更新测试用例** | **PUT** | **`/api/v1/test-cases/{id}`** | **创建者/管理员** | **更新测试用例** |
| **删除测试用例** | **DELETE** | **`/api/v1/test-cases/{id}`** | **创建者/管理员** | **删除测试用例** |
| **判题服务专用-题目** | **GET** | **`/internal/v1/problems/{id}`** | **内部服务** | **供判题服务获取题目信息** |
| **判题服务专用-用例** | **GET** | **`/internal/v1/problems/{id}/test-cases`** | **内部服务** | **供判题服务获取测试用例** |
| 健康检查 | GET | `/api/v1/health` | 无需认证 | 服务健康状态 |
| 服务指标 | GET | `/api/v1/metrics` | 无需认证 | 服务性能指标 |

### 接口示例

#### 创建题目

```bash
curl -X POST http://localhost:8889/api/v1/problems \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "title": "两数之和",
    "description": "给定一个整数数组nums和一个整数目标值target，请你在该数组中找出和为目标值target的那两个整数，并返回它们的数组下标。",
    "input_format": "第一行包含一个整数n，表示数组长度。第二行包含n个整数，表示数组nums。第三行包含一个整数target，表示目标值。",
    "output_format": "输出两个整数，表示和为target的两个数的下标（从0开始），用空格分隔。",
    "sample_input": "4\n2 7 11 15\n9",
    "sample_output": "0 1",
    "difficulty": "easy",
    "time_limit": 1000,
    "memory_limit": 128,
    "languages": ["cpp", "java", "python", "go"],
    "tags": ["数组", "哈希表"],
    "is_public": true
  }'
```

#### 获取题目列表

```bash
curl "http://localhost:8889/api/v1/problems?page=1&limit=10&difficulty=easy&tags=数组,哈希表" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 上传测试用例

```bash
curl -X POST http://localhost:8889/api/v1/problems/1/test-cases \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "problem_id": 1,
    "replace_all": true,
    "test_cases": [
      {
        "input_data": "4\n2 7 11 15\n9",
        "expected_output": "0 1",
        "is_sample": true,
        "score": 20,
        "sort_order": 1
      },
      {
        "input_data": "3\n3 2 4\n6",
        "expected_output": "1 2",
        "is_sample": false,
        "score": 40,
        "sort_order": 2
      }
    ]
  }'
```

#### 获取测试用例列表

```bash
# 获取测试用例列表（不包含具体数据）
curl "http://localhost:8889/api/v1/problems/1/test-cases" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 获取测试用例列表（包含具体数据）
curl "http://localhost:8889/api/v1/problems/1/test-cases?include_data=true" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 只获取示例测试用例
curl "http://localhost:8889/api/v1/problems/1/test-cases?only_samples=true&include_data=true" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 判题服务获取题目信息

```bash
# 获取题目详细信息（供判题服务使用）
curl "http://localhost:8891/internal/v1/problems/11" \
  -H "User-Agent: judge-api/1.0.0"
```

#### 判题服务获取测试用例

```bash
# 获取所有测试用例（供判题服务使用）
curl "http://localhost:8891/internal/v1/problems/11/test-cases?include_hidden=true" \
  -H "User-Agent: judge-service/1.0"

# 只获取示例测试用例
curl "http://localhost:8891/internal/v1/problems/11/test-cases" \
  -H "User-Agent: judge-service/1.0"
```

详细的API文档请参考：[API接口文档](../../docs/API接口文档.md#8-题目管理接口)

## 项目结构

```
services/problem-api/
├── main.go                    # 服务入口
├── problem.api               # API定义文件
├── go.mod                    # Go模块定义
├── etc/                      # 配置文件
│   └── problem-api.yaml      # 服务配置
├── internal/                 # 内部实现
│   ├── config/              # 配置结构
│   ├── handler/             # HTTP处理器
│   │   ├── health/          # 健康检查处理器
│   │   └── problem/         # 题目管理处理器
│   ├── logic/               # 业务逻辑层
│   │   ├── health/          # 健康检查逻辑
│   │   └── problem/         # 题目管理逻辑
│   ├── svc/                 # 服务上下文
│   └── types/               # 类型定义
├── models/                   # 数据模型
│   └── problem_model.go     # 题目数据模型
└── README.md                # 项目说明
```

## 配置说明

### 核心配置项

```yaml
Name: problem-api
Host: 0.0.0.0
Port: 8889

# 数据库配置
DataSource: oj_user:oj_password@tcp(mysql:3306)/oj_problems?charset=utf8mb4&parseTime=true&loc=Local

# Redis缓存配置
CacheConf:
  - Host: redis:6379
    Type: node

# JWT认证配置
Auth:
  AccessSecret: "your-access-secret"
  AccessExpire: 3600

# 业务配置
Business:
  DefaultPageSize: 20          # 默认分页大小
  MaxPageSize: 100            # 最大分页大小
  ProblemListCacheTTL: 300    # 题目列表缓存时间(秒)
  ProblemDetailCacheTTL: 1800 # 题目详情缓存时间(秒)
```

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `MYSQL_HOST` | mysql | MySQL服务器地址 |
| `MYSQL_PORT` | 3306 | MySQL端口 |
| `REDIS_HOST` | redis | Redis服务器地址 |
| `REDIS_PORT` | 6379 | Redis端口 |
| `JWT_SECRET` | - | JWT密钥（生产环境必须设置） |

## 数据库设计

### 主要数据表

#### problems 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 主键，题目ID |
| title | VARCHAR(200) | 题目标题 |
| description | TEXT | 题目描述 |
| input_format | TEXT | 输入格式说明 |
| output_format | TEXT | 输出格式说明 |
| sample_input | TEXT | 样例输入 |
| sample_output | TEXT | 样例输出 |
| difficulty | ENUM | 难度等级(easy/medium/hard) |
| time_limit | INT | 时间限制(毫秒) |
| memory_limit | INT | 内存限制(MB) |
| languages | JSON | 支持的编程语言 |
| tags | JSON | 题目标签 |
| created_by | BIGINT | 创建者用户ID |
| is_public | BOOLEAN | 是否公开 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | 删除时间(软删除) |

### 索引设计

```sql
-- 主要索引
INDEX idx_difficulty (difficulty)                    -- 难度查询
INDEX idx_created_by (created_by)                    -- 创建者查询
INDEX idx_created_at (created_at)                    -- 时间排序
INDEX idx_public_status (is_public, deleted_at)      -- 公开状态复合索引
FULLTEXT INDEX idx_title_description (title, description) -- 全文搜索
```

## 性能优化

### 缓存策略

1. **题目列表缓存**：5分钟TTL，支持分页和条件筛选
2. **题目详情缓存**：30分钟TTL，支持缓存穿透防护
3. **热点数据预热**：启动时预热热门题目
4. **缓存更新策略**：写入时自动清除相关缓存

### 数据库优化

1. **索引优化**：针对常用查询条件建立复合索引
2. **分页优化**：使用LIMIT+OFFSET进行高效分页
3. **连接池管理**：合理配置数据库连接池
4. **慢查询监控**：记录和优化慢查询

### 查询性能指标

- **题目列表查询**：P95 < 50ms（缓存命中）
- **题目详情查询**：P95 < 100ms（缓存命中）
- **缓存命中率**：> 90%
- **并发支持**：1000+ QPS

## 监控和运维

### 健康检查

```bash
# 服务健康状态
curl http://localhost:8889/api/v1/health

# 预期响应
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "v1.0.0"
}
```

### 性能指标

```bash
# 服务性能指标
curl http://localhost:8889/api/v1/metrics

# 预期响应
{
  "request_count": 1000,
  "error_count": 5,
  "avg_response_time": 85.5,
  "cache_hit_rate": 92.3,
  "database_conn_pool": 10
}
```

### 日志管理

- **日志级别**：支持debug、info、warn、error级别
- **日志格式**：结构化JSON格式
- **日志轮转**：按大小和时间自动轮转
- **链路追踪**：支持请求链路追踪

### 服务监控

- **Prometheus集成**：自动收集服务指标
- **Grafana面板**：可视化监控面板
- **告警规则**：关键指标异常告警

## 开发指南

### 添加新接口

1. 在`problem.api`文件中定义接口
2. 在`internal/types/types.go`中定义请求响应结构
3. 在`internal/logic/`中实现业务逻辑
4. 在`internal/handler/`中实现HTTP处理器
5. 在`internal/handler/routes.go`中注册路由
6. 编写单元测试
7. 更新API文档

### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/logic/problem/

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 代码规范

- 遵循Go官方代码规范
- 使用gofmt格式化代码
- 使用golint检查代码质量
- 编写有意义的注释和文档

## 部署说明

### Docker部署

```bash
# 构建镜像
docker build -t problem-api:latest .

# 运行容器
docker run -d \
  --name problem-api \
  -p 8889:8889 \
  -e MYSQL_HOST=mysql \
  -e REDIS_HOST=redis \
  problem-api:latest
```

### Kubernetes部署

参考项目根目录的Kubernetes配置文件：
- `k8s/problem-api-deployment.yaml`
- `k8s/problem-api-service.yaml`
- `k8s/problem-api-configmap.yaml`

### 生产环境注意事项

1. **安全配置**：修改默认JWT密钥
2. **资源限制**：设置合理的CPU和内存限制
3. **监控告警**：配置完整的监控和告警
4. **备份策略**：定期备份数据库
5. **负载均衡**：使用负载均衡器分发请求

## 故障排查

### 常见问题

#### 1. 服务启动失败

```bash
# 检查配置文件
cat services/problem-api/etc/problem-api.yaml

# 检查依赖服务
mysql -h mysql -u oj_user -poj_password -e "SELECT 1"
redis-cli -h redis ping

# 查看服务日志
tail -f services/problem-api/logs/problem-api.log
```

#### 2. 数据库连接失败

```bash
# 检查数据库连接
mysql -h mysql -u oj_user -poj_password -e "USE oj_problems; SHOW TABLES;"

# 检查数据库用户权限
mysql -h mysql -u root -p -e "SHOW GRANTS FOR 'oj_user'@'%';"
```

#### 3. 缓存连接失败

```bash
# 检查Redis连接
redis-cli -h redis ping

# 检查Redis配置
redis-cli -h redis config get "*"
```

### 性能问题诊断

1. **查看服务指标**：`curl http://localhost:8889/api/v1/metrics`
2. **检查数据库慢查询**：查看MySQL慢查询日志
3. **监控缓存命中率**：检查Redis缓存命中情况
4. **分析服务日志**：查找错误和性能瓶颈

## 贡献指南

1. Fork项目
2. 创建特性分支：`git checkout -b feature/new-feature`
3. 提交变更：`git commit -am 'Add new feature'`
4. 推送分支：`git push origin feature/new-feature`
5. 创建Pull Request

## 许可证

本项目采用MIT许可证 - 详情请参阅 [LICENSE](../../LICENSE) 文件。

## 联系我们

- 项目地址：https://github.com/your-org/code-judger
- 问题反馈：https://github.com/your-org/code-judger/issues
- 邮箱：oj-team@example.com
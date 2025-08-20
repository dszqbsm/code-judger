# 不同环境的题目服务配置

## 开发环境 (development)
```yaml
ProblemService:
  Endpoint: "http://localhost:8888"
  Timeout: 10
  UseMock: true          # 使用Mock客户端，不依赖真实题目服务
  MaxRetries: 0          # 开发环境不需要重试
```

**使用场景：**
- 本地开发时题目服务未启动
- 单独测试判题逻辑
- 快速原型验证

## 测试环境 (testing)  
```yaml
ProblemService:
  Endpoint: "http://problem-service-test:8888"
  Timeout: 5
  UseMock: false         # 使用真实HTTP调用
  MaxRetries: 2          # 适度重试
```

**使用场景：**
- 集成测试
- 验证服务间通信
- 性能测试

## 预发布环境 (staging)
```yaml
ProblemService:
  Endpoint: "http://problem-service-staging:8888"
  Timeout: 10
  UseMock: false
  MaxRetries: 3
```

**使用场景：**
- 生产前最后验证
- 完整功能测试
- 性能压测

## 生产环境 (production)
```yaml
ProblemService:
  Endpoint: "https://problem-service.internal.company.com:8888"
  Timeout: 8
  UseMock: false
  MaxRetries: 3
```

**特点：**
- 使用HTTPS保证安全
- 较短的超时时间保证响应速度
- 重试机制保证可靠性
- 内网域名保证安全性

## 容器化部署配置 (Kubernetes)
```yaml
ProblemService:
  Endpoint: "http://problem-service.judge-system.svc.cluster.local:8888"
  Timeout: 10
  UseMock: false
  MaxRetries: 3
```

**特点：**
- 使用Kubernetes服务发现
- 集群内部通信
- 自动负载均衡

## 微服务注册中心配置 (Service Discovery)
```yaml
ProblemService:
  ServiceName: "problem-service"    # 服务名，从注册中心获取地址
  Timeout: 10
  UseMock: false
  MaxRetries: 3
  LoadBalancer: "round_robin"       # 负载均衡策略
```

## 配置管理最佳实践

### 1. 环境变量覆盖
```bash
# 生产环境通过环境变量覆盖
export PROBLEM_SERVICE_ENDPOINT="https://prod-problem-service.com:8888"
export PROBLEM_SERVICE_USE_MOCK="false"
export PROBLEM_SERVICE_MAX_RETRIES="5"
```

### 2. 配置中心集成
```yaml
# 从配置中心动态获取
ProblemService:
  ConfigCenter:
    Enabled: true
    Key: "judge-api/problem-service"
    RefreshInterval: 30s
```

### 3. 服务发现集成
```yaml
ProblemService:
  Discovery:
    Provider: "consul"    # consul, etcd, nacos
    ServiceName: "problem-service"
    HealthCheck: true
```

## 监控和告警配置

### 1. 服务健康检查
```yaml
ProblemService:
  HealthCheck:
    Enabled: true
    Interval: 30s
    Timeout: 5s
    FailureThreshold: 3
```

### 2. 指标收集
```yaml
ProblemService:
  Metrics:
    RequestDuration: true
    ErrorRate: true
    CircuitBreaker: true
```

### 3. 链路追踪
```yaml
ProblemService:
  Tracing:
    Enabled: true
    SampleRate: 0.1       # 10%采样率
    ServiceName: "judge-api-to-problem-service"
```

## 容错和降级策略

### 1. 熔断器配置
```yaml
ProblemService:
  CircuitBreaker:
    Enabled: true
    FailureThreshold: 10   # 连续失败10次触发熔断
    RecoveryTimeout: 30s   # 30秒后尝试恢复
```

### 2. 缓存策略
```yaml
ProblemService:
  Cache:
    Enabled: true
    TTL: 300s             # 5分钟缓存
    MaxSize: 1000         # 最大缓存1000个题目
```

### 3. 降级策略
```yaml
ProblemService:
  Fallback:
    Enabled: true
    DefaultTimeLimit: 1000    # 默认时间限制
    DefaultMemoryLimit: 128   # 默认内存限制
    UseLocalTestCases: true   # 使用本地测试用例
```

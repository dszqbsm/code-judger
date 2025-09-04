# 内部API安全方案

## 🚨 安全风险分析

### 当前风险
1. **无认证访问**：任何知道URL的客户端都能访问内部接口
2. **数据泄露**：测试用例和题目信息可能被恶意获取
3. **服务伪造**：恶意服务可以伪装成判题服务
4. **网络嗅探**：HTTP明文传输可被截获
5. **DDoS攻击**：内部接口可能被恶意请求淹没

## 🔐 安全解决方案

### 方案1：API Key + IP白名单（已实现）

#### 特点
- ✅ 简单易实现
- ✅ 性能开销小
- ✅ 适合内部网络环境
- ⚠️ 密钥泄露风险

#### 实现细节
```go
// 认证头
X-Internal-API-Key: internal-service-secret-key-2024

// IP白名单
127.0.0.1, ::1          // 本地
172.17.0.0/16          // Docker默认网络
10.0.0.0/8             // 私有网络
```

#### 使用示例
```bash
curl "http://localhost:8891/internal/v1/problems/11" \
  -H "X-Internal-API-Key: internal-service-secret-key-2024" \
  -H "User-Agent: judge-api/1.0.0"
```

### 方案2：mTLS双向认证（推荐生产环境）

#### 特点
- ✅ 最高安全级别
- ✅ 证书自动轮转
- ✅ 传输层加密
- ❌ 实现复杂度高

#### 实现步骤
```bash
# 1. 生成CA证书
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 365 -key ca-key.pem -sha256 -out ca.pem

# 2. 生成服务证书
openssl genrsa -out server-key.pem 4096
openssl req -subj "/CN=problem-api" -sha256 -new -key server-key.pem -out server.csr
openssl x509 -req -days 365 -sha256 -in server.csr -CA ca.pem -CAkey ca-key.pem -out server-cert.pem

# 3. 生成客户端证书
openssl genrsa -out client-key.pem 4096
openssl req -subj "/CN=judge-api" -new -key client-key.pem -out client.csr
openssl x509 -req -days 365 -in client.csr -CA ca.pem -CAkey ca-key.pem -out client-cert.pem
```

### 方案3：JWT服务间认证

#### 特点
- ✅ 标准化协议
- ✅ 支持权限细分
- ✅ 可设置过期时间
- ⚠️ 需要密钥管理

#### 实现方式
```go
// 生成服务间JWT
serviceJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "service": "judge-api",
    "scope":   []string{"problem:read", "testcase:read"},
    "exp":     time.Now().Add(time.Hour).Unix(),
})
```

### 方案4：Service Mesh（Istio）

#### 特点
- ✅ 自动mTLS
- ✅ 流量管理
- ✅ 策略控制
- ❌ 基础设施复杂

#### 配置示例
```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: internal-api-auth
spec:
  selector:
    matchLabels:
      app: problem-api
  mtls:
    mode: STRICT
```

## 🛡️ 推荐的分层安全策略

### 1. 网络层安全
- **VPC隔离**：内部服务部署在私有网络
- **安全组**：只允许必要的端口和协议
- **网络策略**：Kubernetes NetworkPolicy限制Pod间通信

### 2. 传输层安全
- **TLS加密**：所有内部通信使用HTTPS
- **证书管理**：定期轮转证书
- **mTLS认证**：双向证书验证

### 3. 应用层安全
- **API密钥**：每个服务独立的密钥
- **JWT认证**：服务间使用专用JWT
- **权限控制**：最小权限原则

### 4. 监控和审计
- **访问日志**：记录所有内部API调用
- **异常检测**：监控异常访问模式
- **告警机制**：安全事件实时告警

## 🔧 当前实现的安全措施

### 已实现
- ✅ **API密钥认证**：X-Internal-API-Key头部验证
- ✅ **IP白名单**：限制访问来源IP
- ✅ **访问日志**：记录所有内部API访问
- ✅ **错误处理**：安全的错误响应

### 待改进
- ⏳ **TLS加密**：当前使用HTTP，建议升级为HTTPS
- ⏳ **密钥轮转**：定期更换API密钥
- ⏳ **速率限制**：基于Redis的分布式限流
- ⏳ **监控告警**：集成Prometheus监控

## 🚀 生产环境部署建议

### 1. 配置管理
```bash
# 环境变量方式
export INTERNAL_API_KEY="$(openssl rand -base64 32)"
export ALLOWED_IPS="10.0.0.0/8,172.16.0.0/12"

# 配置文件方式
internal_api:
  key: ${INTERNAL_API_KEY}
  allowed_ips: ${ALLOWED_IPS}
  enable_tls: true
  cert_file: /etc/ssl/certs/problem-api.crt
  key_file: /etc/ssl/private/problem-api.key
```

### 2. Kubernetes部署
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: internal-api-secret
type: Opaque
data:
  api-key: <base64-encoded-key>
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: problem-api
spec:
  template:
    spec:
      containers:
      - name: problem-api
        env:
        - name: INTERNAL_API_KEY
          valueFrom:
            secretKeyRef:
              name: internal-api-secret
              key: api-key
```

### 3. 监控配置
```yaml
# Prometheus监控规则
groups:
- name: internal-api-security
  rules:
  - alert: UnauthorizedInternalAccess
    expr: rate(internal_api_unauthorized_total[5m]) > 0
    for: 1m
    annotations:
      summary: "检测到未授权的内部API访问"
```

## 📈 安全性能对比

| 方案 | 安全级别 | 性能影响 | 实现复杂度 | 维护成本 |
|------|----------|----------|------------|----------|
| API Key + IP白名单 | ⭐⭐⭐ | 很低 | 低 | 低 |
| mTLS | ⭐⭐⭐⭐⭐ | 中等 | 高 | 中等 |
| JWT服务认证 | ⭐⭐⭐⭐ | 低 | 中等 | 中等 |
| Service Mesh | ⭐⭐⭐⭐⭐ | 中等 | 很高 | 高 |

## 🎯 推荐配置

### 开发环境
- API Key + User-Agent检查
- 本地IP白名单
- HTTP传输

### 测试环境  
- API Key + IP白名单
- 完整的访问日志
- HTTP传输

### 生产环境
- mTLS + API Key双重认证
- 严格的IP白名单
- HTTPS传输 + 证书轮转
- 完整的监控告警










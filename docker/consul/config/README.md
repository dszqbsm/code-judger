# 文件名：consul.json
# 用途：Consul服务注册发现中心的核心配置文件
# 创建日期：2024-01-15
# 版本：v1.0
# 说明：配置Consul作为微服务架构的服务注册中心，提供服务发现、健康检查、配置管理等功能
# 依赖：无，作为基础设施服务独立运行

## 配置说明

### 基础配置
- `datacenter`: "oj-datacenter" - 数据中心名称，标识服务集群
- `data_dir`: "/consul/data" - 数据存储目录，持久化集群状态
- `log_level`: "INFO" - 日志级别，开发环境使用INFO便于调试
- `node_name`: "oj-consul-server" - 节点名称，集群中的唯一标识

### 网络配置
- `bind_addr`: "0.0.0.0" - 绑定地址，允许所有网络接口访问
- `client_addr`: "0.0.0.0" - 客户端访问地址，允许外部访问
- `retry_join`: ["127.0.0.1"] - 集群加入地址，单节点模式自连接

### 服务配置
- `server`: true - 以服务器模式运行，提供完整功能
- `bootstrap_expect`: 1 - 预期服务器数量，单节点集群设置为1
- `ui_config.enabled`: true - 启用Web UI界面，便于管理和监控
- `connect.enabled`: true - 启用Service Mesh功能
- `ports.grpc`: 8502 - gRPC端口，用于服务间通信

### 安全配置
- `acl.enabled`: false - 开发环境禁用ACL，生产环境建议启用
- `acl.default_policy`: "allow" - 默认策略允许所有操作

### 性能配置
- `performance.raft_multiplier`: 1 - Raft算法性能倍数，1为最快响应

### 服务定义
配置了consul服务自身的健康检查：
- 检查URL: http://localhost:8500/v1/status/leader
- 检查间隔: 10秒
- 用途: 确保Consul服务正常运行

## 注意事项
1. 开发环境配置，生产环境需要启用ACL和TLS加密
2. 单节点模式，生产环境建议至少3个节点保证高可用
3. 默认允许所有操作，生产环境需要配置细粒度权限
4. 数据目录已配置持久化，重启不会丢失服务注册信息
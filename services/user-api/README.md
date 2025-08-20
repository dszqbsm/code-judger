# 用户服务 API (User API)

## 服务描述
用户服务负责用户认证、用户信息管理、权限控制等功能的HTTP API服务。

## 功能模块
- 用户注册/登录
- 用户信息管理
- 权限控制
- JWT令牌管理

## 运行要求
- Go 1.21+
- MySQL 8.0
- Redis 7.0

## 快速启动

### 1. 编译服务
```bash
go build -o user-api .
```

### 2. 启动服务
```bash
./start.sh
```

### 3. 或手动启动
```bash
./user-api -f etc/user-api.yaml
```

## 配置说明
配置文件位于 `etc/user-api.yaml`，包含：
- 服务端口配置
- 数据库连接配置
- Redis配置
- JWT密钥配置

## API文档
API定义文件：`api/user.api`
类型定义文件：`api/types/user.api`

## 测试
```bash
./test_apis.sh
```

## 日志
日志文件存储在 `logs/` 目录下。

## 依赖服务
- MySQL (oj_system数据库)
- Redis (缓存服务)

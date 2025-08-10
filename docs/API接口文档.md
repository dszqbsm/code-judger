# 在线判题系统API接口文档

## 1. 文档概述

### 1.1 接口规范
- **Base URL**: `http://localhost:8888`
- **协议**: HTTP/HTTPS
- **数据格式**: JSON
- **字符编码**: UTF-8
- **认证方式**: JWT Bearer Token

### 1.2 统一响应格式

所有API接口均采用统一的响应格式：

```json
{
    "code": 200,           // 状态码
    "message": "操作成功",  // 消息描述
    "data": {}             // 响应数据（可选）
}
```

### 1.3 状态码说明

| 状态码 | 说明 | 描述 |
|--------|------|------|
| 200 | 成功 | 操作成功 |
| 400 | 参数错误 | 请求参数不正确 |
| 401 | 未授权 | 未提供认证信息或认证失败 |
| 403 | 权限不足 | 没有执行此操作的权限 |
| 404 | 资源不存在 | 请求的资源不存在 |
| 409 | 资源冲突 | 资源已存在或状态冲突 |
| 429 | 请求过多 | 请求频率超过限制 |
| 500 | 服务器错误 | 服务器内部错误 |

### 1.4 业务错误码

| 错误码 | 说明 |
|--------|------|
| 1001 | 用户不存在 |
| 1002 | 用户已存在 |
| 1003 | 凭据无效 |
| 1004 | 用户被封禁 |
| 1005 | 邮箱未验证 |
| 1006 | 密码太弱 |
| 1007 | 账户被锁定 |
| 1008 | 无效令牌 |
| 1009 | 令牌过期 |
| 1010 | 权限拒绝 |

## 2. 用户认证接口

### 2.1 用户注册

**接口描述**: 新用户注册账户

**请求信息**:
- **请求方式**: POST
- **请求路径**: `/api/v1/auth/register`
- **请求头**: `Content-Type: application/json`

**请求参数**:
```json
{
    "username": "student123",           // 用户名，3-50字符，字母数字下划线
    "email": "student@example.com",     // 邮箱地址
    "password": "SecurePass123!",       // 密码，最少8位
    "confirm_password": "SecurePass123!", // 确认密码
    "role": "student"                   // 角色：student/teacher
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "注册成功",
  "data": {
    "user_id": 1001,
        "username": "student123",
    "email": "student@example.com",
        "real_name": "",
        "avatar_url": "",
        "bio": "",
    "role": "student",
        "status": "active",
        "email_verified": false,
        "login_count": 0,
        "last_login_at": "",
        "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### 2.2 用户登录

**接口描述**: 用户身份验证，获取访问令牌

**请求信息**:
- **请求方式**: POST
- **请求路径**: `/api/v1/auth/login`
- **请求头**: `Content-Type: application/json`

**请求参数**:
```json
{
    "username": "student123",    // 用户名或邮箱
    "password": "SecurePass123!" // 密码
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "token_type": "Bearer",
    "expires_in": 3600,
        "user_info": {
      "user_id": 1001,
            "username": "student123",
      "email": "student@example.com",
            "real_name": "张三",
            "avatar_url": "https://example.com/avatar.jpg",
            "bio": "热爱编程的学生",
            "role": "student",
            "status": "active",
            "email_verified": true,
            "login_count": 15,
            "last_login_at": "2024-01-15T10:30:00Z",
            "created_at": "2024-01-10T08:00:00Z"
    }
  }
}
```

### 2.3 刷新令牌

**接口描述**: 使用刷新令牌获取新的访问令牌

**请求信息**:
- **请求方式**: POST
- **请求路径**: `/api/v1/auth/refresh`
- **请求头**: `Content-Type: application/json`

**请求参数**:
```json
{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应示例**:
```json
{
  "code": 200,
    "message": "令牌刷新成功",
  "data": {
        "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "token_type": "Bearer",
        "expires_in": 3600
    }
}
```

### 2.4 用户登出

**接口描述**: 用户会话注销，撤销令牌

**请求信息**:
- **请求方式**: POST
- **请求路径**: `/api/v1/auth/logout`
- **请求头**: `Authorization: Bearer {access_token}`

**响应示例**:
```json
{
    "code": 200,
    "message": "登出成功"
}
```

### 2.5 权限验证

**接口描述**: 验证用户是否有执行特定操作的权限

**请求信息**:
- **请求方式**: POST
- **请求路径**: `/api/v1/auth/verify-permission`
- **请求头**: `Authorization: Bearer {access_token}`

**请求参数**:
```json
{
    "resource": "problem:1001",  // 资源标识
    "action": "read"             // 操作类型：create/read/update/delete
}
```

**响应示例**:
```json
{
    "code": 200,
    "message": "权限验证通过"
}
```

## 3. 用户信息管理接口

### 3.1 获取个人信息

**接口描述**: 获取当前登录用户的详细信息

**请求信息**:
- **请求方式**: GET
- **请求路径**: `/api/v1/users/profile`
- **请求头**: `Authorization: Bearer {access_token}`

**响应示例**:
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
        "user_id": 1001,
        "username": "student123",
        "email": "student@example.com",
        "real_name": "张三",
        "avatar_url": "https://example.com/avatar.jpg",
        "bio": "热爱编程的学生",
        "role": "student",
        "status": "active",
        "email_verified": true,
        "login_count": 15,
        "last_login_at": "2024-01-15T10:30:00Z",
        "created_at": "2024-01-10T08:00:00Z"
  }
}
```

### 3.2 更新个人信息

**接口描述**: 更新用户的基本信息

**请求信息**:
- **请求方式**: PUT
- **请求路径**: `/api/v1/users/profile`
- **请求头**: `Authorization: Bearer {access_token}`, `Content-Type: application/json`

**请求参数**:
```json
{
    "real_name": "张三",                           // 真实姓名（可选）
    "avatar_url": "https://example.com/avatar.jpg", // 头像链接（可选）
    "bio": "热爱编程的学生"                        // 个人简介（可选）
}
```

**响应示例**:
```json
{
    "code": 200,
    "message": "更新成功",
  "data": {
        "user_id": 1001,
        "username": "student123",
        "email": "student@example.com",
        "real_name": "张三",
        "avatar_url": "https://example.com/avatar.jpg",
        "bio": "热爱编程的学生",
        "role": "student",
        "status": "active",
        "email_verified": true,
        "login_count": 15,
        "last_login_at": "2024-01-15T10:30:00Z",
        "created_at": "2024-01-10T08:00:00Z"
  }
}
```

### 3.3 修改密码

**接口描述**: 修改用户登录密码

**请求信息**:
- **请求方式**: PUT
- **请求路径**: `/api/v1/users/password`
- **请求头**: `Authorization: Bearer {access_token}`, `Content-Type: application/json`

**请求参数**:
```json
{
    "current_password": "OldPass123!",    // 当前密码
    "new_password": "NewPass123!",        // 新密码
    "confirm_password": "NewPass123!"     // 确认新密码
}
```

**响应示例**:
```json
{
    "code": 200,
    "message": "密码修改成功"
}
```

### 3.4 获取用户统计

**接口描述**: 获取指定用户的统计信息

**请求信息**:
- **请求方式**: GET
- **请求路径**: `/api/v1/users/{user_id}/stats`
- **请求头**: `Authorization: Bearer {access_token}`

**路径参数**:
- `user_id`: 用户ID

**响应示例**:
```json
{
    "code": 200,
    "message": "获取成功",
  "data": {
        "total_submissions": 150,      // 总提交次数
        "accepted_submissions": 89,    // 通过提交次数
        "solved_problems": 75,         // 解决题目数
        "easy_solved": 35,             // 简单题解决数
        "medium_solved": 30,           // 中等题解决数
        "hard_solved": 10,             // 困难题解决数
        "current_rating": 1450,        // 当前评分
        "max_rating": 1520,            // 最高评分
        "rank_level": "silver",        // 段位等级
        "contest_participated": 8,     // 参赛次数
        "contest_won": 2               // 获胜次数
  }
}
```

### 3.5 获取用户权限

**接口描述**: 获取指定用户的权限列表

**请求信息**:
- **请求方式**: GET
- **请求路径**: `/api/v1/users/{user_id}/permissions`
- **请求头**: `Authorization: Bearer {access_token}`

**路径参数**:
- `user_id`: 用户ID

**响应示例**:
```json
{
    "code": 200,
    "message": "获取成功",
  "data": {
        "permissions": [
            "user:profile:read",
            "user:profile:update",
            "user:password:change",
            "problem:read",
            "submission:create",
            "submission:read:own",
            "contest:participate"
    ]
  }
}
```

## 4. 管理员接口

### 4.1 用户列表

**接口描述**: 获取用户列表，支持分页和筛选（管理员权限）

**请求信息**:
- **请求方式**: GET
- **请求路径**: `/api/v1/users`
- **请求头**: `Authorization: Bearer {access_token}`

**查询参数**:
- `page`: 页码，默认1
- `page_size`: 页大小，默认20，最大100
- `role`: 角色筛选，可选值：student/teacher/admin
- `status`: 状态筛选，可选值：active/inactive/banned
- `keyword`: 关键词搜索（用户名、邮箱、真实姓名）

**请求示例**:
```
GET /api/v1/users?page=1&page_size=20&role=student&keyword=张
```

**响应示例**:
```json
{
    "code": 200,
    "message": "获取成功",
    "data": {
        "users": [
            {
                "user_id": 1001,
                "username": "student123",
                "email": "student@example.com",
                "real_name": "张三",
                "avatar_url": "https://example.com/avatar.jpg",
                "bio": "热爱编程的学生",
                "role": "student",
                "status": "active",
                "email_verified": true,
                "login_count": 15,
                "last_login_at": "2024-01-15T10:30:00Z",
                "created_at": "2024-01-10T08:00:00Z"
            }
        ],
        "total": 150
    }
}
```

### 4.2 更新用户角色

**接口描述**: 修改指定用户的角色权限（管理员权限）

**请求信息**:
- **请求方式**: PUT
- **请求路径**: `/api/v1/users/{user_id}/role`
- **请求头**: `Authorization: Bearer {access_token}`, `Content-Type: application/json`

**路径参数**:
- `user_id`: 用户ID

**请求参数**:
```json
{
    "role": "teacher"    // 新角色：student/teacher/admin
}
```

**响应示例**:
```json
{
    "code": 200,
    "message": "角色更新成功"
}
```

## 5. 错误处理

### 5.1 参数验证错误

**错误示例**:
```json
{
  "code": 400,
    "message": "用户名长度必须在3-50个字符之间"
}
```

### 5.2 认证错误

**错误示例**:
```json
{
    "code": 401,
    "message": "令牌已过期"
}
```

### 5.3 权限错误

**错误示例**:
```json
{
    "code": 403,
    "message": "需要管理员权限"
}
```

### 5.4 业务错误

**错误示例**:
```json
{
    "code": 1002,
    "message": "用户名已存在"
}
```

## 6. 接口测试

### 6.1 Postman测试集合

可以导入以下Postman测试集合进行接口测试：

```json
{
    "info": {
        "name": "OJ用户服务API",
        "description": "在线判题系统用户服务接口测试"
    },
    "item": [
        {
            "name": "用户注册",
            "request": {
                "method": "POST",
                "header": [
                    {
                        "key": "Content-Type",
                        "value": "application/json"
                    }
                ],
                "body": {
                    "mode": "raw",
                    "raw": "{\n    \"username\": \"testuser\",\n    \"email\": \"test@example.com\",\n    \"password\": \"TestPass123!\",\n    \"confirm_password\": \"TestPass123!\",\n    \"role\": \"student\"\n}"
                },
                "url": {
                    "raw": "{{baseUrl}}/api/v1/auth/register",
                    "host": ["{{baseUrl}}"],
                    "path": ["api", "v1", "auth", "register"]
                }
            }
        }
    ],
    "variable": [
        {
            "key": "baseUrl",
            "value": "http://localhost:8888"
        }
    ]
}
```

### 6.2 测试用例

1. **注册新用户**
   - 使用有效参数注册
   - 验证响应包含用户信息
   - 验证用户名和邮箱唯一性

2. **用户登录**
   - 使用正确的用户名密码登录
   - 验证返回的JWT令牌
   - 测试错误的凭据

3. **访问受保护的接口**
   - 使用有效令牌访问用户信息
   - 测试无令牌访问
   - 测试过期令牌

4. **权限控制**
   - 测试普通用户访问管理员接口
   - 验证角色权限控制

## 7. SDK和代码示例

### 7.1 JavaScript示例

```javascript
// 用户注册
async function register(userData) {
    const response = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(userData)
    });
    return await response.json();
}

// 用户登录
async function login(credentials) {
    const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(credentials)
    });
    const data = await response.json();
    if (data.code === 200) {
        localStorage.setItem('access_token', data.data.access_token);
    }
    return data;
}

// 获取用户信息
async function getUserProfile() {
    const token = localStorage.getItem('access_token');
    const response = await fetch('/api/v1/users/profile', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    });
    return await response.json();
}
```

### 7.2 Go客户端示例

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

func login(username, password string) error {
    loginReq := LoginRequest{
        Username: username,
        Password: password,
    }
    
    jsonData, _ := json.Marshal(loginReq)
    resp, err := http.Post("http://localhost:8888/api/v1/auth/login", 
        "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // 处理响应
    return nil
}
```

## 8. 版本更新日志

### v1.0.0 (2024-01-15)
- 初始版本发布
- 实现用户认证基础功能
- 支持用户注册、登录、权限管理
- 完整的JWT令牌机制

### 后续版本计划
- v1.1.0: 添加邮箱验证功能
- v1.2.0: 支持第三方登录(OAuth)
- v1.3.0: 增加双因子认证(2FA)
- v2.0.0: 用户行为分析和推荐系统
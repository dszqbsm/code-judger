# UpdateProblem 权限验证完整指南

## 概述

UpdateProblem方法现已完整实现JWT用户信息获取和权限验证逻辑，确保只有有权限的用户才能修改题目。

## 权限验证流程

### 1. 用户身份认证
- 从JWT令牌中提取用户信息（用户ID、用户名、角色等）
- 支持两种方式获取用户信息：
  - 从go-zero的JWT上下文获取
  - 从HTTP请求头的Authorization Bearer令牌获取

### 2. 权限验证规则
- **管理员 (admin)**：可以修改任何题目
- **教师 (teacher)**：只能修改自己创建的题目
- **学生 (student)**：无权限修改题目

### 3. 验证步骤
1. 验证题目ID有效性
2. 获取并验证用户JWT令牌
3. 查询题目是否存在
4. 检查题目是否被删除
5. 验证用户权限（基于角色和题目创建者）
6. 验证更新参数
7. 执行更新操作
8. 记录操作日志

## 使用示例

### 1. 教师修改自己创建的题目

**请求头：**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

**请求体：**
```json
PUT /api/v1/problems/1001
{
  "title": "更新后的题目标题",
  "description": "更新后的题目描述...",
  "difficulty": "medium",
  "time_limit": 2000,
  "memory_limit": 256,
  "languages": ["cpp", "java", "python"],
  "tags": ["算法", "数据结构", "动态规划"],
  "is_public": true
}
```

**成功响应：**
```json
{
  "code": 200,
  "message": "题目更新成功",
  "data": {
    "problem_id": 1001,
    "updated_at": "2024-01-15T10:30:00Z07:00",
    "message": "题目信息已更新"
  }
}
```

### 2. 学生尝试修改题目（权限不足）

**请求头：**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**错误响应：**
```json
{
  "code": 403,
  "message": "权限不足：无题目修改权限"
}
```

### 3. 教师尝试修改其他人创建的题目

**错误响应：**
```json
{
  "code": 403,
  "message": "权限不足：只能修改自己创建的题目"
}
```

### 4. 管理员修改任何题目

**成功响应：**
```json
{
  "code": 200,
  "message": "题目更新成功",
  "data": {
    "problem_id": 1001,
    "updated_at": "2024-01-15T10:30:00Z07:00",
    "message": "题目信息已更新"
  }
}
```

## 错误处理

### 认证错误
- **401 Unauthorized**：JWT令牌无效或缺失
- **401 Unauthorized**：令牌解析失败

### 权限错误
- **403 Forbidden**：用户角色不允许修改题目
- **403 Forbidden**：教师尝试修改非自己创建的题目

### 业务错误
- **400 Bad Request**：题目ID无效
- **400 Bad Request**：更新参数验证失败
- **404 Not Found**：题目不存在或已被删除
- **500 Internal Server Error**：系统内部错误

## 日志记录

系统会记录详细的操作日志：

### 成功操作
```
题目更新成功: ID=1001, 标题=算法基础题, 更新者=teacher1 (ID: 2001), 难度=medium, 公开=true
```

### 权限验证失败
```
用户 student1 (ID: 3001) 权限验证失败，无法修改题目 1001: 权限不足：无题目修改权限
```

### 尝试修改已删除题目
```
用户 teacher1 (ID: 2001) 尝试修改已删除的题目: ID=1001
```

## 安全特性

1. **JWT令牌验证**：确保请求来自已认证用户
2. **角色权限控制**：基于用户角色限制操作权限
3. **资源所有权验证**：教师只能修改自己创建的题目
4. **操作日志记录**：完整记录所有修改操作和权限验证结果
5. **输入验证**：对所有更新参数进行严格验证
6. **错误信息控制**：避免泄露敏感系统信息

## 配置要求

确保在 `problem-api.yaml` 中正确配置JWT密钥：

```yaml
Auth:
  AccessSecret: "your-access-secret-key-for-problem-service"
  AccessExpire: 3600  # 1小时
```

## 技术实现要点

1. **双重用户信息获取**：支持从上下文和请求头两种方式获取用户信息
2. **权限验证中间件**：使用统一的权限验证函数
3. **详细错误处理**：提供清晰的错误信息和日志记录
4. **事务安全**：确保更新操作的原子性
5. **缓存更新**：更新题目时自动清理相关缓存

这样的设计确保了UpdateProblem方法的安全性、可维护性和用户体验。






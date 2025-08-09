# API接口文档

## 概述

本文档详细描述了在线判题系统的RESTful API接口规范，包括用户管理、题目管理、提交管理、判题核心和比赛系统等模块的所有API接口。

### API基础信息

- **Base URL**: `https://api.oj.example.com`
- **API版本**: v1
- **认证方式**: JWT Token
- **数据格式**: JSON
- **字符编码**: UTF-8

### 通用响应格式

所有API接口都遵循统一的响应格式：

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {
    // 具体的响应数据
  },
  "timestamp": "2024-01-01T10:00:00Z"
}
```

### 状态码说明

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未授权访问 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 用户管理API

### 用户认证相关

| 接口 | 方法 | 路径 | 功能 | 优先级 |
|------|------|------|------|--------|
| 用户注册 | POST | `/api/v1/auth/register` | 用户注册 | 核心功能 |
| 用户登录 | POST | `/api/v1/auth/login` | 用户登录 | 核心功能 |
| 用户登出 | POST | `/api/v1/auth/logout` | 用户登出 | 核心功能 |
| 刷新Token | POST | `/api/v1/auth/refresh` | 刷新访问令牌 | 核心功能 |

#### 用户注册

**请求示例**:
```json
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "student001",
  "email": "student@example.com",
  "password": "password123",
  "role": "student"
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "注册成功",
  "data": {
    "user_id": 1001,
    "username": "student001",
    "email": "student@example.com",
    "role": "student",
    "created_at": "2024-01-01T10:00:00Z"
  }
}
```

#### 用户登录

**请求示例**:
```json
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "student001",
  "password": "password123"
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
    "expires_in": 3600,
    "user": {
      "user_id": 1001,
      "username": "student001",
      "email": "student@example.com",
      "role": "student"
    }
  }
}
```

### 用户信息管理

| 接口 | 方法 | 路径 | 功能 | 优先级 |
|------|------|------|------|--------|
| 获取用户信息 | GET | `/api/v1/users/{id}` | 获取用户详情 | 核心功能 |
| 更新用户信息 | PUT | `/api/v1/users/{id}` | 更新用户信息 | 核心功能 |
| 用户列表 | GET | `/api/v1/users` | 获取用户列表 | 扩展功能 |
| 用户统计 | GET | `/api/v1/users/{id}/stats` | 获取用户统计信息 | 扩展功能 |

#### 获取用户信息

**请求示例**:
```http
GET /api/v1/users/1001
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**响应示例**:
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "user_id": 1001,
    "username": "student001",
    "email": "student@example.com",
    "role": "student",
    "avatar_url": "https://cdn.example.com/avatars/1001.jpg",
    "created_at": "2024-01-01T10:00:00Z",
    "last_login": "2024-01-15T14:30:00Z",
    "stats": {
      "total_submissions": 156,
      "accepted_submissions": 89,
      "acceptance_rate": 57.05
    }
  }
}
```

---

## 题目管理API

| 接口 | 方法 | 路径 | 功能 | 优先级 |
|------|------|------|------|--------|
| 创建题目 | POST | `/api/v1/problems` | 创建新题目 | 核心功能 |
| 获取题目列表 | GET | `/api/v1/problems` | 获取题目列表 | 核心功能 |
| 获取题目详情 | GET | `/api/v1/problems/{id}` | 获取题目详情 | 核心功能 |
| 更新题目 | PUT | `/api/v1/problems/{id}` | 更新题目信息 | 核心功能 |
| 删除题目 | DELETE | `/api/v1/problems/{id}` | 删除题目 | 核心功能 |
| 上传测试数据 | POST | `/api/v1/problems/{id}/testdata` | 上传测试数据 | 核心功能 |

#### 创建题目

**请求示例**:
```json
POST /api/v1/problems
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "title": "两数之和",
  "description": "给定一个整数数组nums和一个目标值target，请你在该数组中找出和为目标值的那两个整数，并返回它们的数组下标。",
  "input_format": "第一行包含数组长度n和目标值target\n第二行包含n个整数",
  "output_format": "输出两个整数的下标，用空格分隔",
  "sample_input": "4 9\n2 7 11 15",
  "sample_output": "0 1",
  "difficulty": "easy",
  "time_limit": 1000,
  "memory_limit": 128,
  "languages": ["cpp", "java", "python", "go"],
  "tags": ["数组", "哈希表"]
}
```

**响应示例**:
```json
{
  "code": 201,
  "message": "题目创建成功",
  "data": {
    "problem_id": 1001,
    "title": "两数之和",
    "difficulty": "easy",
    "created_at": "2024-01-01T10:00:00Z",
    "created_by": 1
  }
}
```

#### 获取题目列表

**请求示例**:
```http
GET /api/v1/problems?page=1&page_size=20&difficulty=easy&tag=数组
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认20 |
| difficulty | string | 否 | 难度筛选：easy/medium/hard |
| tag | string | 否 | 标签筛选 |
| keyword | string | 否 | 关键词搜索 |

**响应示例**:
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "problems": [
      {
        "problem_id": 1001,
        "title": "两数之和",
        "difficulty": "easy",
        "acceptance_rate": 45.2,
        "total_submissions": 1250,
        "accepted_submissions": 565,
        "tags": ["数组", "哈希表"]
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 156,
      "total_pages": 8
    }
  }
}
```

---

## 提交管理API

| 接口 | 方法 | 路径 | 功能 | 优先级 |
|------|------|------|------|--------|
| 提交代码 | POST | `/api/v1/submissions` | 提交代码判题 | 核心功能 |
| 获取提交列表 | GET | `/api/v1/submissions` | 获取提交记录 | 核心功能 |
| 获取提交详情 | GET | `/api/v1/submissions/{id}` | 获取提交详情 | 核心功能 |
| 重新判题 | POST | `/api/v1/submissions/{id}/rejudge` | 重新判题 | 扩展功能 |

#### 提交代码

**请求示例**:
```json
POST /api/v1/submissions
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "problem_id": 1001,
  "language": "cpp",
  "code": "#include <iostream>\n#include <vector>\n#include <unordered_map>\nusing namespace std;\n\nclass Solution {\npublic:\n    vector<int> twoSum(vector<int>& nums, int target) {\n        unordered_map<int, int> map;\n        for (int i = 0; i < nums.size(); i++) {\n            int complement = target - nums[i];\n            if (map.count(complement)) {\n                return {map[complement], i};\n            }\n            map[nums[i]] = i;\n        }\n        return {};\n    }\n};"
}
```

**响应示例**:
```json
{
  "code": 201,
  "message": "代码提交成功",
  "data": {
    "submission_id": 10001,
    "status": "pending",
    "created_at": "2024-01-01T10:00:00Z",
    "estimated_time": "30s"
  }
}
```

---

## 判题核心API

| 接口 | 方法 | 路径 | 功能 | 优先级 |
|------|------|------|------|--------|
| 判题状态查询 | GET | `/api/v1/judge/status/{submission_id}` | 查询判题状态 | 核心功能 |
| 判题结果详情 | GET | `/api/v1/judge/result/{submission_id}` | 获取判题结果 | 核心功能 |
| 系统负载查询 | GET | `/api/v1/judge/load` | 查询系统负载 | 扩展功能 |

---

## 比赛系统API

| 接口 | 方法 | 路径 | 功能 | 优先级 |
|------|------|------|------|--------|
| 创建比赛 | POST | `/api/v1/contests` | 创建比赛 | 扩展功能 |
| 获取比赛列表 | GET | `/api/v1/contests` | 获取比赛列表 | 扩展功能 |
| 参加比赛 | POST | `/api/v1/contests/{id}/join` | 参加比赛 | 扩展功能 |
| 比赛排行榜 | GET | `/api/v1/contests/{id}/ranklist` | 获取排行榜 | 扩展功能 |

---

## WebSocket实时通信

### 连接地址
```
wss://api.oj.example.com/ws/judge/{user_id}?token={jwt_token}
```

### 消息格式

#### 判题状态推送
```json
{
  "type": "judge_status",
  "data": {
    "submission_id": 10001,
    "status": "judging",
    "progress": 60,
    "current_test": 6,
    "total_tests": 10
  }
}
```

#### 判题结果推送
```json
{
  "type": "judge_result",
  "data": {
    "submission_id": 10001,
    "status": "accepted",
    "score": 100,
    "time_used": 156,
    "memory_used": 2048,
    "test_results": [
      {
        "test_id": 1,
        "status": "accepted",
        "time_used": 15,
        "memory_used": 1024
      }
    ]
  }
}
```

---

## 错误处理

### 错误响应格式
```json
{
  "code": 400,
  "message": "请求参数错误",
  "error": {
    "type": "ValidationError",
    "details": [
      {
        "field": "email",
        "message": "邮箱格式不正确"
      }
    ]
  }
}
```

### 常见错误码
| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 1001 | 用户名已存在 | 更换用户名 |
| 1002 | 邮箱已被注册 | 更换邮箱或找回密码 |
| 2001 | 题目不存在 | 检查题目ID |
| 3001 | 代码编译失败 | 检查代码语法 |
| 4001 | 比赛未开始 | 等待比赛开始 |
| 5001 | 系统维护中 | 稍后重试 |

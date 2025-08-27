# 判题服务API测试指南

## 概述

本指南提供了在线判题系统判题服务的完整API测试用例，包括正常场景、异常场景和边界条件测试。

## 服务信息

- **服务地址**: `http://localhost:8889`
- **API前缀**: `/api/v1/judge`
- **数据格式**: JSON

## 导入Postman集合

1. 打开Postman
2. 点击左上角的"Import"按钮
3. 选择"File"选项卡
4. 上传`postman_test_cases.json`文件
5. 点击"Import"完成导入

## 环境变量设置

在Postman中设置以下环境变量：

| 变量名 | 值 | 描述 |
|--------|----|----|
| baseUrl | http://localhost:8889 | 服务基础地址 |
| submission_id | 1001 | 测试用的提交ID |

## API接口详细说明

### 1. 判题核心接口

#### 1.1 提交判题任务
- **接口**: `POST /api/v1/judge/submit`
- **功能**: 提交代码进行判题
- **请求体示例**:
```json
{
  "submission_id": 1001,
  "problem_id": 101,
  "user_id": 123,
  "language": "cpp",
  "code": "#include <iostream>\nusing namespace std;\n\nint main() {\n    int a, b;\n    cin >> a >> b;\n    cout << a + b << endl;\n    return 0;\n}"
}
```

#### 1.2 查询判题结果
- **接口**: `GET /api/v1/judge/result/{submission_id}`
- **功能**: 获取判题的详细结果
- **路径参数**: submission_id (提交ID)

#### 1.3 查询判题状态
- **接口**: `GET /api/v1/judge/status/{submission_id}`
- **功能**: 获取判题进度状态
- **路径参数**: submission_id (提交ID)

#### 1.4 取消判题任务
- **接口**: `DELETE /api/v1/judge/cancel/{submission_id}`
- **功能**: 取消正在进行的判题任务
- **路径参数**: submission_id (提交ID)

#### 1.5 重新判题
- **接口**: `POST /api/v1/judge/rejudge/{submission_id}`
- **功能**: 重新执行判题任务
- **路径参数**: submission_id (提交ID)

### 2. 系统管理接口

#### 2.1 获取判题节点状态
- **接口**: `GET /api/v1/judge/nodes`
- **功能**: 获取所有判题节点的运行状态

#### 2.2 获取判题队列状态
- **接口**: `GET /api/v1/judge/queue`
- **功能**: 获取判题队列的统计信息

#### 2.3 健康检查
- **接口**: `GET /api/v1/judge/health`
- **功能**: 检查服务健康状态

#### 2.4 获取支持的编程语言
- **接口**: `GET /api/v1/judge/languages`
- **功能**: 获取系统支持的编程语言配置

## 支持的编程语言

根据配置文件，系统支持以下编程语言：

| 语言 | 标识符 | 文件扩展名 | 编译器版本 |
|------|--------|-----------|-----------|
| C++ | cpp | .cpp | g++ 9.4.0 |
| C | c | .c | gcc 9.4.0 |
| Java | java | .java | OpenJDK 11.0.16 |
| Python | python | .py | Python 3.8.10 |
| Go | go | .go | Go 1.19.8 |
| JavaScript | javascript | .js | Node.js 16.15.1 |

## 测试用例分类

### 1. 正常功能测试
- 提交不同语言的正确代码
- 查询判题结果和状态
- 系统信息查询

### 2. 参数验证测试
- 缺少必填字段
- 无效的编程语言
- 超出范围的参数值

### 3. 编译错误测试
- C++语法错误
- Python语法错误
- Java编译错误

### 4. 运行时错误测试
- 时间限制超时（无限循环）
- 内存限制超时（内存泄漏）
- 运行时异常（数组越界等）

### 5. 边界条件测试
- 查询不存在的提交ID
- 极长代码提交
- 特殊字符处理

## 响应状态码

| 状态码 | 含义 | 说明 |
|--------|------|------|
| 200 | 成功 | 请求处理成功 |
| 400 | 客户端错误 | 请求参数错误 |
| 404 | 未找到 | 请求的资源不存在 |
| 500 | 服务器错误 | 服务器内部错误 |

## 典型响应格式

### 成功响应
```json
{
  "code": 200,
  "message": "success",
  "data": {
    // 具体数据内容
  }
}
```

### 错误响应
```json
{
  "code": 400,
  "message": "参数验证失败"
}
```

## 测试建议

### 1. 测试顺序建议
1. 先运行健康检查，确保服务正常
2. 获取支持的编程语言列表
3. 提交简单的正确代码测试
4. 查询判题结果和状态
5. 测试各种错误情况

### 2. 注意事项
- 确保服务已启动并监听8889端口
- 测试前确认数据库连接正常
- 大批量测试时注意服务器负载
- 保存重要的测试结果用于问题排查

### 3. 性能测试建议
- 并发提交测试
- 长时间运行稳定性测试
- 内存和CPU使用率监控
- 队列处理能力测试

## 常见问题

### Q: 提交后一直显示pending状态？
A: 检查判题引擎是否正常运行，查看服务日志确认具体原因。

### Q: 编译错误但没有详细信息？
A: 查看compile_info字段，包含详细的编译错误信息。

### Q: 某些语言不支持？
A: 检查语言配置是否正确，确认编译器已安装。

## 监控和调试

### 日志查看
```bash
# 查看服务日志
docker logs oj-judge-api

# 查看实时日志
docker logs -f oj-judge-api
```

### 指标监控
访问 `http://localhost:9091/metrics` 查看Prometheus指标。

### 数据库查询
使用之前配置的MySQL连接查看提交记录：
```sql
-- 查看最近的提交记录
SELECT * FROM submissions ORDER BY created_at DESC LIMIT 10;

-- 查看判题统计
SELECT status, COUNT(*) as count FROM submissions GROUP BY status;
```

## 自动化测试脚本

可以使用Newman（Postman的命令行工具）进行自动化测试：

```bash
# 安装Newman
npm install -g newman

# 运行测试集合
newman run postman_test_cases.json
```

---

通过以上测试用例和指南，你可以全面测试判题服务的各项功能，确保系统的稳定性和可靠性。


# 判题服务架构设计

## 消息队列 vs 任务队列的区别

### 1. 消息队列 (Kafka) - 服务间解耦

**作用**: 提交服务 ↔ 判题服务 之间的异步通信

**职责**:
- 服务间消息传递和解耦
- 持久化存储，保证消息不丢失
- 支持多个消费者和负载均衡
- 提供消息顺序保证和重播能力

**Kafka Topics**:
- `judge_tasks`: 接收来自提交服务的判题任务
- `judge_results`: 发送判题结果回提交服务
- `judge_status`: 发送判题状态更新
- `judge_dead_letter`: 处理失败消息的死信队列

### 2. 任务队列 (TaskScheduler) - 服务内任务调度

**作用**: 判题服务内部的任务管理和执行

**职责**:
- 任务优先级排序和调度
- 资源限制和并发控制
- 任务状态管理和重试机制
- Worker池管理和负载均衡

**特点**:
- 内存中的优先级队列
- 支持任务取消和重新判题
- 实时状态监控和进度反馈

## 完整的业务流程

### 正确的调用链路

```
用户 → 提交服务 → [Kafka消息队列] → 判题服务 → [内部任务队列] → Worker执行
     ↑                                    ↓
     └─── [Kafka结果队列] ← 判题完成 ←─────┘
```

### 详细流程

1. **用户提交代码**
   - 用户调用提交服务API: `POST /api/v1/submissions`
   - 提交服务验证权限、保存提交记录
   - 从题目服务获取测试用例
   - 发送判题任务到Kafka `judge_tasks` topic

2. **判题服务处理**
   - Kafka消费者接收任务消息
   - 验证任务数据完整性
   - 提交到内部任务调度器 (TaskScheduler)
   - 任务进入优先级队列等待执行

3. **任务执行**
   - Worker从任务队列获取任务
   - 执行编译和测试用例
   - 实时更新任务状态
   - 完成后发送结果到Kafka

4. **结果回传**
   - 判题结果发送到 `judge_results` topic
   - 提交服务消费结果，更新数据库
   - 通过WebSocket推送给用户

## 为什么用户不能直接调用判题服务？

### 架构原因
1. **职责分离**: 提交服务负责业务逻辑，判题服务专注计算
2. **安全隔离**: 判题服务运行在内网，不对外暴露
3. **负载均衡**: 通过消息队列实现水平扩展
4. **容错性**: 消息队列提供重试和故障恢复

### 业务原因
1. **权限验证**: 提交服务检查用户权限和题目访问权限
2. **数据持久化**: 提交记录需要保存到数据库
3. **统计分析**: 提交服务收集用户行为数据
4. **限流控制**: 防止用户恶意提交大量任务

## 消息格式定义

### 判题任务消息 (judge_tasks)
```json
{
  "submission_id": 12345,
  "problem_id": 1001,
  "user_id": 5,
  "language": "cpp",
  "code": "#include<iostream>...",
  "time_limit": 1000,
  "memory_limit": 128,
  "test_cases": [
    {
      "case_id": 1,
      "input": "3 4",
      "expected_output": "7"
    }
  ],
  "priority": 3,
  "created_at": "2025-09-02T09:00:00Z"
}
```

### 判题结果消息 (judge_results)
```json
{
  "submission_id": 12345,
  "status": "completed",
  "result": {
    "verdict": "accepted",
    "score": 100,
    "time_used": 150,
    "memory_used": 2048,
    "test_cases": [
      {
        "case_id": 1,
        "status": "accepted",
        "time_used": 150,
        "memory_used": 2048,
        "input": "3 4",
        "output": "7",
        "expected": "7"
      }
    ]
  },
  "compile_info": {
    "success": true,
    "message": "",
    "time": 500
  },
  "timestamp": 1756775524
}
```

## 配置说明

### Kafka配置
```yaml
KafkaConf:
  Brokers:
    - localhost:9092
  Topic: judge_tasks              # 接收判题任务
  Group: judge_consumer_group     # 消费者组
  ResultTopic: judge_results      # 发送判题结果
  StatusTopic: judge_status       # 发送状态更新
  DeadLetterTopic: judge_dead_letter # 死信队列
```

### 任务队列配置
```yaml
TaskQueue:
  MaxWorkers: 10              # 最大工作协程数
  QueueSize: 1000            # 队列大小
  TaskTimeout: 300           # 任务超时时间(秒)
  RetryTimes: 3              # 重试次数
  AverageTaskTime: 30        # 平均任务执行时间(秒)
```

## 监控和故障处理

### 关键指标
- Kafka消费延迟
- 任务队列长度
- Worker利用率
- 判题成功率
- 平均执行时间

### 故障处理
- 消息处理失败 → 死信队列
- 任务执行超时 → 自动重试
- 服务重启 → Kafka自动重新分配分区
- 数据丢失 → Kafka持久化保证

## 扩展性设计

### 水平扩展
- 增加判题服务实例 → Kafka自动负载均衡
- 增加Worker数量 → 提高并发处理能力
- 分区策略 → 按用户ID或题目ID分区

### 垂直扩展
- 增加服务器资源
- 优化编译器和执行环境
- 使用更快的存储设备

这种架构设计确保了系统的高可用性、可扩展性和容错性，同时保持了良好的性能和用户体验。





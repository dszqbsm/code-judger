# 等待时间计算的问题分析

## 当前实现的问题

### 1. **时序问题：任务已提交但队列状态未包含**

```go
// 在 submitjudgelogic.go 中的流程：
// 步骤7: 提交任务到调度器
l.svcCtx.TaskScheduler.SubmitTask(task)

// 步骤8: 获取队列状态
queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()

// 问题：此时获取的队列状态是否包含刚刚提交的任务？
```

### 2. **SubmitTask方法分析**

```go
func (s *TaskScheduler) SubmitTask(task *JudgeTask) error {
    // 1. 生成任务ID
    task.ID = fmt.Sprintf("task_%d_%d", task.SubmissionID, time.Now().UnixNano())
    
    // 2. 设置任务状态
    task.Status = TaskStatusPending
    task.CreatedAt = time.Now()
    
    // 3. 存储任务
    s.tasks.Store(task.ID, task)
    
    // 4. 加入优先级队列 ← 关键步骤
    s.priorityQueue.Push(task)
    
    // 5. 更新统计
    atomic.AddInt64(&s.stats.TotalTasks, 1)
    atomic.AddInt64(&s.stats.PendingTasks, 1)
    
    return nil
}
```

### 3. **GetQueueStatus方法分析**

```go
func (s *TaskScheduler) GetQueueStatus() *QueueStatus {
    queueItems := make([]*QueueItem, 0)
    
    // 遍历优先级队列中的任务
    s.priorityQueue.mutex.Lock()
    for _, task := range s.priorityQueue.tasks {
        if task.Status == TaskStatusPending {
            queueItems = append(queueItems, &QueueItem{
                SubmissionID:  task.SubmissionID,
                // ...
                EstimatedTime: s.estimateWaitTime(len(queueItems)), // 使用当前位置
            })
        }
    }
    s.priorityQueue.mutex.Unlock()
    
    return &QueueStatus{
        QueueLength: len(queueItems),  // 返回队列长度
        // ...
    }
}
```

## 核心问题分析

### 问题1：返回的不是当前任务的等待时间

```go
// 当前逻辑
queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()

return &types.SubmitJudgeData{
    QueuePosition: queueStatus.QueueLength,              // 队列总长度
    EstimatedTime: l.estimateWaitTime(queueStatus.QueueLength), // 基于总长度计算
}
```

**问题：**
- `queueStatus.QueueLength` 是队列中**所有任务**的数量
- 这**不是**当前任务在队列中的位置
- 返回的等待时间是**队列最后一个任务**的等待时间

### 问题2：时序不一致

```go
// 时间线分析
t1: task = SubmitTask(newTask)        // 新任务已加入队列
t2: status = GetQueueStatus()         // 获取队列状态（包含新任务）
t3: return EstimatedTime              // 返回的是整个队列的等待时间

// 结果：返回的不是新任务的等待时间，而是队列尾部的等待时间
```

## 正确的实现方式

### 方案1：在SubmitTask中返回位置信息

```go
// 修改SubmitTask方法返回任务在队列中的位置
func (s *TaskScheduler) SubmitTask(task *JudgeTask) (position int, err error) {
    // ... 原有逻辑 ...
    
    // 加入优先级队列并返回位置
    position = s.priorityQueue.PushAndGetPosition(task)
    
    return position, nil
}

// 修改PriorityQueue添加位置计算
func (pq *PriorityQueue) PushAndGetPosition(task *JudgeTask) int {
    pq.mutex.Lock()
    defer pq.mutex.Unlock()
    
    // 找到插入位置
    position := 0
    for i, t := range pq.tasks {
        if task.Priority < t.Priority ||
           (task.Priority == t.Priority && task.CreatedAt.Before(t.CreatedAt)) {
            pq.tasks = append(pq.tasks[:i], append([]*JudgeTask{task}, pq.tasks[i:]...)...)
            position = i
            break
        }
        position = i + 1
    }
    
    if position == len(pq.tasks) {
        pq.tasks = append(pq.tasks, task)
    }
    
    return position + 1  // 返回1基的位置（第1个、第2个...）
}
```

### 方案2：获取特定任务的队列位置

```go
// 添加获取特定任务位置的方法
func (s *TaskScheduler) GetTaskPosition(taskID string) (int, error) {
    s.priorityQueue.mutex.Lock()
    defer s.priorityQueue.mutex.Unlock()
    
    for i, task := range s.priorityQueue.tasks {
        if task.ID == taskID && task.Status == TaskStatusPending {
            return i + 1, nil  // 返回1基的位置
        }
    }
    
    return 0, fmt.Errorf("task not found in queue")
}
```

### 方案3：修改业务逻辑（推荐）

```go
func (l *SubmitJudgeLogic) SubmitJudge(req *types.SubmitJudgeReq) (resp *types.SubmitJudgeResp, err error) {
    // ... 前面的逻辑 ...
    
    // 7. 提交任务到调度器，并获取位置
    position, err := l.svcCtx.TaskScheduler.SubmitTaskWithPosition(task)
    if err != nil {
        return nil, err
    }
    
    // 8. 基于实际位置计算等待时间
    estimatedTime := l.estimateWaitTime(position)
    
    return &types.SubmitJudgeResp{
        BaseResp: types.BaseResp{
            Code:    200,
            Message: "判题任务已提交",
        },
        Data: types.SubmitJudgeData{
            SubmissionId:  req.SubmissionId,
            Status:        "pending",
            QueuePosition: position,        // 实际队列位置
            EstimatedTime: estimatedTime,   // 基于实际位置的等待时间
        },
    }, nil
}
```

## 等待时间计算公式

### 当前（错误）的计算

```go
// 返回的是队列最后位置的等待时间
queueLength := 10  // 假设队列有10个任务
estimatedTime := (10 / 3) * 30 = 100秒  // 假设3个工作器

// 但新任务可能在队列第5位，实际等待时间应该是：
actualTime := (5 / 3) * 30 = 50秒
```

### 正确的计算

```go
func calculateWaitTime(queuePosition, workerCount, avgTaskTime int) int {
    if workerCount <= 0 {
        return queuePosition * avgTaskTime
    }
    
    // 考虑并行处理
    return ((queuePosition - 1) / workerCount) * avgTaskTime
}

// 例子：
// 队列位置：第5个任务
// 工作器数：3个
// 平均任务时间：30秒
// 等待时间：((5-1) / 3) * 30 = 1 * 30 = 30秒
```

## 优先级的影响

```go
// 还需要考虑优先级对位置的影响
func (pq *PriorityQueue) GetTaskPosition(taskID string) int {
    position := 1
    for _, task := range pq.tasks {
        if task.ID == taskID {
            return position
        }
        if task.Status == TaskStatusPending {
            position++
        }
    }
    return -1  // 未找到
}
```

## 总结

**当前实现的问题：**
1. 返回的是队列总长度，不是任务实际位置
2. 等待时间计算基于错误的位置信息
3. 没有考虑优先级对队列位置的影响

**解决方案：**
1. 修改SubmitTask返回任务在队列中的实际位置
2. 基于实际位置计算等待时间
3. 考虑优先级和并行处理的影响

# 判题任务调度流程详细分析

## 整体架构概览

```
[用户提交] → [API层] → [业务逻辑] → [任务调度器] → [工作器池] → [判题引擎]
                                        ↓
                          [优先级队列] → [任务分发器] → [工作器]
```

## 详细的调度流程

### 1. 任务提交阶段

#### 步骤1：创建任务并提交到调度器
```go
// 在 submitjudgelogic.go 中
task := &scheduler.JudgeTask{
    SubmissionID: req.SubmissionId,
    ProblemID:    req.ProblemId,
    UserID:       req.UserId,
    Language:     req.Language,
    Code:         req.Code,
    TimeLimit:    problemInfo.TimeLimit,
    MemoryLimit:  problemInfo.MemoryLimit,
    TestCases:    testCases,
    Priority:     l.determinePriority(req.UserId),
}

// 提交任务到调度器
l.svcCtx.TaskScheduler.SubmitTask(task)
```

#### 步骤2：调度器接收任务
```go
func (s *TaskScheduler) SubmitTask(task *JudgeTask) error {
    // 1. 生成唯一任务ID
    task.ID = fmt.Sprintf("task_%d_%d", task.SubmissionID, time.Now().UnixNano())
    
    // 2. 设置任务超时上下文
    task.Context, task.CancelFunc = context.WithTimeout(s.ctx, timeout)
    task.Status = TaskStatusPending
    task.CreatedAt = time.Now()
    
    // 3. 存储任务到内存映射（用于快速查找）
    s.tasks.Store(task.ID, task)
    
    // 4. 加入优先级队列（等待调度）
    s.priorityQueue.Push(task)
    
    // 5. 更新统计信息
    atomic.AddInt64(&s.stats.TotalTasks, 1)
    atomic.AddInt64(&s.stats.PendingTasks, 1)
    
    return nil
}
```

### 2. 优先级队列管理

#### 优先级规则
```go
const (
    PriorityHigh   = 1 // 比赛任务（最高优先级）
    PriorityNormal = 2 // VIP用户任务
    PriorityLow    = 3 // 普通任务（最低优先级）
)
```

#### 队列插入逻辑
```go
func (pq *PriorityQueue) Push(task *JudgeTask) {
    // 线程安全操作
    pq.mutex.Lock()
    defer pq.mutex.Unlock()
    
    // 按优先级和创建时间排序插入
    for i, t := range pq.tasks {
        if task.Priority < t.Priority ||  // 数字越小优先级越高
           (task.Priority == t.Priority && task.CreatedAt.Before(t.CreatedAt)) {
            // 插入到合适位置
            pq.tasks = append(pq.tasks[:i], append([]*JudgeTask{task}, pq.tasks[i:]...)...)
            return
        }
    }
    
    // 如果优先级最低，插入到队尾
    pq.tasks = append(pq.tasks, task)
}
```

### 3. 任务分发阶段

#### 分发器循环运行
```go
func (s *TaskScheduler) dispatch() {
    ticker := time.NewTicker(100 * time.Millisecond)  // 每100ms检查一次
    defer ticker.Stop()
    
    for {
        select {
        case <-s.ctx.Done():
            return  // 调度器关闭
            
        case <-ticker.C:
            // 1. 从优先级队列获取最高优先级任务
            task := s.priorityQueue.Pop()
            if task == nil {
                continue  // 队列为空，继续等待
            }
            
            // 2. 检查任务状态
            if task.Status == TaskStatusCancelled {
                continue  // 任务已取消，跳过
            }
            
            // 3. 尝试分配给工作器
            select {
            case s.taskQueue <- task:
                // 成功分配给工作器
                logx.Infof("Task %s dispatched to worker", task.ID)
                
            default:
                // 工作器都忙，重新放回队列
                s.priorityQueue.Push(task)
                logx.Debugf("All workers busy, task %s returned to queue", task.ID)
            }
        }
    }
}
```

### 4. 工作器处理阶段

#### 工作器池架构
```go
// 调度器初始化时创建工作器池
func NewTaskScheduler(config *TaskQueueConf, judge *JudgeEngine) *TaskScheduler {
    scheduler := &TaskScheduler{
        workers:   make([]*Worker, config.MaxWorkers),  // 工作器数组
        taskQueue: make(chan *JudgeTask, config.QueueSize),  // 任务通道
    }
    
    // 创建指定数量的工作器
    for i := 0; i < config.MaxWorkers; i++ {
        scheduler.workers[i] = NewWorker(i, judge)
    }
    
    return scheduler
}
```

#### 工作器运行循环
```go
func (w *Worker) Start(wg *sync.WaitGroup) {
    defer wg.Done()
    
    for {
        select {
        case task := <-w.TaskChan:
            // 接收到任务，开始处理
            w.processTask(task)
            
        case <-w.QuitChan:
            // 收到停止信号
            return
        }
    }
}
```

#### 任务处理详细流程
```go
func (w *Worker) processTask(task *JudgeTask) {
    logx.Infof("Worker %d processing task %s", w.ID, task.ID)
    
    // 1. 更新任务状态
    task.Status = TaskStatusRunning
    task.StartedAt = &time.Now()
    
    // 2. 更新统计信息
    atomic.AddInt64(&scheduler.stats.PendingTasks, -1)
    atomic.AddInt64(&scheduler.stats.RunningTasks, 1)
    
    // 3. 调用判题引擎执行判题
    result, err := w.Judge.Judge(task.Context, &judge.JudgeRequest{
        SubmissionID: task.SubmissionID,
        ProblemID:    task.ProblemID,
        UserID:       task.UserID,
        Language:     task.Language,
        Code:         task.Code,
        TimeLimit:    task.TimeLimit,
        MemoryLimit:  task.MemoryLimit,
        TestCases:    task.TestCases,
    })
    
    // 4. 更新任务完成状态
    task.CompletedAt = &time.Now()
    
    if err != nil {
        // 判题失败
        task.Status = TaskStatusFailed
        task.Error = err.Error()
        atomic.AddInt64(&scheduler.stats.FailedTasks, 1)
        logx.Errorf("Worker %d task %s failed: %v", w.ID, task.ID, err)
        
        // 可能需要重试
        if task.RetryCount < maxRetries {
            scheduler.retryFailedTask(task)
        }
    } else {
        // 判题成功
        task.Status = TaskStatusCompleted
        task.Result = result
        atomic.AddInt64(&scheduler.stats.CompletedTasks, 1)
        logx.Infof("Worker %d task %s completed successfully", w.ID, task.ID)
    }
    
    // 5. 更新统计信息
    atomic.AddInt64(&scheduler.stats.RunningTasks, -1)
}
```

## 关键设计特点

### 1. 并发安全
- 使用 `sync.Map` 存储任务映射
- 优先级队列使用 `sync.Mutex` 保护
- 使用原子操作更新统计信息

### 2. 优先级调度
- 比赛任务 > VIP用户 > 普通用户
- 相同优先级按时间先后顺序

### 3. 资源管理
- 工作器池限制并发数量
- 任务队列缓冲区防止内存溢出
- 任务超时机制防止死锁

### 4. 监控和统计
- 实时统计任务数量
- 任务状态跟踪
- 性能指标收集

### 5. 容错机制
- 任务重试机制
- 优雅关闭
- 任务取消支持

## 性能优化策略

### 1. 内存管理
```go
// 定期清理已完成的旧任务
func (s *TaskScheduler) cleanupOldTasks() {
    cutoff := time.Now().Add(-24 * time.Hour)
    
    s.tasks.Range(func(key, value interface{}) bool {
        task := value.(*JudgeTask)
        if task.CompletedAt != nil && task.CompletedAt.Before(cutoff) {
            s.tasks.Delete(key)  // 删除24小时前的任务
        }
        return true
    })
}
```

### 2. 负载均衡
- 工作器自动从共享队列获取任务
- 避免某些工作器过载

### 3. 预估等待时间
```go
func (s *TaskScheduler) estimateWaitTime(position int) int {
    avgTaskTime := 30  // 平均任务执行时间（秒）
    workersCount := len(s.workers)
    
    return (position / workersCount) * avgTaskTime
}
```

## 扩展性考虑

### 1. 水平扩展
- 可以增加更多工作器
- 支持分布式部署（通过消息队列）

### 2. 垂直扩展
- 可以调整队列大小
- 动态调整工作器数量

### 3. 监控集成
- 暴露指标接口
- 支持实时监控dashboard

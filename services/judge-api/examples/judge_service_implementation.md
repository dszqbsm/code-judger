# 判题服务完整实现流程详解

## 🏗️ 系统架构总览

判题服务采用微服务架构，基于Go-Zero框架实现，整个系统分为以下几个核心层次：

### 架构层次划分
```
┌─────────────────┐
│   用户层        │  ← 用户提交代码
├─────────────────┤
│   API网关层     │  ← HTTP接口、路由分发
├─────────────────┤  
│   业务逻辑层    │  ← 参数验证、业务处理
├─────────────────┤
│   任务调度层    │  ← 任务队列、工作器池
├─────────────────┤
│   判题执行层    │  ← 代码编译、沙箱执行
├─────────────────┤
│   基础设施层    │  ← 数据库、缓存、监控
└─────────────────┘
```

## 📋 完整的判题流程

### 阶段1：请求接收与路由 (API Gateway Layer)

#### 1.1 HTTP请求接收
```go
// services/judge-api/internal/handler/routes.go
func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
    server.AddRoutes([]rest.Route{
        {
            Method: http.MethodPost, 
            Path: "/api/v1/judge/submit", 
            Handler: judge.SubmitJudgeHandler(serverCtx)
        },
    })
}
```

#### 1.2 请求数据结构
```go
// 用户提交的判题请求
type SubmitJudgeReq struct {
    SubmissionId int64  `json:"submission_id" validate:"required,min=1"`
    ProblemId    int64  `json:"problem_id" validate:"required,min=1"`
    UserId       int64  `json:"user_id" validate:"required,min=1"`
    Language     string `json:"language" validate:"required,oneof=cpp c java python go"`
    Code         string `json:"code" validate:"required,min=1"`
}
```

### 阶段2：业务逻辑处理 (Business Logic Layer)

#### 2.1 参数验证与安全检查
```go
// services/judge-api/internal/logic/judge/submitjudgelogic.go
func (l *SubmitJudgeLogic) SubmitJudge(req *types.SubmitJudgeReq) (*types.SubmitJudgeResp, error) {
    // 1. 基础参数验证
    if err := l.validateBasicRequest(req); err != nil {
        return errorResponse(400, err.Error()), nil
    }
    
    // 验证内容包括：
    // - 用户ID、提交ID、题目ID的有效性
    // - 代码长度限制（防止代码炸弹攻击）
    // - 编程语言支持检查
    // - 代码内容非空验证
}
```

#### 2.2 从题目服务获取权威数据
```go
// 2. 从题目服务获取题目详细信息
problemInfo, err := l.svcCtx.ProblemClient.GetProblemDetail(l.ctx, req.ProblemId)
if err != nil {
    return errorResponse(404, "获取题目信息失败"), nil
}

// 获取的题目信息包括：
// - 时间限制 (TimeLimit)
// - 内存限制 (MemoryLimit) 
// - 支持的编程语言列表
// - 完整的测试用例数据
// - 题目状态和权限信息
```

**为什么要从题目服务获取数据？**
- **单一数据源原则**：题目的时间、内存限制等属性由题目服务统一管理
- **数据一致性保证**：避免不同服务间数据不同步
- **权限控制**：题目服务可以控制题目的可见性和可访问性
- **业务解耦**：判题服务只关注执行逻辑，不管理题目元数据

#### 2.3 多层语言验证
```go
// 3. 验证编程语言支持（双重验证）
if err := l.validateLanguageSupport(req.Language, problemInfo); err != nil {
    return errorResponse(400, err.Error()), nil
}

func (l *SubmitJudgeLogic) validateLanguageSupport(language string, problemInfo *types.ProblemInfo) error {
    // 第一层：题目业务限制验证
    if !l.isLanguageSupportedByProblem(language, problemInfo.Languages) {
        return fmt.Errorf("题目不支持 %s 语言", language)
    }
    
    // 第二层：系统技术能力验证  
    if !l.svcCtx.JudgeEngine.IsLanguageSupported(language) {
        return fmt.Errorf("判题系统暂不支持 %s 语言", language)
    }
    
    return nil
}
```

**双重验证的必要性：**
- **业务层限制**：某些题目可能只允许特定语言（如算法竞赛题目）
- **技术层限制**：判题系统可能暂未安装某种语言的编译器
- **灵活性保证**：业务规则和技术能力可以独立变更

### 阶段3：任务调度 (Task Scheduling Layer)

#### 3.1 创建判题任务
```go
// 5. 转换测试用例（值类型 -> 指针类型）
testCases := make([]*types.TestCase, len(problemInfo.TestCases))
for i, tc := range problemInfo.TestCases {
    testCases[i] = &types.TestCase{
        CaseId:         tc.CaseId,
        Input:          tc.Input,
        ExpectedOutput: tc.ExpectedOutput,
        TimeLimit:      tc.TimeLimit,
        MemoryLimit:    tc.MemoryLimit,
    }
}

// 6. 创建判题任务（使用题目的权威限制）
task := &scheduler.JudgeTask{
    SubmissionID: req.SubmissionId,
    ProblemID:    req.ProblemId,
    UserID:       req.UserId,
    Language:     req.Language,
    Code:         req.Code,
    TimeLimit:    problemInfo.TimeLimit,    // 使用题目服务的权威数据
    MemoryLimit:  problemInfo.MemoryLimit,  // 使用题目服务的权威数据
    TestCases:    testCases,                // 使用题目服务的测试用例
    Priority:     l.determinePriority(req.UserId),
    Status:       scheduler.TaskStatusPending,
    CreatedAt:    time.Now(),
}
```

**数据转换的目的：**
- **内存效率**：指针类型避免大数据结构的深拷贝
- **状态修改**：判题过程中可能需要修改测试用例的状态
- **接口统一**：判题引擎统一使用指针类型接口

#### 3.2 任务优先级确定
```go
func (l *SubmitJudgeLogic) determinePriority(userID int64) int {
    // 根据用户类型确定任务优先级
    userInfo := l.getUserInfo(userID)
    
    switch {
    case userInfo.IsInContest():
        return scheduler.PriorityHigh    // 比赛中的提交 = 高优先级
    case userInfo.IsVIP():
        return scheduler.PriorityNormal  // VIP用户 = 普通优先级  
    default:
        return scheduler.PriorityLow     // 普通用户 = 低优先级
    }
}
```

#### 3.3 提交到任务调度器
```go
// 7. 提交任务到调度器
if err := l.svcCtx.TaskScheduler.SubmitTask(task); err != nil {
    return errorResponse(500, "任务提交失败"), nil
}

// TaskScheduler.SubmitTask 实现
func (s *TaskScheduler) SubmitTask(task *JudgeTask) error {
    // 生成唯一任务ID
    task.ID = generateTaskID(task.SubmissionID)
    
    // 创建可取消的上下文
    task.Context, task.CancelFunc = context.WithCancel(s.ctx)
    
    // 存储到任务映射表（用于查询和取消）
    s.tasks.Store(task.ID, task)
    
    // 推入优先级队列
    s.priorityQueue.Push(task)
    
    // 统计信息更新
    atomic.AddInt64(&s.stats.TotalSubmitted, 1)
    
    return nil
}
```

### 阶段4：任务调度与分发 (Task Dispatch)

#### 4.1 优先级队列管理
```go
// services/judge-api/internal/scheduler/scheduler.go
type TaskScheduler struct {
    priorityQueue   *PriorityQueue      // 优先级队列
    taskQueue       chan *JudgeTask     // 工作器任务通道
    workers         []*Worker           // 工作器池
    tasks           sync.Map            // 任务映射表
    // ...
}

// 调度器核心分发逻辑
func (s *TaskScheduler) dispatch() {
    ticker := time.NewTicker(100 * time.Millisecond)  // 100ms检查一次
    defer ticker.Stop()
    
    for {
        select {
        case <-s.ctx.Done():
            return
        case <-ticker.C:
            // 从优先级队列取出最高优先级任务
            task := s.priorityQueue.Pop()
            if task == nil {
                continue  // 队列为空，继续等待
            }
            
            // 检查任务是否被取消
            if task.Status == TaskStatusCancelled {
                continue
            }
            
            // 尝试分发给空闲工作器
            select {
            case s.taskQueue <- task:
                // 成功分发给工作器
                logx.Infof("Task %s dispatched to worker", task.ID)
            default:
                // 所有工作器都忙，重新放回队列
                s.priorityQueue.Push(task)
            }
        }
    }
}
```

#### 4.2 工作器池管理
```go
// 工作器启动
func (s *TaskScheduler) Start() error {
    // 启动指定数量的工作器
    for i := 0; i < s.config.MaxWorkers; i++ {
        worker := NewWorker(i, s.judgeEngine)
        s.workers = append(s.workers, worker)
        s.wg.Add(1)
        go worker.Start(&s.wg)  // 每个工作器在独立协程中运行
    }
    
    // 启动任务分发器
    go s.dispatch()
    
    return nil
}
```

### 阶段5：判题执行 (Judge Execution Layer)

#### 5.1 工作器接收任务
```go
// services/judge-api/internal/scheduler/worker.go
func (w *Worker) Start(wg *sync.WaitGroup) {
    defer wg.Done()
    
    for {
        select {
        case task := <-w.TaskChan:
            w.processTask(task)  // 处理判题任务
        case <-w.QuitChan:
            return  // 工作器停止
        }
    }
}
```

#### 5.2 任务执行流程
```go
func (w *Worker) processTask(task *JudgeTask) {
    logx.Infof("Worker %d processing task %s", w.ID, task.ID)
    
    // 1. 更新任务状态
    task.Status = TaskStatusRunning
    now := time.Now()
    task.StartedAt = &now
    
    // 2. 构造判题请求
    judgeReq := &judge.JudgeRequest{
        SubmissionID: task.SubmissionID,
        ProblemID:    task.ProblemID,
        UserID:       task.UserID,
        Language:     task.Language,
        Code:         task.Code,
        TimeLimit:    task.TimeLimit,
        MemoryLimit:  task.MemoryLimit,
        TestCases:    task.TestCases,
    }
    
    // 3. 调用判题引擎执行
    result, err := w.Judge.Judge(task.Context, judgeReq)
    
    // 4. 处理执行结果
    if err != nil {
        task.Status = TaskStatusFailed
        task.Error = err.Error()
        logx.Errorf("Judge failed for task %s: %v", task.ID, err)
    } else {
        task.Status = TaskStatusCompleted
        task.Result = result
        logx.Infof("Judge completed for task %s: status=%s", task.ID, result.Status)
    }
    
    // 5. 更新完成时间
    completedAt := time.Now()
    task.CompletedAt = &completedAt
}
```

### 阶段6：代码编译与执行 (Code Compilation & Execution)

#### 6.1 判题引擎处理流程
```go
// services/judge-api/internal/judge/judge.go
func (je *JudgeEngine) Judge(ctx context.Context, req *JudgeRequest) (*types.JudgeResult, error) {
    // 1. 验证请求参数（安全检查）
    if err := je.validateRequest(req); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    // 2. 获取语言执行器
    executor, err := je.languageManager.GetExecutor(req.Language)
    if err != nil {
        return nil, fmt.Errorf("unsupported language: %w", err)
    }
    
    // 3. 创建隔离的临时工作目录
    tempDir, err := je.createTempDir(req.SubmissionID)
    if err != nil {
        return nil, fmt.Errorf("failed to create temp dir: %w", err)
    }
    defer je.cleanupTempDir(tempDir)  // 确保资源清理
    
    // 4. 初始化判题结果
    result := &types.JudgeResult{
        SubmissionId: req.SubmissionID,
        Status:       "judging",
        TestCases:    make([]types.TestCaseResult, 0, len(req.TestCases)),
        JudgeInfo: types.JudgeInfo{
            JudgeServer:     je.getServerID(),
            JudgeTime:       time.Now().Format(time.RFC3339),
            LanguageVersion: executor.GetVersion(),
        },
    }
    
    // 5. 代码编译阶段
    compileResult, err := je.compileCode(ctx, executor, req.Code, tempDir)
    if err != nil {
        return nil, fmt.Errorf("compilation failed: %w", err)
    }
    
    result.CompileInfo = types.CompileInfo{
        Success: compileResult.Success,
        Message: compileResult.Message,
        Time:    int(compileResult.CompileTime.Milliseconds()),
    }
    
    // 编译失败直接返回
    if !compileResult.Success {
        result.Status = "compile_error"
        return result, nil
    }
    
    // 6. 测试用例执行循环
    totalScore := 0
    maxTimeUsed := 0
    maxMemoryUsed := 0
    
    for i, testCase := range req.TestCases {
        testResult, err := je.runTestCase(ctx, executor, compileResult.ExecutablePath,
            testCase, tempDir, req.TimeLimit, req.MemoryLimit)
        
        if err != nil {
            // 系统错误处理
            testResult = &types.TestCaseResult{
                CaseId:      testCase.CaseId,
                Status:      "system_error",
                ErrorOutput: err.Error(),
            }
        }
        
        result.TestCases = append(result.TestCases, *testResult)
        
        // 更新资源使用统计
        if testResult.TimeUsed > maxTimeUsed {
            maxTimeUsed = testResult.TimeUsed
        }
        if testResult.MemoryUsed > maxMemoryUsed {
            maxMemoryUsed = testResult.MemoryUsed
        }
        
        // 计算分数（平均分配）
        if testResult.Status == "accepted" {
            totalScore += 100 / len(req.TestCases)
        }
        
        // 可选：遇到错误立即停止（根据配置）
        if testResult.Status != "accepted" {
            break
        }
    }
    
    // 7. 设置最终结果
    result.Score = totalScore
    result.TimeUsed = maxTimeUsed
    result.MemoryUsed = maxMemoryUsed
    result.Status = je.determineFinalStatus(result.TestCases)
    
    return result, nil
}
```

#### 6.2 单个测试用例执行详情
```go
func (je *JudgeEngine) runTestCase(ctx context.Context, executor languages.LanguageExecutor,
    executablePath string, testCase *types.TestCase, workDir string,
    timeLimit int, memoryLimit int) (*types.TestCaseResult, error) {
    
    // 1. 准备测试文件
    inputFile := filepath.Join(workDir, fmt.Sprintf("input_%d.txt", testCase.CaseId))
    outputFile := filepath.Join(workDir, fmt.Sprintf("output_%d.txt", testCase.CaseId))
    errorFile := filepath.Join(workDir, fmt.Sprintf("error_%d.txt", testCase.CaseId))
    
    // 2. 写入测试输入
    if err := os.WriteFile(inputFile, []byte(testCase.Input), 0644); err != nil {
        return nil, fmt.Errorf("failed to write input file: %w", err)
    }
    
    // 3. 应用语言特定的资源调整
    adjustedTimeLimit := int64(float64(timeLimit) * executor.GetTimeMultiplier())
    adjustedMemoryLimit := int64(float64(memoryLimit) * executor.GetMemoryMultiplier() * 1024)
    
    // 4. 配置执行参数
    execConfig := &languages.ExecutionConfig{
        TimeLimit:   adjustedTimeLimit,
        MemoryLimit: adjustedMemoryLimit,
        InputFile:   inputFile,
        OutputFile:  outputFile,
        ErrorFile:   errorFile,
        Environment: []string{"PATH=/usr/bin:/bin"},
    }
    
    // 5. 在安全沙箱中执行
    execResult, err := executor.Execute(ctx, executablePath, workDir, execConfig)
    if err != nil {
        return nil, fmt.Errorf("execution failed: %w", err)
    }
    
    // 6. 读取程序输出
    var output, errorOutput string
    if outputData, err := os.ReadFile(outputFile); err == nil {
        output = string(outputData)
    }
    if errorData, err := os.ReadFile(errorFile); err == nil {
        errorOutput = string(errorData)
    }
    
    // 7. 创建测试结果
    result := &types.TestCaseResult{
        CaseId:      testCase.CaseId,
        TimeUsed:    int(execResult.TimeUsed),
        MemoryUsed:  int(execResult.MemoryUsed),
        Input:       testCase.Input,
        Output:      strings.TrimSpace(output),
        Expected:    strings.TrimSpace(testCase.ExpectedOutput),
        ErrorOutput: errorOutput,
    }
    
    // 8. 确定执行状态
    result.Status = je.determineTestCaseStatus(execResult, result)
    
    return result, nil
}
```

### 阶段7：安全沙箱执行 (Sandbox Execution)

#### 7.1 沙箱安全机制
```go
// services/judge-api/internal/sandbox/sandbox.go
type SandboxConfig struct {
    // 用户权限隔离
    UID     int    // 运行用户ID (通常是nobody: 65534)
    GID     int    // 运行组ID
    Chroot  string // chroot根目录隔离
    WorkDir string // 工作目录限制
    
    // 资源限制
    TimeLimit     int64 // CPU时间限制(毫秒)
    WallTimeLimit int64 // 墙钟时间限制(毫秒)
    MemoryLimit   int64 // 内存限制(KB)
    StackLimit    int64 // 栈大小限制(KB)
    FileSizeLimit int64 // 文件大小限制(KB)
    ProcessLimit  int   // 进程数限制
    
    // 系统调用控制
    AllowedSyscalls []int // 允许的系统调用白名单
    EnableSeccomp   bool  // 启用seccomp过滤
    
    // 输入输出重定向
    InputFile  string // 标准输入重定向
    OutputFile string // 标准输出重定向
    ErrorFile  string // 错误输出重定向
}
```

#### 7.2 执行监控与资源控制
```go
func (sb *Sandbox) Execute(ctx context.Context, config *SandboxConfig) (*ExecuteResult, error) {
    // 1. 创建子进程
    cmd := exec.CommandContext(ctx, config.ExecutablePath)
    
    // 2. 设置工作目录和环境
    cmd.Dir = config.WorkDir
    cmd.Env = config.Environment
    
    // 3. 重定向输入输出
    cmd.Stdin = openFile(config.InputFile)
    cmd.Stdout = createFile(config.OutputFile)
    cmd.Stderr = createFile(config.ErrorFile)
    
    // 4. 应用系统级资源限制
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,  // 创建新进程组
        // 设置用户和组ID
        Credential: &syscall.Credential{
            Uid: uint32(config.UID),
            Gid: uint32(config.GID),
        },
        // 设置资源限制
        Rlimits: []syscall.Rlimit{
            {Cur: uint64(config.TimeLimit), Max: uint64(config.TimeLimit)},     // CPU时间
            {Cur: uint64(config.MemoryLimit), Max: uint64(config.MemoryLimit)}, // 内存
            {Cur: uint64(config.FileSizeLimit), Max: uint64(config.FileSizeLimit)}, // 文件大小
        },
    }
    
    // 5. 启动进程并监控
    startTime := time.Now()
    
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("failed to start process: %w", err)
    }
    
    // 6. 等待执行完成或超时
    done := make(chan error, 1)
    go func() {
        done <- cmd.Wait()
    }()
    
    select {
    case err := <-done:
        // 正常完成或异常退出
        endTime := time.Now()
        
        return &ExecuteResult{
            Status:     sb.determineStatus(cmd.ProcessState),
            TimeUsed:   endTime.Sub(startTime).Milliseconds(),
            MemoryUsed: sb.getMemoryUsage(cmd.Process.Pid),
            ExitCode:   cmd.ProcessState.ExitCode(),
        }, err
        
    case <-time.After(time.Duration(config.WallTimeLimit) * time.Millisecond):
        // 墙钟时间超时，强制杀死进程
        cmd.Process.Kill()
        return &ExecuteResult{
            Status: StatusTimeLimitExceeded,
        }, nil
    }
}
```

## 🔄 数据流转与状态管理

### 任务状态生命周期
```
pending → running → completed/failed/cancelled
   ↓         ↓           ↓
[等待中] → [执行中] → [已完成/失败/取消]
```

### 数据转换路径
```
HTTP请求 → SubmitJudgeReq → JudgeTask → JudgeRequest → JudgeResult → SubmitJudgeResp
```

### 内存管理策略
- **临时文件自动清理**：每个判题任务完成后自动删除临时目录
- **任务状态缓存**：已完成的任务结果缓存在Redis中
- **工作器池复用**：避免频繁创建销毁线程的开销

## 🔒 安全防护体系

### 多层安全防护
1. **API层安全**：参数验证、请求频率限制
2. **业务层安全**：代码模式检查、资源限制验证
3. **执行层安全**：沙箱隔离、系统调用过滤
4. **系统层安全**：用户权限隔离、资源配额限制

### 恶意代码防护
```go
// 禁止的代码模式示例
ForbiddenPatterns: [
    "system\\s*\\(",          // 防止系统命令执行
    "exec\\s*\\(",            // 防止执行外部程序
    "fork\\s*\\(",            // 防止进程分叉攻击
    "while\\s*\\(\\s*1\\s*\\)", // 防止无限循环
    "__import__\\s*\\(",      // Python危险导入
    "eval\\s*\\(",            // 防止动态代码执行
]
```

## 📊 性能优化策略

### 1. 并发处理优化
- **工作器池**：预创建固定数量的工作器，避免频繁创建线程
- **优先级队列**：重要任务优先处理，提升用户体验
- **异步处理**：判题结果异步返回，支持高并发提交

### 2. 资源利用优化
- **编译缓存**：相同代码的编译结果可以复用
- **语言特定优化**：不同语言采用不同的资源倍数
- **内存池化**：减少内存分配和GC压力

### 3. 系统监控与调优
- **实时监控**：任务队列长度、工作器利用率、系统资源使用
- **性能指标**：平均判题时间、吞吐量、错误率统计
- **自动扩缩容**：根据负载动态调整工作器数量

这个判题服务实现了完整的从代码提交到结果返回的全流程，具有高性能、高安全性和良好的可扩展性，能够支撑大规模的在线判题需求。


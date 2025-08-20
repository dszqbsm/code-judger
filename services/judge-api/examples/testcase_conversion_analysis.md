# 测试用例转换分析

## 数据类型差异

### 题目服务返回的数据结构
```go
type ProblemInfo struct {
    TestCases []TestCase  // 值类型切片
}

type TestCase struct {
    CaseId         int    `json:"case_id"`
    Input          string `json:"input"`
    ExpectedOutput string `json:"expected_output"`
    TimeLimit      int    `json:"time_limit,omitempty"`
    MemoryLimit    int    `json:"memory_limit,omitempty"`
}
```

### 调度器需要的数据结构
```go
type JudgeTask struct {
    TestCases []*types.TestCase  // 指针类型切片
}
```

## 转换的必要性

### 1. **内存管理考虑**

```go
// 值类型：每次赋值都会复制整个结构体
testCases := problemInfo.TestCases  // 复制所有测试用例数据

// 指针类型：只复制指针，共享底层数据
testCases := []*TestCase{&tc1, &tc2}  // 只复制指针
```

**内存对比：**
- 100个测试用例，每个1KB → 值类型需要100KB
- 100个测试用例，每个1KB → 指针类型只需要800字节（64位系统8字节/指针）

### 2. **性能优化**

```go
// 场景：判题过程中需要更新测试用例状态
for i, testCase := range task.TestCases {
    // 如果是值类型，无法直接修改
    // testCase.Status = "running"  // 这不会影响原数据
    
    // 如果是指针类型，可以直接修改
    testCase.Status = "running"     // 直接修改原数据
    testCase.ExecutionTime = 150    // 记录执行时间
}
```

### 3. **并发安全**

```go
// 多个goroutine可能同时处理同一个任务的不同测试用例
func processTestCase(tc *TestCase, result chan<- TestCaseResult) {
    // 原地更新测试用例状态
    tc.Status = "running"
    
    // 执行测试
    output := runTest(tc.Input)
    
    // 更新结果
    tc.ActualOutput = output
    tc.Status = "completed"
    
    result <- TestCaseResult{...}
}
```

## 实际的业务场景

### 场景1：实时状态更新
```go
// 判题过程中需要实时更新每个测试用例的状态
task := &JudgeTask{
    TestCases: testCases,  // 指针切片
}

// 工作器可以直接更新测试用例状态
for _, tc := range task.TestCases {
    tc.Status = "pending"      // 初始状态
}

// 执行时更新状态
tc.Status = "running"          // 执行中
tc.StartTime = time.Now()      // 开始时间

// 完成时更新结果
tc.Status = "completed"        // 已完成
tc.EndTime = time.Now()        // 结束时间
tc.ActualOutput = output       // 实际输出
```

### 场景2：内存优化
```go
// 大量测试用例的题目（如压力测试题）
problemWithManyTestCases := ProblemInfo{
    TestCases: make([]TestCase, 1000),  // 1000个测试用例
}

// 如果直接复制值，内存消耗巨大
// task.TestCases = problemInfo.TestCases  // 复制1000个结构体

// 使用指针转换，节省内存
testCases := make([]*types.TestCase, len(problemInfo.TestCases))
for i, tc := range problemInfo.TestCases {
    testCases[i] = &types.TestCase{...}  // 只存储指针
}
```

### 场景3：判题结果收集
```go
// 判题引擎需要收集每个测试用例的详细结果
func collectResults(task *JudgeTask) *JudgeResult {
    results := make([]TestCaseResult, len(task.TestCases))
    
    for i, tc := range task.TestCases {
        results[i] = TestCaseResult{
            CaseId:     tc.CaseId,
            Status:     tc.Status,        // 从指针直接读取最新状态
            TimeUsed:   tc.ExecutionTime, // 从指针读取执行时间
            MemoryUsed: tc.MemoryUsed,    // 从指针读取内存使用
            Output:     tc.ActualOutput,  // 从指针读取实际输出
        }
    }
    
    return &JudgeResult{TestCases: results}
}
```

## 设计模式分析

这种转换体现了几个重要的设计模式：

### 1. **适配器模式 (Adapter Pattern)**
```go
// 将题目服务的数据格式适配为调度器需要的格式
func adaptTestCases(problemTestCases []TestCase) []*TestCase {
    adapted := make([]*TestCase, len(problemTestCases))
    for i, tc := range problemTestCases {
        adapted[i] = &TestCase{
            CaseId:         tc.CaseId,
            Input:          tc.Input,
            ExpectedOutput: tc.ExpectedOutput,
            TimeLimit:      tc.TimeLimit,
            MemoryLimit:    tc.MemoryLimit,
        }
    }
    return adapted
}
```

### 2. **数据传输对象模式 (DTO Pattern)**
```go
// 题目服务的DTO（值类型，适合网络传输）
type ProblemDTO struct {
    TestCases []TestCase  // 值类型，JSON序列化友好
}

// 判题引擎的业务对象（指针类型，适合内存操作）
type JudgeTask struct {
    TestCases []*TestCase  // 指针类型，内存操作友好
}
```

## 优化建议

### 1. **深拷贝 vs 浅拷贝**
```go
// 当前实现：深拷贝（安全但消耗内存）
testCases[i] = &types.TestCase{
    CaseId:         tc.CaseId,         // 复制值
    Input:          tc.Input,          // 复制字符串
    ExpectedOutput: tc.ExpectedOutput, // 复制字符串
    TimeLimit:      tc.TimeLimit,      // 复制值
    MemoryLimit:    tc.MemoryLimit,    // 复制值
}

// 优化方案：浅拷贝（节省内存但需要注意并发安全）
testCases[i] = &tc  // 直接使用指针（需要确保tc不会被修改）
```

### 2. **对象池模式**
```go
// 使用对象池减少GC压力
var testCasePool = sync.Pool{
    New: func() interface{} {
        return &TestCase{}
    },
}

func convertTestCases(problemTestCases []TestCase) []*TestCase {
    testCases := make([]*TestCase, len(problemTestCases))
    for i, tc := range problemTestCases {
        pooled := testCasePool.Get().(*TestCase)
        *pooled = tc  // 复制内容到池化对象
        testCases[i] = pooled
    }
    return testCases
}
```

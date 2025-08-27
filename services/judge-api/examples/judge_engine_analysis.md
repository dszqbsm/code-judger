# 判题引擎详细功能分析

## 整体架构概览

```
[调度器] → [判题引擎] → [语言管理器] → [沙箱执行器]
                    ↓
              [编译] → [执行] → [评测]
```

判题引擎采用分层设计，实现了完整的代码执行和评测流程。

## 核心组件分析

### 1. 判题引擎主体 (JudgeEngine)

#### 结构定义
```go
type JudgeEngine struct {
    config          *config.JudgeEngineConf     // 配置信息
    languageManager *languages.LanguageManager  // 语言管理器
    workDir         string                      // 工作目录
    tempDir         string                      // 临时目录
}
```

#### 核心功能模块
1. **请求验证**：参数合法性检查、安全检查
2. **代码编译**：多语言编译支持
3. **沙箱执行**：安全的代码执行环境
4. **结果评测**：输出比较和状态判定
5. **资源监控**：时间、内存、进程数控制

## 详细的判题流程

### 阶段1：请求接收和验证

```go
func (je *JudgeEngine) Judge(ctx context.Context, req *JudgeRequest) (*types.JudgeResult, error) {
    // 1. 验证请求参数
    if err := je.validateRequest(req); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
```

**验证内容包括：**
- 提交ID有效性检查
- 代码长度限制检查（防止超大代码攻击）
- 时间和内存限制范围验证
- 测试用例数据完整性
- 禁止的代码模式检查（安全防护）

**安全检查实现：**
```go
// 检查禁止的代码模式
for _, pattern := range je.config.Security.ForbiddenPatterns {
    if strings.Contains(req.Code, pattern) {
        return fmt.Errorf("code contains forbidden pattern: %s", pattern)
    }
}
```

### 阶段2：环境准备

```go
// 获取语言执行器
executor, err := je.languageManager.GetExecutor(req.Language)

// 创建临时工作目录
tempDir, err := je.createTempDir(req.SubmissionID)
defer je.cleanupTempDir(tempDir)
```

**临时目录结构：**
```
/tmp/judge/judge_{submission_id}_{timestamp}/
├── main.cpp          # 源代码文件
├── main.exe          # 编译后的可执行文件
├── input_1.txt       # 测试用例输入
├── output_1.txt      # 程序输出
├── error_1.txt       # 错误输出
└── compile.log       # 编译日志
```

### 阶段3：代码编译

#### 编译型语言处理
```go
func (je *JudgeEngine) compileCode(ctx context.Context, executor languages.LanguageExecutor,
    code string, workDir string) (*languages.CompileResult, error) {
    
    if !executor.IsCompiled() {
        // 解释型语言，直接保存源文件
        sourceFile := filepath.Join(workDir, "main"+executor.GetFileExtension())
        return &languages.CompileResult{
            Success:        true,
            ExecutablePath: sourceFile,
            CompileTime:    0,
            Message:        "No compilation required",
        }, nil
    }
    
    // 编译型语言，调用编译器
    return executor.Compile(ctx, code, workDir)
}
```

#### 不同语言的编译策略
- **C/C++**: 使用gcc/g++编译，生成可执行文件
- **Java**: 使用javac编译.class文件
- **Python/JavaScript**: 解释型语言，无需编译
- **Go**: 使用go build编译

**编译结果处理：**
```go
result.CompileInfo = types.CompileInfo{
    Success: compileResult.Success,
    Message: compileResult.Message,
    Time:    int(compileResult.CompileTime.Milliseconds()),
}

if !compileResult.Success {
    result.Status = "compile_error"
    return result, nil  // 编译失败，直接返回
}
```

### 阶段4：测试用例执行

#### 核心执行循环
```go
for i, testCase := range req.TestCases {
    testResult, err := je.runTestCase(ctx, executor, compileResult.ExecutablePath,
        testCase, tempDir, req.TimeLimit, req.MemoryLimit)
    
    result.TestCases = append(result.TestCases, *testResult)
    
    // 计算分数
    if testResult.Status == "accepted" {
        totalScore += 100 / len(req.TestCases)
    }
    
    // 可选：遇到错误立即停止
    if testResult.Status != "accepted" {
        break
    }
}
```

#### 单个测试用例执行流程
```go
func (je *JudgeEngine) runTestCase(ctx context.Context, executor languages.LanguageExecutor,
    executablePath string, testCase *types.TestCase, workDir string,
    timeLimit int, memoryLimit int) (*types.TestCaseResult, error) {
    
    // 1. 准备输入输出文件
    inputFile := filepath.Join(workDir, fmt.Sprintf("input_%d.txt", testCase.CaseId))
    outputFile := filepath.Join(workDir, fmt.Sprintf("output_%d.txt", testCase.CaseId))
    errorFile := filepath.Join(workDir, fmt.Sprintf("error_%d.txt", testCase.CaseId))
    
    // 2. 写入测试输入
    os.WriteFile(inputFile, []byte(testCase.Input), 0644)
    
    // 3. 应用语言特定的资源限制倍数
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
    
    // 5. 在沙箱中执行程序
    execResult, err := executor.Execute(ctx, executablePath, workDir, execConfig)
    
    // 6. 读取程序输出
    output, _ := os.ReadFile(outputFile)
    errorOutput, _ := os.ReadFile(errorFile)
    
    // 7. 创建测试结果
    result := &types.TestCaseResult{
        CaseId:      testCase.CaseId,
        TimeUsed:    int(execResult.TimeUsed),
        MemoryUsed:  int(execResult.MemoryUsed),
        Input:       testCase.Input,
        Output:      strings.TrimSpace(string(output)),
        Expected:    strings.TrimSpace(testCase.ExpectedOutput),
        ErrorOutput: string(errorOutput),
    }
    
    // 8. 确定执行状态
    result.Status = je.determineTestCaseStatus(execResult, result)
    
    return result, nil
}
```

### 阶段5：结果评测

#### 状态判定逻辑
```go
func (je *JudgeEngine) determineTestCaseStatus(execResult *sandbox.ExecuteResult,
    testResult *types.TestCaseResult) string {
    
    switch execResult.Status {
    case sandbox.StatusAccepted:
        // 检查输出是否正确
        if je.compareOutput(testResult.Output, testResult.Expected) {
            return "accepted"
        }
        return "wrong_answer"
        
    case sandbox.StatusTimeLimitExceeded:
        return "time_limit_exceeded"
        
    case sandbox.StatusMemoryLimitExceeded:
        return "memory_limit_exceeded"
        
    case sandbox.StatusOutputLimitExceeded:
        return "output_limit_exceeded"
        
    case sandbox.StatusRuntimeError:
        return "runtime_error"
        
    default:
        return "system_error"
    }
}
```

#### 输出比较算法
```go
func (je *JudgeEngine) compareOutput(actual, expected string) bool {
    // 标准化输出（去除前后空白，统一换行符）
    actual = strings.TrimSpace(strings.ReplaceAll(actual, "\r\n", "\n"))
    expected = strings.TrimSpace(strings.ReplaceAll(expected, "\r\n", "\n"))
    
    // 精确匹配
    if actual == expected {
        return true
    }
    
    // TODO: 支持更多比较模式
    // 1. 忽略行末空格
    // 2. 忽略多余空行
    // 3. 浮点数误差比较
    // 4. Special Judge支持
    
    return false
}
```

#### 最终状态确定
```go
func (je *JudgeEngine) determineFinalStatus(testCases []types.TestCaseResult) string {
    acceptedCount := 0
    hasRuntimeError := false
    hasTimeLimitExceeded := false
    hasMemoryLimitExceeded := false
    hasWrongAnswer := false
    
    for _, testCase := range testCases {
        switch testCase.Status {
        case "accepted":
            acceptedCount++
        case "wrong_answer":
            hasWrongAnswer = true
        // ... 其他状态统计
        }
    }
    
    // 全部通过
    if acceptedCount == len(testCases) {
        return "accepted"
    }
    
    // 优先级：运行时错误 > 时间超限 > 内存超限 > 答案错误
    if hasRuntimeError {
        return "runtime_error"
    }
    // ... 其他状态判定
    
    return "system_error"
}
```

## 语言管理器 (LanguageManager)

### 多语言支持架构

每种编程语言都实现了 `LanguageExecutor` 接口：

```go
type LanguageExecutor interface {
    GetName() string                    // 语言名称
    GetDisplayName() string             // 显示名称
    GetVersion() string                 // 版本信息
    GetFileExtension() string           // 文件扩展名
    
    Compile(ctx, code, workDir) (*CompileResult, error)     // 编译代码
    Execute(ctx, execPath, workDir, config) (*ExecuteResult, error) // 执行代码
    
    IsCompiled() bool                   // 是否需要编译
    GetTimeMultiplier() float64         // 时间限制倍数
    GetMemoryMultiplier() float64       // 内存限制倍数
    GetMaxProcesses() int               // 最大进程数
    GetAllowedSyscalls() []int          // 允许的系统调用
}
```

### 不同语言的特殊处理

#### C/C++ 语言执行器
```yaml
cpp:
  Name: "C++"
  Version: "g++ 9.4.0"
  FileExtension: ".cpp"
  CompileCommand: "g++ -o {executable} {source} -std=c++17 -O2 -Wall -Wextra"
  ExecuteCommand: "{executable}"
  CompileTimeout: 10000
  TimeMultiplier: 1.0      # C++执行效率高，不需要额外时间
  MemoryMultiplier: 1.0    # 内存使用相对较少
  MaxProcesses: 1          # 单进程执行
```

#### Java 语言执行器
```yaml
java:
  Name: "Java"
  Version: "OpenJDK 11.0.16"
  FileExtension: ".java"
  CompileCommand: "javac -cp . -d . {source}"
  ExecuteCommand: "java -cp . -Xmx{memory_limit}m -Xss8m Main"
  CompileTimeout: 15000
  TimeMultiplier: 2.0      # JVM启动开销，需要更多时间
  MemoryMultiplier: 2.0    # JVM内存开销较大
  MaxProcesses: 64         # JVM需要多个线程
```

#### Python 语言执行器
```yaml
python:
  Name: "Python"
  Version: "Python 3.8.10"
  FileExtension: ".py"
  CompileCommand: ""       # 解释型语言，无需编译
  ExecuteCommand: "python3 {source}"
  TimeMultiplier: 3.0      # 解释执行，速度较慢
  MemoryMultiplier: 1.5    # 解释器内存开销
  MaxProcesses: 1
```

## 沙箱安全机制

### 资源隔离
```go
type SandboxConfig struct {
    UID     int    // 运行用户ID (nobody)
    GID     int    // 运行组ID
    Chroot  string // chroot根目录隔离
    WorkDir string // 工作目录
    
    // 资源限制
    TimeLimit     int64 // CPU时间限制(毫秒)
    WallTimeLimit int64 // 墙钟时间限制(毫秒)
    MemoryLimit   int64 // 内存限制(KB)
    StackLimit    int64 // 栈大小限制(KB)
    FileSizeLimit int64 // 文件大小限制(KB)
    ProcessLimit  int   // 进程数限制
    
    // 系统调用控制
    AllowedSyscalls []int // 允许的系统调用号
    EnableSeccomp   bool  // 启用seccomp过滤
}
```

### 安全防护机制

1. **用户权限隔离**：以nobody用户身份运行
2. **文件系统隔离**：chroot限制文件访问范围
3. **资源限制**：CPU、内存、进程数、文件大小限制
4. **系统调用过滤**：只允许安全的系统调用
5. **网络隔离**：禁止网络访问

### 执行监控
```go
type ExecuteResult struct {
    Status       int    // 执行状态
    TimeUsed     int64  // 实际使用时间(毫秒)
    MemoryUsed   int64  // 实际使用内存(KB)
    ExitCode     int    // 程序退出码
    Signal       int    // 接收到的信号
    ErrorMessage string // 错误信息
}
```

## 性能优化策略

### 1. 编译缓存
- 相同代码的编译结果可以缓存
- 减少重复编译开销

### 2. 沙箱复用
- 预创建沙箱环境
- 避免重复的环境初始化

### 3. 并行执行
- 多个测试用例可以并行执行
- 充分利用多核CPU

### 4. 资源池化
- 文件句柄池化
- 内存池化管理

## 扩展功能

### 1. Special Judge 支持
- 自定义比较器
- 支持多种答案的题目

### 2. 交互式判题
- 支持与程序交互的题目
- 实时输入输出交换

### 3. 多文件程序
- 支持包含多个源文件的项目
- 复杂项目结构支持

### 4. 实时反馈
- 执行进度实时更新
- 中间结果即时返回

## 错误处理和容错

### 1. 编译错误处理
```go
if !compileResult.Success {
    result.Status = "compile_error"
    result.CompileInfo.Message = compileResult.Message
    return result, nil
}
```

### 2. 运行时错误分类
- 时间超限 (Time Limit Exceeded)
- 内存超限 (Memory Limit Exceeded)
- 运行时错误 (Runtime Error)
- 输出超限 (Output Limit Exceeded)
- 系统错误 (System Error)

### 3. 异常恢复
- 沙箱崩溃自动恢复
- 临时文件自动清理
- 资源泄露防护

这个判题引擎实现了完整的在线判题功能，具有高安全性、高性能和良好的扩展性，能够支持多种编程语言的代码执行和评测。


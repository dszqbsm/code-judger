# setrlimit + cgroups 双层资源控制实现总结

## 概述

本文档总结了判题服务中setrlimit+cgroups双层资源控制方案的完整实现，通过两种技术的优势互补，实现了最佳的资源限制效果。

## 实现架构

### 双层控制架构图

```
用户代码进程
      ↓
┌─────────────────────────────────────┐
│        setrlimit 基础防护层          │
│  ┌─────────────────────────────────┐ │
│  │ - 栈大小限制 (RLIMIT_STACK)     │ │
│  │ - 文件大小限制 (RLIMIT_FSIZE)   │ │
│  │ - 虚拟内存兜底 (RLIMIT_AS)      │ │
│  │ - CPU时间兜底 (RLIMIT_CPU)      │ │
│  │ - 核心转储禁用 (RLIMIT_CORE)    │ │
│  └─────────────────────────────────┘ │
└─────────────────────────────────────┘
      ↓
┌─────────────────────────────────────┐
│       cgroups 精确控制层            │
│  ┌─────────────────────────────────┐ │
│  │ - 物理内存限制 (memory)         │ │
│  │ - CPU配额控制 (cpu)             │ │
│  │ - 进程数限制 (pids)             │ │
│  │ - I/O带宽控制 (blkio)           │ │
│  │ - CPU核心绑定 (cpuset)          │ │
│  └─────────────────────────────────┘ │
└─────────────────────────────────────┘
      ↓
┌─────────────────────────────────────┐
│       墙上时钟监控兜底              │
│     (父进程定时器监控)              │
└─────────────────────────────────────┘
```

## 核心实现组件

### 1. CgroupManager - cgroup管理器

```go
type CgroupManager struct {
    config     *CgroupConfig
    groupPaths map[string]string // 各子系统的组路径
    created    bool              // 是否已创建
}
```

**主要功能**：
- 动态创建和删除cgroup控制组
- 支持memory、cpu、cpuset、pids、blkio五种子系统
- 提供详细的资源使用统计
- 自动进程绑定和清理

**核心方法**：
- `Create()`: 创建cgroup控制组并应用资源限制
- `AddProcess(pid)`: 将进程添加到cgroup
- `GetStats()`: 获取详细资源使用统计
- `Cleanup()`: 清理cgroup资源

### 2. 双层资源控制集成

```go
func (s *SystemCallSandbox) setupResourceLimits() error {
    // 1. 优先设置cgroups（如果启用且可用）
    if s.config.EnableCgroups && s.cgroupManager != nil {
        if err := s.setupCgroups(); err != nil {
            // 根据模式决定是否降级
            if s.config.CgroupsMode == "primary" {
                return fmt.Errorf("cgroups setup failed in primary mode: %w", err)
            }
            logx.Errorf("cgroups setup failed, falling back to setrlimit: %v", err)
        }
    }
    
    // 2. 设置setrlimit（作为基础保护或降级方案）
    if err := s.setupSetrlimit(); err != nil {
        // 错误处理逻辑
    }
    
    return nil
}
```

### 3. 智能资源监控

```go
func (s *SystemCallSandbox) monitorProcessWithCgroups(pid int, startTime time.Time) (*ExecuteResult, error) {
    // 集成三种监控机制：
    // 1. ptrace系统调用跟踪
    // 2. cgroup资源统计检查
    // 3. 墙上时钟时间监控
    
    for {
        // 检查墙钟时间限制
        if elapsed > wallTimeLimit {
            // 终止进程并记录超时
        }
        
        // 检查cgroup资源限制
        if exceeded, limitType := s.checkCgroupLimits(); exceeded {
            // 根据限制类型终止进程
        }
        
        // 检查setrlimit限制（兜底）
        if rusage.Maxrss > memoryLimit {
            // 内存超限处理
        }
        
        // 进程状态检查...
    }
}
```

## 资源控制策略

### 内存控制策略

| 控制层 | 控制目标 | 配置策略 | 作用 |
|--------|----------|----------|------|
| **cgroups** | 物理内存+swap | 精确限制 | 主要控制手段 |
| **setrlimit** | 虚拟内存 | 2倍物理内存限制 | 防止虚拟内存爆炸 |

```go
// cgroups: 精确控制物理内存
config.MemoryLimitBytes = s.config.MemoryLimit * 1024 // 128MB
config.MemorySwapLimit = config.MemoryLimitBytes      // 禁用swap

// setrlimit: 虚拟内存兜底保护  
memoryLimit := s.config.MemoryLimit * 1024 * 2 // 256MB虚拟内存限制
syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
    Cur: uint64(memoryLimit),
    Max: uint64(memoryLimit),
})
```

### CPU控制策略

| 控制层 | 控制目标 | 配置策略 | 作用 |
|--------|----------|----------|------|
| **cgroups** | CPU使用率 | 配额+周期机制 | 主要控制手段 |
| **setrlimit** | CPU时间 | 2倍预期时间 | 兜底时间保护 |
| **墙上时钟** | 总运行时间 | 父进程监控 | 最终兜底保护 |

```go
// cgroups: CPU使用率控制
config.CPUQuotaUs = 50000   // 50%使用率
config.CPUPeriodUs = 100000 // 100ms周期

// setrlimit: CPU时间兜底
timeLimit := s.config.TimeLimit / 1000 * 2 // 6秒兜底保护
syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{
    Cur: uint64(timeLimit),
    Max: uint64(timeLimit),
})
```

### 进程控制策略

| 控制层 | 控制目标 | 配置策略 | 作用 |
|--------|----------|----------|------|
| **cgroups** | 组内进程总数 | 精确限制 | 主要控制手段 |
| **setrlimit** | 用户进程数 | 配合限制 | 辅助保护 |

## 配置模式

### 1. 混合模式 (hybrid) - 推荐
- **特点**: setrlimit + cgroups 双层控制
- **优势**: 最全面的资源控制和兜底保护
- **适用**: 生产环境，要求最高安全性

### 2. 主控模式 (primary)
- **特点**: 主要依赖cgroups，setrlimit辅助
- **优势**: 精确的资源控制和统计
- **适用**: cgroups环境稳定的场景

### 3. 降级模式 (fallback)
- **特点**: 主要依赖setrlimit，cgroups不可用时使用
- **优势**: 兼容性好，轻量级
- **适用**: cgroups不可用的环境

## 语言特定优化

### 资源配置矩阵

| 语言 | 内存(MB) | CPU配额(%) | Swap比例 | 特点 |
|------|----------|------------|----------|------|
| **C/C++** | 64 | 70% | 1.0 | 编译程序效率高，禁用swap |
| **Java** | 256 | 80% | 1.2 | JVM需要大内存，允许少量swap |
| **Python** | 128 | 60% | 1.1 | 解释执行，适中配置 |
| **Go** | 128 | 75% | 1.0 | 高效编译程序，禁用swap |
| **JavaScript** | 192 | 65% | 1.2 | V8引擎需要较多内存 |

### 自动配置逻辑

```go
func NewLanguageSandboxConfig(language, workDir string) *SandboxConfig {
    config := NewDefaultSandboxConfig(workDir)
    config.Language = language
    
    switch language {
    case "java":
        config.MemoryLimit = 262144      // 256MB
        config.CPUQuotaPercent = 80      // 80%CPU配额
        config.MemorySwapRatio = 1.2     // 允许20%swap
    case "cpp", "c":
        config.MemoryLimit = 65536       // 64MB
        config.CPUQuotaPercent = 70      // 70%CPU配额  
        config.MemorySwapRatio = 1.0     // 禁用swap
    // ... 其他语言配置
    }
    
    return config
}
```

## 性能优化

### 1. cgroup复用策略

```go
// 预创建语言级cgroup，避免每次任务都创建
var languageCgroups = map[string]*CgroupManager{
    "cpp":    NewCgroupManager("judge/cpp"),
    "java":   NewCgroupManager("judge/java"),
    "python": NewCgroupManager("judge/python"),
}

// 任务执行时只创建任务级子组
taskCgroup := languageCgroups["cpp"].CreateSubGroup(taskID)
```

### 2. 批量操作优化

```go
func (c *CgroupManager) BatchSetLimits(limits map[string]string) error {
    // 批量设置cgroup配置，减少系统调用次数
    for file, value := range limits {
        if err := c.writeFile(file, value); err != nil {
            errors = append(errors, err)
        }
    }
    return combineErrors(errors)
}
```

### 3. 异步清理机制

```go
// 异步清理cgroup，避免阻塞判题流程
go func() {
    time.Sleep(5 * time.Second) // 延迟清理，确保进程完全结束
    cgroupManager.Cleanup(taskID)
}()
```

## 详细资源统计

### 统计数据结构

```go
type ResourceUsageDetail struct {
    // setrlimit统计数据
    SetrlimitStats struct {
        CPUTimeUsed    int64 // CPU时间使用(毫秒)
        MaxRSSUsed     int64 // 最大常驻内存(KB)
        WallTimeUsed   int64 // 墙钟时间(毫秒)
    }
    
    // cgroups统计数据
    CgroupsStats struct {
        MemoryUsed      int64   // 内存使用量(字节)
        MemoryPeakUsed  int64   // 内存使用峰值(字节)
        MemoryLimit     int64   // 内存限制(字节)
        CPUUsageTotal   int64   // CPU总使用时间(纳秒)
        CPUUsagePercent float64 // CPU使用率百分比
        CPUThrottled    int64   // CPU被限流次数
        PIDsUsed        int64   // 进程数使用
        IOReadBytes     int64   // I/O读取字节数
        IOWriteBytes    int64   // I/O写入字节数
    }
    
    // 综合判断结果
    LimitExceeded   string // 超限类型："memory", "cpu", "time", "none"
    ControlMethod   string // 控制方式："setrlimit", "cgroups", "hybrid"
    PerformanceData struct {
        SetupTimeMs    int64 // 资源控制设置耗时(毫秒)
        MonitorTimeMs  int64 // 监控耗时(毫秒)
        CleanupTimeMs  int64 // 清理耗时(毫秒)
    }
}
```

### 统计数据收集

```go
func (s *SystemCallSandbox) collectResourceStats(result *ExecuteResult, pid int) error {
    // 收集setrlimit统计（从rusage）
    result.ResourceUsage.SetrlimitStats.CPUTimeUsed = rusage.Utime + rusage.Stime
    result.ResourceUsage.SetrlimitStats.MaxRSSUsed = rusage.Maxrss
    
    // 收集cgroup统计（从cgroup文件系统）
    if stats, err := s.cgroupManager.GetStats(); err == nil {
        result.ResourceUsage.CgroupsStats.MemoryUsed = stats.MemoryUsage
        result.ResourceUsage.CgroupsStats.MemoryPeakUsed = stats.MemoryMaxUsage
        result.ResourceUsage.CgroupsStats.CPUUsageTotal = stats.CPUUsageTotal
        // ... 其他统计数据
    }
    
    return nil
}
```

## 错误处理和降级

### 降级策略

```go
func (s *SystemCallSandbox) setupResourceLimits() error {
    // 优先使用cgroups
    if err := s.setupCgroups(); err != nil {
        logx.Errorf("cgroups setup failed, fallback to setrlimit: %v", err)
        // 降级到纯setrlimit模式
        return s.setupSetrlimitOnly()
    }
    
    // cgroups成功，再设置setrlimit作为兜底
    return s.setupSetrlimit()
}
```

### 资源限制验证

```go
func (s *SystemCallSandbox) validateResourceLimits(pid int) error {
    // 检查进程是否在正确的cgroup中
    if !s.cgroupManager.IsProcessInGroup(pid) {
        return fmt.Errorf("process not in expected cgroup")
    }
    
    // 检查setrlimit是否生效
    if !s.validateSetrlimits(pid) {
        return fmt.Errorf("setrlimit validation failed")
    }
    
    return nil
}
```

## 使用示例

### 基础使用

```go
// 创建启用双层资源控制的沙箱配置
config := sandbox.NewLanguageSandboxConfig("cpp", workDir)
config.TaskID = "task_001"
config.EnableCgroups = true
config.CgroupsMode = "hybrid"

// 创建沙箱并执行
sb := sandbox.NewSystemCallSandbox(config)
result, err := sb.Execute(ctx, executable, args)

// 查看详细资源使用统计
if result.ResourceUsage != nil {
    fmt.Printf("控制方法: %s\n", result.ResourceUsage.ControlMethod)
    fmt.Printf("内存峰值: %.2f MB\n", 
        float64(result.ResourceUsage.CgroupsStats.MemoryPeakUsed)/1024/1024)
    fmt.Printf("CPU使用率: %.2f%%\n", 
        result.ResourceUsage.CgroupsStats.CPUUsagePercent)
}
```

### 自定义配置

```go
config := &sandbox.SandboxConfig{
    // 基础配置
    UID:     65534,
    GID:     65534,
    WorkDir: workDir,
    
    // 资源限制
    MemoryLimit:   131072, // 128MB
    TimeLimit:     5000,   // 5秒
    ProcessLimit:  10,     // 10个进程
    
    // cgroups配置
    EnableCgroups:    true,
    CgroupsMode:      "hybrid",
    TaskID:           "custom_task",
    Language:         "cpp",
    CPUQuotaPercent:  60,    // 60% CPU配额
    MemorySwapRatio:  1.0,   // 禁用swap
    IOWeightPercent:  50,    // 50% I/O权重
}
```

## 技术优势总结

### 1. 精确控制
- **内存**: cgroups精确控制物理内存，setrlimit防止虚拟内存爆炸
- **CPU**: cgroups控制使用率，setrlimit+墙上时钟双重时间保护
- **进程**: cgroups组级别控制，setrlimit用户级别限制

### 2. 全面监控
- **实时统计**: cgroups提供详细的资源使用统计
- **性能分析**: 记录资源控制各阶段耗时
- **超限诊断**: 精确识别超限类型和原因

### 3. 可靠兜底
- **多层防护**: setrlimit基础防护 + cgroups精确控制 + 墙上时钟兜底
- **降级机制**: cgroups不可用时自动降级到setrlimit模式
- **错误恢复**: 完善的错误处理和资源清理机制

### 4. 性能优化
- **cgroup复用**: 预创建语言级cgroup，减少创建开销
- **批量操作**: 批量设置配置，减少系统调用
- **异步清理**: 异步清理资源，不阻塞判题流程

通过这套完整的setrlimit+cgroups双层资源控制方案，判题服务实现了精确、可靠、高效的资源管理，为安全的代码执行环境提供了坚实的保障。

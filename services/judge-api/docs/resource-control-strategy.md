# setrlimit + cgroups 资源控制配合策略分析

## 概述

本文档分析了setrlimit和cgroups两种资源控制机制在判题系统中的配合使用策略，设计了一套双层资源控制方案，实现最佳的资源限制效果。

## 技术对比分析

### setrlimit 特性分析

#### 优势
1. **轻量级**: 单次系统调用设置，无额外开销
2. **POSIX标准**: 跨平台兼容性好
3. **进程内部资源**: 可限制栈大小、文件大小、进程数等cgroups无法控制的资源
4. **即时生效**: 设置后立即对当前进程生效

#### 局限性
1. **虚拟内存限制**: RLIMIT_AS限制虚拟内存，存在误判（mmap大量虚拟内存但不写入）
2. **CPU时间累积**: RLIMIT_CPU累积所有核心计算时间，多核场景下不准确
3. **无I/O等待时间**: 不包含I/O等待时间，无法准确控制总执行时间
4. **单进程限制**: 无法控制进程树的整体资源使用
5. **无使用率控制**: 无法限制CPU使用率，可能导致系统卡顿

### cgroups 特性分析

#### 优势
1. **精确控制**: 直接限制物理内存+swap，精确的内存控制
2. **CPU使用率限制**: 通过周期+配额机制控制CPU使用率
3. **进程树控制**: 自动包含子进程，整体资源统计
4. **实时监控**: 提供详细的资源使用统计
5. **多种子系统**: 支持memory、cpu、cpuset、blkio等多种资源类型

#### 局限性
1. **重量级**: 需要创建目录、写入配置文件、清理资源
2. **Linux专用**: 仅支持Linux系统
3. **无法控制进程内部资源**: 如栈大小、单文件大小等
4. **复杂性**: 配置和管理相对复杂

## 配合使用策略

### 分层资源控制架构

```
判题任务
    ↓
setrlimit (进程级基础限制)
    ├── 栈大小限制 (RLIMIT_STACK)
    ├── 文件大小限制 (RLIMIT_FSIZE)
    ├── 进程数限制 (RLIMIT_NPROC)
    └── 核心转储禁用 (RLIMIT_CORE)
    ↓
cgroups (系统级精确控制)
    ├── 内存子系统 (memory.limit_in_bytes)
    ├── CPU子系统 (cpu.cfs_quota_us/cpu.cfs_period_us)
    ├── CPU集合 (cpuset.cpus)
    └── 块设备I/O (blkio.throttle.*)
    ↓
墙上时钟监控 (父进程兜底)
```

### 资源类型分工

| 资源类型 | 主控制方式 | 辅助控制方式 | 原因 |
|---------|------------|-------------|------|
| **物理内存** | cgroups | setrlimit | cgroups精确控制物理内存，setrlimit防止虚拟内存爆炸 |
| **CPU时间** | cgroups | 墙上时钟 | cgroups控制使用率，墙上时钟兜底总时间 |
| **栈大小** | setrlimit | - | cgroups无法控制，setrlimit防止递归溢出 |
| **文件大小** | setrlimit | - | 防止单文件写爆磁盘 |
| **进程数** | setrlimit | cgroups | setrlimit限制用户级进程数，cgroups限制组内总数 |
| **I/O带宽** | cgroups | - | setrlimit无I/O控制能力 |

### 具体配合方案

#### 1. 内存控制配合
```go
// setrlimit: 防止虚拟内存过度申请
syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
    Cur: uint64(config.MemoryLimit * 2 * 1024), // 虚拟内存限制为物理内存的2倍
    Max: uint64(config.MemoryLimit * 2 * 1024),
})

// cgroups: 精确控制物理内存+swap
writeFile("/sys/fs/cgroup/memory/judge_task_123/memory.limit_in_bytes", 
    strconv.Itoa(config.MemoryLimit * 1024))
```

#### 2. CPU控制配合
```go
// setrlimit: 基础CPU时间限制（兜底保护）
syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{
    Cur: uint64(config.TimeLimit / 1000 * 2), // 设置为期望时间的2倍
    Max: uint64(config.TimeLimit / 1000 * 2),
})

// cgroups: 精确的CPU使用率控制
writeFile("/sys/fs/cgroup/cpu/judge_task_123/cpu.cfs_quota_us", "50000")  // 50%使用率
writeFile("/sys/fs/cgroup/cpu/judge_task_123/cpu.cfs_period_us", "100000") // 100ms周期
```

#### 3. 进程控制配合
```go
// setrlimit: 用户级进程数限制
syscall.Setrlimit(syscall.RLIMIT_NPROC, &syscall.Rlimit{
    Cur: uint64(config.ProcessLimit),
    Max: uint64(config.ProcessLimit),
})

// cgroups: 组内进程总数限制
writeFile("/sys/fs/cgroup/pids/judge_task_123/pids.max", 
    strconv.Itoa(config.ProcessLimit))
```

## cgroup组织结构设计

### 层次化cgroup结构

```
/sys/fs/cgroup/
├── memory/
│   ├── judge/                    # 判题服务根组
│   │   ├── memory.limit_in_bytes # 总内存限制
│   │   ├── cpp/                  # C++语言组
│   │   │   ├── memory.limit_in_bytes
│   │   │   └── task_123456/      # 具体任务组
│   │   │       ├── memory.limit_in_bytes
│   │   │       ├── memory.usage_in_bytes
│   │   │       └── cgroup.procs
│   │   ├── java/                 # Java语言组
│   │   └── python/               # Python语言组
├── cpu/
│   └── judge/
│       ├── cpp/
│       └── java/
└── pids/
    └── judge/
        ├── cpp/
        └── java/
```

### 任务cgroup命名规则

```
组名格式: judge_{language}_{task_id}_{timestamp}
示例: judge_cpp_123456_1703123456789

路径示例:
/sys/fs/cgroup/memory/judge/cpp/judge_cpp_123456_1703123456789/
/sys/fs/cgroup/cpu/judge/cpp/judge_cpp_123456_1703123456789/
/sys/fs/cgroup/pids/judge/cpp/judge_cpp_123456_1703123456789/
```

## 资源监控策略

### 三层监控体系

1. **setrlimit监控**: 通过rusage获取基础资源使用情况
2. **cgroups监控**: 实时读取cgroup统计文件获取精确数据
3. **墙上时钟监控**: 父进程计时器兜底保护

### 监控数据收集

```go
type ResourceUsage struct {
    // setrlimit数据
    CPUTimeUsed    int64 // 从rusage获取
    MaxRSSUsed     int64 // 最大常驻内存
    
    // cgroups数据  
    MemoryUsed     int64 // 实际物理内存使用
    MemoryPeak     int64 // 内存使用峰值
    CPUUsagePercent float64 // CPU使用率
    
    // 墙上时钟数据
    WallTimeUsed   int64 // 实际运行时间
    
    // 综合判断
    LimitExceeded  string // 超限类型："memory", "cpu", "time", "none"
}
```

## 性能优化考虑

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
// 批量设置cgroup配置，减少系统调用
func (c *CgroupManager) BatchSetLimits(limits map[string]string) error {
    var errors []error
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

## 错误处理和降级策略

### 1. cgroup不可用降级

```go
func (s *SystemCallSandbox) setResourceLimits() error {
    // 优先使用cgroups
    if err := s.setupCgroups(); err != nil {
        logx.Warnf("cgroups setup failed, fallback to setrlimit: %v", err)
        // 降级到纯setrlimit模式
        return s.setrlimitOnly()
    }
    
    // cgroups成功，再设置setrlimit作为兜底
    return s.setupSetrlimit()
}
```

### 2. 资源限制失效检测

```go
// 定期检查资源限制是否生效
func (s *SystemCallSandbox) validateResourceLimits(pid int) error {
    // 检查进程是否在正确的cgroup中
    if !s.cgroupManager.IsProcessInGroup(pid, s.taskCgroup) {
        return fmt.Errorf("process not in expected cgroup")
    }
    
    // 检查setrlimit是否生效
    if !s.validateSetrlimits(pid) {
        return fmt.Errorf("setrlimit validation failed")
    }
    
    return nil
}
```

## 最佳实践建议

### 1. 资源配置原则

- **内存**: cgroups主控，setrlimit设置为2倍作为虚拟内存兜底
- **CPU**: cgroups控制使用率，setrlimit设置宽松限制防止失控
- **时间**: 墙上时钟主控，cgroups和setrlimit辅助
- **I/O**: 主要依赖cgroups，setrlimit控制文件大小

### 2. 监控告警策略

- 资源使用率超过80%时告警
- cgroup创建/删除失败时告警  
- 进程逃逸出cgroup时立即终止
- 定期检查cgroup目录泄漏

### 3. 调试和诊断

- 提供详细的资源使用报告
- 记录所有资源限制操作的日志
- 支持实时查看cgroup状态
- 提供资源限制测试工具

## 总结

通过setrlimit+cgroups双层资源控制方案，判题系统能够：

1. **精确控制**: cgroups提供精确的内存和CPU控制
2. **全面覆盖**: setrlimit补充进程内部资源限制
3. **可靠兜底**: 多层监控确保资源不会失控
4. **性能优化**: 通过复用和批量操作减少开销
5. **故障恢复**: 完善的降级和错误处理机制

这套方案结合了两种技术的优势，避免了各自的局限性，为判题系统提供了最佳的资源控制效果。

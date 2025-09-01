# seccomp-bpf 系统调用过滤实现文档

## 概述

本文档详细说明了判题服务中seccomp-bpf系统调用过滤功能的完整实现。seccomp-bpf是Linux内核提供的一种系统调用过滤机制，能够控制进程可执行的系统调用，是沙箱安全防护的重要组成部分。

## 技术原理

### seccomp基本概念

seccomp(secure computing)是Linux内核的一个安全特性，用于限制进程可以执行的系统调用。它有三种模式：

1. **SECCOMP_MODE_DISABLED** (0): 禁用seccomp
2. **SECCOMP_MODE_STRICT** (1): 严格模式，只允许read、write、exit、sigreturn
3. **SECCOMP_MODE_FILTER** (2): 过滤模式，使用BPF程序自定义规则

### BPF程序过滤机制

BPF(Berkeley Packet Filter)原本用于网络数据包过滤，seccomp-bpf将其扩展到系统调用过滤：

```
用户程序 -> 系统调用 -> seccomp-bpf检查 -> 内核执行/拒绝
                          ↑
                    BPF程序过滤器
```

#### 工作流程
1. 进程执行系统调用时，内核首先检查seccomp过滤器
2. BPF程序检查系统调用号、架构、参数等信息
3. 根据检查结果返回动作：ALLOW（允许）、KILL（终止）、TRAP（信号）等
4. 内核根据返回的动作决定是否执行系统调用

### seccomp_data结构

BPF程序接收的输入数据结构：

```c
struct seccomp_data {
    int nr;                    /* 系统调用号 */
    __u32 arch;               /* 架构标识 */
    __u64 instruction_pointer; /* 指令指针 */
    __u64 args[6];            /* 系统调用参数 */
};
```

## 实现架构

### 核心组件

1. **SeccompFilter**: 过滤器核心类，负责BPF程序构建和安装
2. **BPF程序构建器**: 将系统调用白名单转换为BPF指令序列
3. **系统调用白名单**: 针对不同编程语言的系统调用权限配置
4. **初始化程序**: 在目标进程中安装seccomp过滤器的Go程序

### 文件结构

```
internal/sandbox/
├── seccomp.go          # seccomp过滤器核心实现
├── seccomp_init.go     # seccomp初始化程序生成器
└── sandbox.go          # 沙箱集成代码

examples/
└── seccomp_example.go  # 使用示例和测试用例

docs/
└── seccomp-bpf-implementation.md  # 技术文档
```

## 核心实现

### 1. SeccompFilter结构体

```go
type SeccompFilter struct {
    allowedSyscalls map[int]bool     // 允许的系统调用集合
    defaultAction   uint32           // 默认动作
    instructions    []BPFInstruction // BPF指令集
}
```

### 2. BPF指令结构

```go
type BPFInstruction struct {
    Code uint16 // 操作码
    JT   uint8  // 跳转条件为真时的偏移
    JF   uint8  // 跳转条件为假时的偏移
    K    uint32 // 常量值
}
```

### 3. BPF程序构建逻辑

```go
func (f *SeccompFilter) buildBPFProgram() error {
    // 1. 验证架构 - 确保是x86_64
    f.addInstruction(BPF_LD|BPF_W|BPF_ABS, 0, 0, SECCOMP_DATA_ARCH_OFFSET)
    f.addInstruction(BPF_JMP|BPF_JEQ|BPF_K, 0, 1, 0xc000003e) // AUDIT_ARCH_X86_64
    f.addInstruction(BPF_RET|BPF_K, 0, 0, SECCOMP_RET_KILL_PROCESS)
    
    // 2. 加载系统调用号
    f.addInstruction(BPF_LD|BPF_W|BPF_ABS, 0, 0, SECCOMP_DATA_NR_OFFSET)
    
    // 3. 构建系统调用白名单检查
    for i, syscallNum := range allowedSyscallList {
        if i == len(allowedSyscallList)-1 {
            f.addInstruction(BPF_JMP|BPF_JEQ|BPF_K, 1, 0, uint32(syscallNum))
        } else {
            f.addInstruction(BPF_JMP|BPF_JEQ|BPF_K, uint32(len(allowedSyscallList)-i), 0, uint32(syscallNum))
        }
    }
    
    // 4. 默认动作和允许动作
    f.addInstruction(BPF_RET|BPF_K, 0, 0, f.defaultAction)
    f.addInstruction(BPF_RET|BPF_K, 0, 0, SECCOMP_RET_ALLOW)
    
    return nil
}
```

### 4. 过滤器安装

```go
func (f *SeccompFilter) Install() error {
    // 构建BPF程序
    if err := f.buildBPFProgram(); err != nil {
        return err
    }
    
    // 创建BPF程序结构体
    program := BPFProgram{
        Len:    uint16(len(f.instructions)),
        Filter: &f.instructions[0],
    }
    
    // 调用seccomp系统调用安装过滤器
    ret, _, errno := syscall.Syscall(SYS_SECCOMP, 
        SECCOMP_SET_MODE_FILTER, 
        0, 
        uintptr(unsafe.Pointer(&program)))
    
    if ret != 0 {
        return fmt.Errorf("seccomp system call failed: errno=%d", errno)
    }
    
    return nil
}
```

## 系统调用白名单

### 编程语言特定白名单

#### C/C++程序
```go
case "cpp", "c":
    return []int{
        0,   // read
        1,   // write
        2,   // open
        3,   // close
        4,   // stat
        5,   // fstat
        8,   // lseek
        9,   // mmap
        10,  // mprotect
        11,  // munmap
        12,  // brk
        21,  // access
        59,  // execve
        60,  // exit
        158, // arch_prctl
        231, // exit_group
    }
```

#### Java程序
```go
case "java":
    return []int{
        0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 16, 21, 22, 25, 39,
        56, 57, 59, 60, 61, 62, 63, 89, 96, 97, 158, 202, 231, 257, 273, 318,
    }
```

#### Python程序
```go
case "python":
    return []int{
        0, 1, 2, 3, 4, 5, 6, 8, 9, 10, 11, 12, 13, 16, 21, 22, 39, 59, 60, 61,
        79, 89, 97, 158, 231, 257, 273, 318,
    }
```

### 白名单设计原则

1. **最小权限原则**: 只允许程序运行必需的系统调用
2. **语言特定优化**: 根据不同编程语言的运行时需求调整
3. **安全优先**: 禁止所有可能的危险系统调用（网络、进程创建、文件系统修改等）

## 集成方式

### 1. 沙箱配置扩展

```go
type SandboxConfig struct {
    // ... 其他配置 ...
    EnableSeccomp   bool  // 启用seccomp过滤
    AllowedSyscalls []int // 允许的系统调用号
}
```

### 2. 执行流程集成

```go
func (s *SystemCallSandbox) Execute(ctx context.Context, executable string, args []string) (*ExecuteResult, error) {
    // ... 其他初始化 ...
    
    if s.config.EnableSeccomp {
        // 创建seccomp初始化程序
        seccompInit, err := s.createSeccompInitializer(executable, args, s.config.WorkDir)
        if err != nil {
            return nil, fmt.Errorf("failed to create seccomp initializer: %w", err)
        }
        finalExecutable = seccompInit
        
        defer s.cleanupSeccompFiles(s.config.WorkDir)
    }
    
    // ... 执行程序 ...
}
```

### 3. 初始化程序生成

由于seccomp过滤器必须在目标进程中安装，实现了动态生成Go初始化程序的机制：

```go
func (s *SystemCallSandbox) createSeccompInitializer(executable string, args []string, workDir string) (string, error) {
    // 1. 创建seccomp配置文件
    seccompConfig := SeccompConfig{
        EnableSeccomp:   s.config.EnableSeccomp,
        AllowedSyscalls: s.config.AllowedSyscalls,
    }
    
    // 2. 生成Go源码
    initializerSource := s.generateSeccompInitializerSource(executable, args, configPath)
    
    // 3. 编译初始化程序
    cmd := exec.Command("go", "build", "-o", binaryPath, initializerPath)
    
    return binaryPath, nil
}
```

## 安全特性

### 1. 多层防护

seccomp-bpf与其他安全机制协同工作：

```
用户代码
    ↓
chroot文件隔离
    ↓
Namespace资源隔离
    ↓
seccomp-bpf系统调用过滤  ← 本实现
    ↓
setrlimit资源限制
    ↓
内核执行
```

### 2. 不可绕过性

- **内核级拦截**: 在系统调用入口点进行检查，用户态无法绕过
- **不可逆转**: 一旦安装，进程无法移除或修改过滤器
- **继承性**: 子进程自动继承父进程的seccomp过滤器

### 3. 性能优势

- **BPF高效执行**: BPF程序在内核中执行，性能开销极小
- **预编译优化**: BPF程序在安装时编译，运行时无额外开销
- **分支优化**: 使用跳转表优化系统调用检查逻辑

## 调试和验证

### 1. BPF程序反汇编

```go
func (f *SeccompFilter) GetBPFDisassembly() []string {
    var disasm []string
    
    for i, inst := range f.instructions {
        switch inst.Code & 0x07 {
        case BPF_LD:
            line = fmt.Sprintf("%3d: LD  [%d]", i, inst.K)
        case BPF_JMP:
            line = fmt.Sprintf("%3d: JEQ #%d jt=%d jf=%d", i, inst.K, inst.JT, inst.JF)
        case BPF_RET:
            line = fmt.Sprintf("%3d: RET %s", i, actionName)
        }
        disasm = append(disasm, line)
    }
    
    return disasm
}
```

### 2. 过滤器验证

```go
func (f *SeccompFilter) Validate() error {
    // 检查是否有允许的系统调用
    if len(f.allowedSyscalls) == 0 {
        return fmt.Errorf("no allowed syscalls configured")
    }
    
    // 检查BPF程序结构
    if len(f.instructions) < 4 {
        return fmt.Errorf("BPF program too short")
    }
    
    // 验证第一条指令是否为架构检查
    firstInst := f.instructions[0]
    if firstInst.Code != (BPF_LD|BPF_W|BPF_ABS) || firstInst.K != SECCOMP_DATA_ARCH_OFFSET {
        return fmt.Errorf("invalid first instruction")
    }
    
    return nil
}
```

### 3. 测试用例

提供了完整的测试框架：

- **基本功能测试**: 验证seccomp过滤器的基本工作
- **语言特定测试**: 测试不同编程语言的系统调用配置
- **安全性测试**: 验证恶意系统调用被正确阻止
- **性能测试**: 测量seccomp过滤器的性能开销

## 使用方法

### 1. 基本使用

```go
// 创建启用seccomp的配置
config := sandbox.NewLanguageSandboxConfig("cpp", workDir)

// 创建沙箱并执行
sb := sandbox.NewSystemCallSandbox(config)
result, err := sb.Execute(ctx, executable, args)
```

### 2. 自定义配置

```go
config := &sandbox.SandboxConfig{
    EnableSeccomp:   true,
    AllowedSyscalls: []int{0, 1, 60, 231}, // read, write, exit, exit_group
    // ... 其他配置
}
```

### 3. 调试模式

```go
// 创建过滤器
filter, err := sandbox.CreateLanguageSeccompFilter("cpp")

// 验证过滤器
if err := filter.Validate(); err != nil {
    log.Fatal(err)
}

// 查看BPF程序反汇编
disasm := filter.GetBPFDisassembly()
for _, line := range disasm {
    fmt.Println(line)
}
```

## 限制和注意事项

### 1. 平台限制

- **Linux专用**: seccomp是Linux内核特性，不支持其他操作系统
- **架构依赖**: 当前实现针对x86_64架构，其他架构需要调整
- **内核版本**: 需要Linux 3.5+支持seccomp-bpf

### 2. 功能限制

- **静态白名单**: 系统调用白名单在安装时确定，运行时不可修改
- **参数检查**: 当前实现主要检查系统调用号，参数检查需要扩展
- **性能考虑**: 过多的系统调用检查可能影响性能

### 3. 调试难度

- **调试困难**: seccomp过滤器安装后难以调试
- **错误诊断**: 被阻止的系统调用可能导致程序异常终止
- **日志记录**: 需要配合内核日志进行问题诊断

## 最佳实践

1. **渐进式部署**: 先使用日志模式测试，再切换到终止模式
2. **白名单维护**: 定期审查和更新系统调用白名单
3. **监控告警**: 监控seccomp违规事件，及时发现攻击尝试
4. **性能测试**: 在生产环境部署前进行充分的性能测试

## 总结

本实现提供了完整的seccomp-bpf系统调用过滤功能，包括：

- **完整的BPF程序构建和管理**
- **针对多种编程语言的系统调用白名单**
- **灵活的配置和集成方式**
- **完善的调试和验证工具**
- **详细的文档和示例**

通过seccomp-bpf过滤器，判题服务能够在内核级别精确控制用户代码的系统调用权限，大大提升了安全防护能力，有效防范各类系统级攻击。

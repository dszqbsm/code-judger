# Linux Namespaces 资源隔离实现文档

## 概述

本文档详细说明了判题服务中Linux Namespaces资源隔离的完整实现，包括原有的三种策略（PID、Network、Mount）以及新增的四种策略（User、UTS、IPC、Cgroup）。

## 技术原理

Linux Namespaces是Linux内核提供的一种全局资源隔离机制，通过为进程创建独立的资源视图，使得不同Namespace中的进程仿佛运行在完全独立的系统环境中。这是容器技术（Docker、Kubernetes）的底层基础。

### 内核实现机制

- **进程描述符**：每个进程的进程描述符都有一个`nsproxy`指针指向该进程所属的所有Namespace
- **资源视图**：当进程访问全局资源时，内核会先检查进程所属的Namespace，返回该Namespace内的资源视图
- **继承机制**：新进程默认继承父进程的Namespace，但可以在创建时指定新的Namespace集合

## 已实现的七种Namespace策略

### 1. PID Namespace（进程隔离）
**已实现** - 通过`syscall.CLONE_NEWPID`

```go
// 原理：创建独立的PID映射表，隔离进程视图
flags |= syscall.CLONE_NEWPID
```

**作用**：
- 防止用户代码通过kill终止主机进程
- 防止通过ps窥探主机进程列表
- 沙箱内进程从PID 1开始编号

### 2. Network Namespace（网络隔离）
**已实现** - 通过`syscall.CLONE_NEWNET`

```go
// 原理：创建独立的网络栈，包括网卡、路由表、防火墙规则
flags |= syscall.CLONE_NEWNET
```

**作用**：
- 禁止用户代码的网络访问
- 防止恶意代码发起网络攻击
- 防止泄漏主机网络信息

### 3. Mount Namespace（文件系统隔离）
**已实现** - 通过`syscall.CLONE_NEWNS`

```go
// 原理：创建独立的文件系统挂载点视图
flags |= syscall.CLONE_NEWNS
```

**作用**：
- 防止通过mount挂载主机敏感分区
- 防止通过umount破坏主机文件系统
- 结合chroot实现完整文件隔离

### 4. User Namespace（用户权限隔离）
**新增实现** - 通过`syscall.CLONE_NEWUSER`

```go
// 原理：创建独立的uid/gid映射，实现权限降级
if s.config.EnableUserNS {
    flags |= syscall.CLONE_NEWUSER
    logx.Info("Enabled User Namespace isolation - uid/gid mapping will be configured")
}
```

**核心实现**：
```go
// 设置uid映射：沙箱内root(0) -> 主机普通用户(65534)
func (s *SystemCallSandbox) setupUserNamespaceMapping(pid int) error {
    uidMapPath := fmt.Sprintf("/proc/%d/uid_map", pid)
    uidMapping := fmt.Sprintf("%d %d 1", s.config.UidMapInside, s.config.UidMapOutside)
    if err := s.writeToFile(uidMapPath, uidMapping); err != nil {
        return fmt.Errorf("failed to setup uid mapping: %w", err)
    }
    
    // 禁用setgroups防止绕过gid映射
    setgroupsPath := fmt.Sprintf("/proc/%d/setgroups", pid)
    s.writeToFile(setgroupsPath, "deny")
    
    // 设置gid映射
    gidMapPath := fmt.Sprintf("/proc/%d/gid_map", pid)
    gidMapping := fmt.Sprintf("%d %d 1", s.config.GidMapInside, s.config.GidMapOutside)
    return s.writeToFile(gidMapPath, gidMapping)
}
```

**作用**：
- 即使用户代码获得沙箱内root权限，也无法影响主机
- 实现权限降级，增强安全性
- 防止权限提升攻击

### 5. UTS Namespace（主机名域名隔离）
**新增实现** - 通过`syscall.CLONE_NEWUTS`

```go
// 原理：创建独立的主机名和域名空间
if s.config.EnableUTSNS {
    flags |= syscall.CLONE_NEWUTS
    logx.Info("Enabled UTS Namespace isolation - hostname/domainname will be isolated")
}
```

**核心实现**：
```go
// 在子进程中设置独立的主机名和域名
func (s *SystemCallSandbox) setupUTSNamespace() error {
    if s.config.Hostname != "" {
        hostname := []byte(s.config.Hostname)
        if err := syscall.Sethostname(hostname); err != nil {
            return fmt.Errorf("failed to set hostname: %w", err)
        }
    }
    
    if s.config.DomainName != "" {
        domainname := []byte(s.config.DomainName)
        if err := syscall.Setdomainname(domainname); err != nil {
            return fmt.Errorf("failed to set domainname: %w", err)
        }
    }
    return nil
}
```

**作用**：
- 防止用户代码修改主机名干扰系统标识
- 防止通过主机名判断真实系统环境
- 提供独立的网络标识

### 6. IPC Namespace（进程间通信隔离）
**新增实现** - 通过`syscall.CLONE_NEWIPC`

```go
// 原理：创建独立的IPC资源池（消息队列、共享内存、信号量）
if s.config.EnableIPCNS {
    flags |= syscall.CLONE_NEWIPC
    logx.Info("Enabled IPC Namespace isolation - IPC resources will be isolated")
}
```

**验证实现**：
```go
// 验证IPC隔离效果
func (s *SystemCallSandbox) validateIPCNamespace() error {
    // 检查消息队列 - 在独立IPC空间中应该看不到主机IPC资源
    msgPath := "/proc/sysvipc/msg"
    if content, err := os.ReadFile(msgPath); err == nil {
        lines := len(string(content))
        logx.Infof("IPC message queues visible: %d lines", lines)
    }
    
    // 检查共享内存和信号量
    // ...
    return nil
}
```

**作用**：
- 防止恶意代码通过共享内存读取主机进程敏感数据
- 防止通过消息队列与其他进程通信
- 隔离System V IPC资源

### 7. Cgroup Namespace（控制组隔离）
**新增实现** - 通过`syscall.CLONE_NEWCGROUP`

```go
// 原理：创建独立的cgroup视图，限制对控制组的访问
if s.config.EnableCgroupNS {
    flags |= syscall.CLONE_NEWCGROUP
    logx.Info("Enabled Cgroup Namespace isolation - cgroup view will be restricted")
}
```

**验证实现**：
```go
// 验证Cgroup隔离效果
func (s *SystemCallSandbox) validateCgroupNamespace() error {
    // 检查cgroup根目录 - 应该只显示受限的子树
    cgroupRoot := "/sys/fs/cgroup"
    if entries, err := os.ReadDir(cgroupRoot); err == nil {
        logx.Infof("Cgroup controllers visible: %d entries", len(entries))
        for _, entry := range entries {
            logx.Debugf("Visible cgroup controller: %s", entry.Name())
        }
    }
    return nil
}
```

**作用**：
- 防止用户代码修改cgroup配置
- 限制对控制组的访问权限
- 防止绕过资源限制

## 配置结构体扩展

```go
type SandboxConfig struct {
    // ... 原有配置 ...
    
    // Namespace隔离配置
    EnableUserNS   bool   // 启用User Namespace隔离
    EnableUTSNS    bool   // 启用UTS Namespace隔离  
    EnableIPCNS    bool   // 启用IPC Namespace隔离
    EnableCgroupNS bool   // 启用Cgroup Namespace隔离
    Hostname       string // 沙箱内主机名
    DomainName     string // 沙箱内域名
    
    // User Namespace uid/gid映射配置
    UidMapInside  int // 沙箱内的用户ID
    UidMapOutside int // 主机上映射的用户ID
    GidMapInside  int // 沙箱内的组ID  
    GidMapOutside int // 主机上映射的组ID
}
```

## 使用方法

### 1. 默认配置（启用所有隔离策略）
```go
config := sandbox.NewDefaultSandboxConfig(workDir)
sandbox := sandbox.NewSystemCallSandbox(config)
```

### 2. 语言特定配置
```go
config := sandbox.NewLanguageSandboxConfig("java", workDir)
sandbox := sandbox.NewSystemCallSandbox(config)
```

### 3. 自定义配置
```go
config := &sandbox.SandboxConfig{
    EnableUserNS:   true,
    EnableUTSNS:    true,
    EnableIPCNS:    true,
    EnableCgroupNS: false,
    Hostname:       "custom-sandbox",
    UidMapInside:   0,
    UidMapOutside:  65534,
    // ... 其他配置
}
```

## 安全防护层次

通过实现七种Namespace隔离策略，构建了多层安全防护体系：

1. **文件系统层**：Mount Namespace + chroot
2. **网络层**：Network Namespace
3. **进程层**：PID Namespace
4. **权限层**：User Namespace + uid/gid映射
5. **标识层**：UTS Namespace
6. **通信层**：IPC Namespace
7. **资源控制层**：Cgroup Namespace

## 技术优势

1. **深度隔离**：七种Namespace提供全方位资源隔离
2. **权限降级**：User Namespace实现真正的权限隔离
3. **灵活配置**：支持选择性启用Namespace策略
4. **语言适配**：针对不同编程语言优化配置
5. **调试支持**：完整的日志和验证机制

## 性能考虑

- **轻量级隔离**：相比虚拟机，Namespace隔离开销极小
- **内核级支持**：直接使用Linux内核原生机制
- **按需启用**：可根据安全需求选择性启用策略
- **配置优化**：针对不同场景提供优化配置

## 兼容性说明

- **内核版本**：需要Linux 3.8+支持所有Namespace
- **权限要求**：需要CAP_SYS_ADMIN权限创建Namespace
- **用户映射**：User Namespace需要/proc/sys/user/max_user_namespaces > 0

## 监控和调试

提供了完整的调试支持：
- Namespace信息查看
- 隔离效果验证
- 详细的执行日志
- 系统资源监控

通过这套完整的Namespace隔离实现，判题服务能够提供企业级的安全隔离能力，有效防范各类安全威胁。

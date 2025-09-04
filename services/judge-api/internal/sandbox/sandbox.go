package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/zeromicro/go-zero/core/logx"
)

// 执行状态常量
const (
	StatusAccepted = iota
	StatusTimeLimitExceeded
	StatusMemoryLimitExceeded
	StatusOutputLimitExceeded
	StatusRuntimeError
	StatusSystemError
	StatusCompileError
)

// 沙箱配置
type SandboxConfig struct {
	// 基础配置
	UID     int    // 运行用户ID
	GID     int    // 运行组ID
	Chroot  string // chroot根目录
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

	// 输入输出
	InputFile  string // 输入文件路径
	OutputFile string // 输出文件路径
	ErrorFile  string // 错误输出文件路径

	// 环境变量
	Environment []string // 环境变量

	// Namespace隔离配置
	EnableUserNS   bool   // 启用User Namespace隔离
	EnableUTSNS    bool   // 启用UTS Namespace隔离
	EnableIPCNS    bool   // 启用IPC Namespace隔离
	EnableCgroupNS bool   // 启用Cgroup Namespace隔离
	Hostname       string // 沙箱内主机名（用于UTS Namespace）
	DomainName     string // 沙箱内域名（用于UTS Namespace）

	// User Namespace uid/gid映射配置
	UidMapInside  int // 沙箱内的用户ID
	UidMapOutside int // 主机上映射的用户ID
	GidMapInside  int // 沙箱内的组ID
	GidMapOutside int // 主机上映射的组ID

	// cgroups资源控制配置
	EnableCgroups bool   // 启用cgroups资源控制
	CgroupsMode   string // cgroups模式：primary(主控), fallback(降级), hybrid(混合)
	TaskID        string // 任务ID（用于cgroup命名）
	Language      string // 编程语言（用于cgroup分组）

	// cgroups高级配置
	CPUQuotaPercent int     // CPU配额百分比（0-100）
	CPUSetCores     string  // 绑定的CPU核心（如"0-1"或"0,2"）
	MemorySwapRatio float64 // 内存swap比例（swap = memory * ratio）
	IOWeightPercent int     // I/O权重百分比（10-1000）
}

// 执行结果
type ExecuteResult struct {
	Status      int    // 执行状态
	ExitCode    int    // 退出码
	Signal      int    // 信号
	TimeUsed    int64  // 实际使用时间(毫秒)
	MemoryUsed  int64  // 实际使用内存(KB)
	OutputSize  int64  // 输出大小
	ErrorOutput string // 错误信息

	// 详细资源使用统计
	ResourceUsage *ResourceUsageDetail `json:"resource_usage,omitempty"`
}

// 详细资源使用统计
type ResourceUsageDetail struct {
	// setrlimit统计数据
	SetrlimitStats struct {
		CPUTimeUsed  int64 // CPU时间使用(毫秒)
		MaxRSSUsed   int64 // 最大常驻内存(KB)
		WallTimeUsed int64 // 墙钟时间(毫秒)
	} `json:"setrlimit_stats"`

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
	} `json:"cgroups_stats"`

	// 综合判断结果
	LimitExceeded   string `json:"limit_exceeded"` // 超限类型："memory", "cpu", "time", "none"
	ControlMethod   string `json:"control_method"` // 控制方式："setrlimit", "cgroups", "hybrid"
	PerformanceData struct {
		SetupTimeMs   int64 // 资源控制设置耗时(毫秒)
		MonitorTimeMs int64 // 监控耗时(毫秒)
		CleanupTimeMs int64 // 清理耗时(毫秒)
	} `json:"performance_data"`
}

// 系统调用沙箱
type SystemCallSandbox struct {
	config        *SandboxConfig
	cgroupManager *CgroupManager // cgroup管理器
}

// 创建新的沙箱
func NewSystemCallSandbox(config *SandboxConfig) *SystemCallSandbox {
	sandbox := &SystemCallSandbox{
		config: config,
	}

	// 如果启用cgroups，初始化cgroup管理器
	if config.EnableCgroups {
		cgroupConfig := sandbox.buildCgroupConfig()
		sandbox.cgroupManager = NewCgroupManager(cgroupConfig)
	}

	return sandbox
}

// 构建cgroup配置
func (s *SystemCallSandbox) buildCgroupConfig() *CgroupConfig {
	// 生成唯一的cgroup组名
	groupName := fmt.Sprintf("judge_%s_%s_%d", s.config.Language, s.config.TaskID, time.Now().UnixNano())

	config := &CgroupConfig{
		GroupName: groupName,
		Language:  s.config.Language,
		TaskID:    s.config.TaskID,

		// 内存配置
		MemoryLimitBytes: s.config.MemoryLimit * 1024, // KB转字节

		// CPU配置
		CPUPeriodUs: 100000, // 100ms周期

		// 进程数配置
		PIDsMax: int64(s.config.ProcessLimit),

		// I/O配置
		BlkIOWeight: 500, // 默认权重
	}

	// 设置内存+swap限制
	if s.config.MemorySwapRatio > 0 {
		config.MemorySwapLimit = int64(float64(config.MemoryLimitBytes) * s.config.MemorySwapRatio)
	} else {
		// 默认禁用swap（设置为与内存限制相同）
		config.MemorySwapLimit = config.MemoryLimitBytes
	}

	// 设置CPU配额
	if s.config.CPUQuotaPercent > 0 && s.config.CPUQuotaPercent <= 100 {
		// CPU配额 = 周期 * 百分比 / 100
		config.CPUQuotaUs = config.CPUPeriodUs * int64(s.config.CPUQuotaPercent) / 100
	}

	// 设置CPU核心绑定
	if s.config.CPUSetCores != "" {
		config.CPUSetCPUs = s.config.CPUSetCores
	}

	// 设置I/O权重
	if s.config.IOWeightPercent > 0 {
		// I/O权重范围：10-1000
		weight := int64(s.config.IOWeightPercent * 10)
		if weight < 10 {
			weight = 10
		} else if weight > 1000 {
			weight = 1000
		}
		config.BlkIOWeight = weight
	}

	logx.Infof("Built cgroup config: group=%s, memory=%dMB, cpu_quota=%d%%",
		groupName, config.MemoryLimitBytes/1024/1024, s.config.CPUQuotaPercent)

	return config
}

// 执行程序
func (s *SystemCallSandbox) Execute(ctx context.Context, executable string, args []string) (*ExecuteResult, error) {
	logx.Infof("Starting execution: %s %v", executable, args)

	// 创建命令
	cmd := exec.CommandContext(ctx, executable, args...)

	// 设置进程属性
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// 创建新的命名空间 - 构建Cloneflags
		Cloneflags: s.buildCloneFlags(),
		// 设置用户和组
		Credential: &syscall.Credential{
			Uid: uint32(s.config.UID),
			Gid: uint32(s.config.GID),
		},
		// 设置chroot
		Chroot: s.config.Chroot,
	}

	// 处理seccomp过滤器初始化
	finalExecutable := executable
	finalArgs := args

	if s.config.EnableSeccomp {
		// 创建seccomp初始化程序
		seccompInit, err := s.createSeccompInitializer(executable, args, s.config.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create seccomp initializer: %w", err)
		}
		finalExecutable = seccompInit
		finalArgs = []string{} // seccomp初始化程序不需要额外参数

		// 延迟清理seccomp相关文件
		defer s.cleanupSeccompFiles(s.config.WorkDir)
	}

	// 如果需要设置UTS/IPC/Cgroup Namespace，需要在子进程中执行初始化
	if s.config.EnableUTSNS || s.config.EnableIPCNS || s.config.EnableCgroupNS {
		// 创建一个包装脚本来处理Namespace初始化
		wrapperScript, err := s.createNamespaceWrapper(finalExecutable, finalArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to create namespace wrapper: %w", err)
		}
		defer os.Remove(wrapperScript)

		// 使用包装脚本替换命令
		cmd = exec.CommandContext(ctx, "/bin/sh", wrapperScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: s.buildCloneFlags(),
			Credential: &syscall.Credential{
				Uid: uint32(s.config.UID),
				Gid: uint32(s.config.GID),
			},
			Chroot: s.config.Chroot,
		}
	} else if s.config.EnableSeccomp {
		// 只需要seccomp，直接使用初始化程序
		cmd = exec.CommandContext(ctx, finalExecutable, finalArgs...)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: s.buildCloneFlags(),
			Credential: &syscall.Credential{
				Uid: uint32(s.config.UID),
				Gid: uint32(s.config.GID),
			},
			Chroot: s.config.Chroot,
		}
	}

	// 设置工作目录
	cmd.Dir = s.config.WorkDir

	// 设置环境变量
	cmd.Env = s.config.Environment

	// 设置输入输出重定向
	if err := s.setupIO(cmd); err != nil {
		return nil, fmt.Errorf("failed to setup IO: %w", err)
	}

	// 设置双层资源限制：setrlimit + cgroups
	setupStart := time.Now()
	if err := s.setupResourceLimits(); err != nil {
		return nil, fmt.Errorf("failed to setup resource limits: %w", err)
	}
	setupTime := time.Since(setupStart).Milliseconds()

	// 启动进程
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	logx.Infof("Process started with PID: %d", cmd.Process.Pid)

	// 设置User Namespace的uid/gid映射（必须在进程启动后立即设置）
	if err := s.setupUserNamespaceMapping(cmd.Process.Pid); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to setup user namespace mapping: %w", err)
	}

	// 将进程添加到cgroup（如果启用）
	if s.config.EnableCgroups && s.cgroupManager != nil {
		if err := s.cgroupManager.AddProcess(cmd.Process.Pid); err != nil {
			cmd.Process.Kill()
			return nil, fmt.Errorf("failed to add process to cgroup: %w", err)
		}
		logx.Infof("Process %d added to cgroup: %s", cmd.Process.Pid, s.cgroupManager.config.GroupName)
	}

	// 记录Namespace信息用于调试
	nsInfo := s.getNamespaceInfo(cmd.Process.Pid)
	logx.Infof("Process Namespace info: %+v", nsInfo)

	// 监控进程执行
	monitorStart := time.Now()
	result, err := s.monitorProcessWithCgroups(cmd.Process.Pid, startTime)
	if err != nil {
		// 确保进程被终止
		cmd.Process.Kill()
		// 清理cgroup
		s.cleanupCgroup()
		return nil, fmt.Errorf("failed to monitor process: %w", err)
	}
	monitorTime := time.Since(monitorStart).Milliseconds()

	// 等待进程结束
	cmd.Wait()

	// 收集详细的资源使用统计
	cleanupStart := time.Now()
	if err := s.collectResourceStats(result, cmd.Process.Pid); err != nil {
		logx.Errorf("Failed to collect resource stats: %v", err)
	}

	// 清理cgroup资源
	s.cleanupCgroup()
	cleanupTime := time.Since(cleanupStart).Milliseconds()

	// 设置性能数据
	if result.ResourceUsage != nil {
		result.ResourceUsage.PerformanceData.SetupTimeMs = setupTime
		result.ResourceUsage.PerformanceData.MonitorTimeMs = monitorTime
		result.ResourceUsage.PerformanceData.CleanupTimeMs = cleanupTime
	}

	logx.Infof("Process execution completed: status=%d, time=%dms, memory=%dKB, method=%s",
		result.Status, result.TimeUsed, result.MemoryUsed, s.getControlMethod())

	return result, nil
}

// 设置输入输出重定向
func (s *SystemCallSandbox) setupIO(cmd *exec.Cmd) error {
	// 设置输入文件
	if s.config.InputFile != "" {
		inputFile, err := os.Open(s.config.InputFile)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		cmd.Stdin = inputFile
	}

	// 设置输出文件
	if s.config.OutputFile != "" {
		outputFile, err := os.Create(s.config.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		cmd.Stdout = outputFile
	}

	// 设置错误输出文件
	if s.config.ErrorFile != "" {
		errorFile, err := os.Create(s.config.ErrorFile)
		if err != nil {
			return fmt.Errorf("failed to create error file: %w", err)
		}
		cmd.Stderr = errorFile
	}

	return nil
}

// 设置双层资源限制：setrlimit + cgroups
func (s *SystemCallSandbox) setupResourceLimits() error {
	logx.Infof("Setting up resource limits: mode=%s, cgroups=%v", s.config.CgroupsMode, s.config.EnableCgroups)

	// 1. 优先设置cgroups（如果启用且可用）
	if s.config.EnableCgroups && s.cgroupManager != nil {
		if err := s.setupCgroups(); err != nil {
			if s.config.CgroupsMode == "primary" {
				return fmt.Errorf("cgroups setup failed in primary mode: %w", err)
			}
			logx.Errorf("cgroups setup failed, falling back to setrlimit: %v", err)
		} else {
			logx.Info("cgroups resource control enabled")
		}
	}

	// 2. 设置setrlimit（作为基础保护或降级方案）
	if err := s.setupSetrlimit(); err != nil {
		if !s.config.EnableCgroups || s.cgroupManager == nil {
			return fmt.Errorf("setrlimit setup failed: %w", err)
		}
		logx.Errorf("setrlimit setup failed, relying on cgroups: %v", err)
	} else {
		logx.Info("setrlimit resource control enabled")
	}

	return nil
}

// 设置cgroups资源控制
func (s *SystemCallSandbox) setupCgroups() error {
	if s.cgroupManager == nil {
		return fmt.Errorf("cgroup manager not initialized")
	}

	// 创建cgroup控制组
	if err := s.cgroupManager.Create(); err != nil {
		return fmt.Errorf("failed to create cgroup: %w", err)
	}

	logx.Infof("Created cgroup: %s", s.cgroupManager.config.GroupName)
	return nil
}

// 设置setrlimit基础资源限制
func (s *SystemCallSandbox) setupSetrlimit() error {
	// CPU时间限制（配合cgroups时设置为宽松值）
	if s.config.TimeLimit > 0 {
		timeLimit := s.config.TimeLimit / 1000 // 转换为秒
		if s.config.EnableCgroups {
			// cgroups模式下，setrlimit设置为2倍作为兜底保护
			timeLimit *= 2
		}
		if err := syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{
			Cur: uint64(timeLimit),
			Max: uint64(timeLimit),
		}); err != nil {
			return fmt.Errorf("failed to set CPU time limit: %w", err)
		}
		logx.Debugf("Set CPU time limit: %d seconds", timeLimit)
	}

	// 内存限制（配合cgroups时设置为虚拟内存保护）
	if s.config.MemoryLimit > 0 {
		memoryLimit := s.config.MemoryLimit * 1024 // 转换为字节
		if s.config.EnableCgroups {
			// cgroups模式下，setrlimit设置为2倍防止虚拟内存爆炸
			memoryLimit *= 2
		}
		if err := syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
			Cur: uint64(memoryLimit),
			Max: uint64(memoryLimit),
		}); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
		logx.Debugf("Set virtual memory limit: %d bytes", memoryLimit)
	}

	// 栈大小限制（cgroups无法控制，setrlimit主控）
	if s.config.StackLimit > 0 {
		stackLimit := s.config.StackLimit * 1024 // 转换为字节
		if err := syscall.Setrlimit(syscall.RLIMIT_STACK, &syscall.Rlimit{
			Cur: uint64(stackLimit),
			Max: uint64(stackLimit),
		}); err != nil {
			return fmt.Errorf("failed to set stack limit: %w", err)
		}
		logx.Debugf("Set stack limit: %d bytes", stackLimit)
	}

	// 文件大小限制（cgroups无法控制，setrlimit主控）
	if s.config.FileSizeLimit > 0 {
		fileSizeLimit := s.config.FileSizeLimit * 1024 // 转换为字节
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{
			Cur: uint64(fileSizeLimit),
			Max: uint64(fileSizeLimit),
		}); err != nil {
			return fmt.Errorf("failed to set file size limit: %w", err)
		}
		logx.Debugf("Set file size limit: %d bytes", fileSizeLimit)
	}

	// 进程数限制（与cgroups配合）
	// 注意：某些系统可能不支持RLIMIT_NPROC，这里简化处理
	if s.config.ProcessLimit > 0 {
		logx.Debugf("Process limit configured: %d (handled by cgroups)", s.config.ProcessLimit)
	}

	// 禁用核心转储
	if err := syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{
		Cur: 0,
		Max: 0,
	}); err != nil {
		logx.Errorf("Failed to disable core dump: %v", err)
	}

	return nil
}

// 带cgroups支持的进程监控
func (s *SystemCallSandbox) monitorProcessWithCgroups(pid int, startTime time.Time) (*ExecuteResult, error) {
	result := &ExecuteResult{
		ResourceUsage: &ResourceUsageDetail{},
	}

	// 初始化控制方法标识
	result.ResourceUsage.ControlMethod = s.getControlMethod()

	// 使用ptrace附加到进程
	if err := syscall.PtraceAttach(pid); err != nil {
		return nil, fmt.Errorf("failed to attach ptrace: %w", err)
	}
	defer syscall.PtraceDetach(pid)

	var status syscall.WaitStatus
	var rusage syscall.Rusage

	for {
		// 等待进程状态变化
		_, err := syscall.Wait4(pid, &status, 0, &rusage)
		if err != nil {
			if err == syscall.ECHILD {
				// 进程已经结束
				break
			}
			return nil, fmt.Errorf("failed to wait for process: %w", err)
		}

		// 检查墙钟时间限制
		elapsed := time.Since(startTime)
		if s.config.WallTimeLimit > 0 && elapsed > time.Duration(s.config.WallTimeLimit)*time.Millisecond {
			syscall.Kill(pid, syscall.SIGKILL)
			result.Status = StatusTimeLimitExceeded
			result.TimeUsed = s.config.WallTimeLimit
			result.ResourceUsage.LimitExceeded = "time"
			break
		}

		// 检查cgroup资源限制（如果启用）
		if s.config.EnableCgroups && s.cgroupManager != nil {
			if exceeded, limitType := s.checkCgroupLimits(); exceeded {
				syscall.Kill(pid, syscall.SIGKILL)
				result.ResourceUsage.LimitExceeded = limitType
				switch limitType {
				case "memory":
					result.Status = StatusMemoryLimitExceeded
				case "cpu":
					result.Status = StatusTimeLimitExceeded
				default:
					result.Status = StatusRuntimeError
				}
				break
			}
		}

		// 检查setrlimit内存限制（兜底保护）
		if s.config.MemoryLimit > 0 && rusage.Maxrss > s.config.MemoryLimit {
			syscall.Kill(pid, syscall.SIGKILL)
			result.Status = StatusMemoryLimitExceeded
			result.MemoryUsed = rusage.Maxrss
			result.ResourceUsage.LimitExceeded = "memory"
			break
		}

		// 进程正常结束
		if status.Exited() {
			result.Status = StatusAccepted
			result.ExitCode = status.ExitStatus()
			result.ResourceUsage.LimitExceeded = "none"
			break
		}

		// 进程被信号终止
		if status.Signaled() {
			signal := status.Signal()
			result.Signal = int(signal)

			switch signal {
			case syscall.SIGXCPU:
				result.Status = StatusTimeLimitExceeded
				result.ResourceUsage.LimitExceeded = "cpu"
			case syscall.SIGKILL:
				if rusage.Maxrss > s.config.MemoryLimit {
					result.Status = StatusMemoryLimitExceeded
					result.ResourceUsage.LimitExceeded = "memory"
				} else {
					result.Status = StatusTimeLimitExceeded
					result.ResourceUsage.LimitExceeded = "time"
				}
			case syscall.SIGSEGV, syscall.SIGFPE, syscall.SIGABRT:
				result.Status = StatusRuntimeError
				result.ResourceUsage.LimitExceeded = "none"
			default:
				result.Status = StatusRuntimeError
				result.ResourceUsage.LimitExceeded = "none"
			}
			break
		}

		// 进程停止（被ptrace）
		if status.Stopped() {
			// 继续执行进程
			if err := syscall.PtraceCont(pid, 0); err != nil {
				return nil, fmt.Errorf("failed to continue process: %w", err)
			}
		}
	}

	// 记录setrlimit资源使用情况
	result.ResourceUsage.SetrlimitStats.CPUTimeUsed = int64(rusage.Utime.Sec*1000 + rusage.Utime.Usec/1000)
	result.ResourceUsage.SetrlimitStats.MaxRSSUsed = rusage.Maxrss
	result.ResourceUsage.SetrlimitStats.WallTimeUsed = time.Since(startTime).Milliseconds()

	// 设置兼容性字段
	result.TimeUsed = result.ResourceUsage.SetrlimitStats.CPUTimeUsed
	result.MemoryUsed = result.ResourceUsage.SetrlimitStats.MaxRSSUsed

	// 检查输出文件大小
	if s.config.OutputFile != "" {
		if stat, err := os.Stat(s.config.OutputFile); err == nil {
			result.OutputSize = stat.Size()
			// 检查输出大小限制
			if result.OutputSize > int64(s.config.FileSizeLimit)*1024 {
				result.Status = StatusOutputLimitExceeded
				result.ResourceUsage.LimitExceeded = "output"
			}
		}
	}

	// 读取错误输出
	if s.config.ErrorFile != "" {
		if errorData, err := os.ReadFile(s.config.ErrorFile); err == nil {
			result.ErrorOutput = string(errorData)
			logx.Infof("Debug: Read error file %s, length=%d, content='%s'", s.config.ErrorFile, len(result.ErrorOutput), result.ErrorOutput)
		} else {
			logx.Errorf("Debug: Failed to read error file %s: %v", s.config.ErrorFile, err)
		}
	} else {
		logx.Infof("Debug: ErrorFile not configured")
	}

	return result, nil
}

// 检查cgroup资源限制
func (s *SystemCallSandbox) checkCgroupLimits() (bool, string) {
	if s.cgroupManager == nil {
		return false, ""
	}

	stats, err := s.cgroupManager.GetStats()
	if err != nil {
		logx.Errorf("Failed to get cgroup stats: %v", err)
		return false, ""
	}

	// 检查内存限制
	if stats.MemoryLimit > 0 && stats.MemoryUsage >= stats.MemoryLimit {
		logx.Infof("Memory limit exceeded: used=%d, limit=%d", stats.MemoryUsage, stats.MemoryLimit)
		return true, "memory"
	}

	// 检查OOM事件
	if stats.MemoryOOMCount > 0 {
		logx.Infof("OOM event detected: count=%d", stats.MemoryOOMCount)
		return true, "memory"
	}

	// 检查进程数限制
	if stats.PIDsMax > 0 && stats.PIDsCurrent >= stats.PIDsMax {
		logx.Infof("PIDs limit exceeded: current=%d, max=%d", stats.PIDsCurrent, stats.PIDsMax)
		return true, "pids"
	}

	// 检查CPU限流
	if stats.CPUThrottled > 10 { // 允许少量限流
		logx.Infof("CPU heavily throttled: count=%d", stats.CPUThrottled)
		return true, "cpu"
	}

	return false, ""
}

// 收集详细的资源使用统计
func (s *SystemCallSandbox) collectResourceStats(result *ExecuteResult, pid int) error {
	if result.ResourceUsage == nil {
		result.ResourceUsage = &ResourceUsageDetail{}
	}

	// 收集cgroup统计信息
	if s.config.EnableCgroups && s.cgroupManager != nil {
		stats, err := s.cgroupManager.GetStats()
		if err != nil {
			logx.Errorf("Failed to get final cgroup stats: %v", err)
		} else {
			result.ResourceUsage.CgroupsStats.MemoryUsed = stats.MemoryUsage
			result.ResourceUsage.CgroupsStats.MemoryPeakUsed = stats.MemoryMaxUsage
			result.ResourceUsage.CgroupsStats.MemoryLimit = stats.MemoryLimit
			result.ResourceUsage.CgroupsStats.CPUUsageTotal = stats.CPUUsageTotal
			result.ResourceUsage.CgroupsStats.CPUUsagePercent = stats.CPUUsagePercent
			result.ResourceUsage.CgroupsStats.CPUThrottled = stats.CPUThrottled
			result.ResourceUsage.CgroupsStats.PIDsUsed = stats.PIDsCurrent
			result.ResourceUsage.CgroupsStats.IOReadBytes = stats.BlkIOReadBytes
			result.ResourceUsage.CgroupsStats.IOWriteBytes = stats.BlkIOWriteBytes

			// 如果cgroups数据更准确，优先使用
			if stats.MemoryMaxUsage > 0 {
				result.MemoryUsed = stats.MemoryMaxUsage / 1024 // 转换为KB
			}
		}
	}

	return nil
}

// 清理cgroup资源
func (s *SystemCallSandbox) cleanupCgroup() {
	if s.config.EnableCgroups && s.cgroupManager != nil {
		if err := s.cgroupManager.Cleanup(); err != nil {
			logx.Errorf("Failed to cleanup cgroup: %v", err)
		} else {
			logx.Infof("Cleaned up cgroup: %s", s.cgroupManager.config.GroupName)
		}
	}
}

// 获取当前使用的资源控制方法
func (s *SystemCallSandbox) getControlMethod() string {
	if s.config.EnableCgroups && s.cgroupManager != nil {
		switch s.config.CgroupsMode {
		case "primary":
			return "cgroups"
		case "fallback":
			return "setrlimit"
		default:
			return "hybrid"
		}
	}
	return "setrlimit"
}

// 原有的监控进程函数（保持兼容性）
func (s *SystemCallSandbox) monitorProcess(pid int, startTime time.Time) (*ExecuteResult, error) {
	result := &ExecuteResult{}

	// 使用ptrace附加到进程
	if err := syscall.PtraceAttach(pid); err != nil {
		return nil, fmt.Errorf("failed to attach ptrace: %w", err)
	}
	defer syscall.PtraceDetach(pid)

	var status syscall.WaitStatus
	var rusage syscall.Rusage

	for {
		// 等待进程状态变化
		_, err := syscall.Wait4(pid, &status, 0, &rusage)
		if err != nil {
			if err == syscall.ECHILD {
				// 进程已经结束
				break
			}
			return nil, fmt.Errorf("failed to wait for process: %w", err)
		}

		// 检查墙钟时间限制
		elapsed := time.Since(startTime)
		if s.config.WallTimeLimit > 0 && elapsed > time.Duration(s.config.WallTimeLimit)*time.Millisecond {
			syscall.Kill(pid, syscall.SIGKILL)
			result.Status = StatusTimeLimitExceeded
			result.TimeUsed = s.config.WallTimeLimit
			break
		}

		// 检查内存使用
		if s.config.MemoryLimit > 0 && rusage.Maxrss > s.config.MemoryLimit {
			syscall.Kill(pid, syscall.SIGKILL)
			result.Status = StatusMemoryLimitExceeded
			result.MemoryUsed = rusage.Maxrss
			break
		}

		// 进程正常结束
		if status.Exited() {
			result.Status = StatusAccepted
			result.ExitCode = status.ExitStatus()
			break
		}

		// 进程被信号终止
		if status.Signaled() {
			signal := status.Signal()
			result.Signal = int(signal)

			switch signal {
			case syscall.SIGXCPU:
				result.Status = StatusTimeLimitExceeded
			case syscall.SIGKILL:
				if rusage.Maxrss > s.config.MemoryLimit {
					result.Status = StatusMemoryLimitExceeded
				} else {
					result.Status = StatusTimeLimitExceeded
				}
			case syscall.SIGSEGV, syscall.SIGFPE, syscall.SIGABRT:
				result.Status = StatusRuntimeError
			default:
				result.Status = StatusRuntimeError
			}
			break
		}

		// 进程停止（被ptrace）
		if status.Stopped() {
			// 继续执行进程
			if err := syscall.PtraceCont(pid, 0); err != nil {
				return nil, fmt.Errorf("failed to continue process: %w", err)
			}
		}
	}

	// 记录资源使用情况
	result.TimeUsed = int64(rusage.Utime.Sec*1000 + rusage.Utime.Usec/1000)
	result.MemoryUsed = rusage.Maxrss

	// 检查输出文件大小
	if s.config.OutputFile != "" {
		if stat, err := os.Stat(s.config.OutputFile); err == nil {
			result.OutputSize = stat.Size()
			// 检查输出大小限制
			if result.OutputSize > int64(s.config.FileSizeLimit)*1024 {
				result.Status = StatusOutputLimitExceeded
			}
		}
	}

	// 读取错误输出
	if s.config.ErrorFile != "" {
		if errorData, err := os.ReadFile(s.config.ErrorFile); err == nil {
			result.ErrorOutput = string(errorData)
			logx.Infof("Debug: Read error file %s, length=%d, content='%s'", s.config.ErrorFile, len(result.ErrorOutput), result.ErrorOutput)
		} else {
			logx.Errorf("Debug: Failed to read error file %s: %v", s.config.ErrorFile, err)
		}
	} else {
		logx.Infof("Debug: ErrorFile not configured")
	}

	return result, nil
}

// 创建安全的工作目录
func (s *SystemCallSandbox) CreateWorkDir(baseDir string) (string, error) {
	// 创建临时工作目录
	workDir := filepath.Join(baseDir, fmt.Sprintf("judge_%d", time.Now().UnixNano()))

	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create work directory: %w", err)
	}

	// 设置目录权限
	if err := os.Chown(workDir, s.config.UID, s.config.GID); err != nil {
		return "", fmt.Errorf("failed to chown work directory: %w", err)
	}

	return workDir, nil
}

// 清理工作目录
func (s *SystemCallSandbox) CleanupWorkDir(workDir string) error {
	return os.RemoveAll(workDir)
}

// 设置seccomp-bpf系统调用过滤器
// 原理：通过BPF程序自定义过滤规则，进而根据过滤规则判断是否允许进程执行该系统调用
// seccomp-bpf是Linux内核提供的一种系统调用过滤机制，能够控制进程可执行的系统调用
func (s *SystemCallSandbox) setupSeccomp() error {
	if !s.config.EnableSeccomp {
		logx.Info("Seccomp filtering is disabled")
		return nil
	}

	if len(s.config.AllowedSyscalls) == 0 {
		logx.Error("No allowed syscalls configured, using strict mode")
		// 如果没有配置允许的系统调用，使用严格模式
		filter, err := CreateStrictSeccompFilter()
		if err != nil {
			return fmt.Errorf("failed to create strict seccomp filter: %w", err)
		}

		// 验证过滤器
		if err := filter.Validate(); err != nil {
			return fmt.Errorf("seccomp filter validation failed: %w", err)
		}

		// 安装过滤器
		if err := filter.Install(); err != nil {
			return fmt.Errorf("failed to install seccomp filter: %w", err)
		}

		logx.Info("Strict seccomp filter installed successfully")
		return nil
	}

	logx.Infof("Setting up seccomp filter with %d allowed syscalls", len(s.config.AllowedSyscalls))

	// 1. 创建seccomp过滤器
	filter := NewSeccompFilter(s.config.AllowedSyscalls, SECCOMP_RET_KILL_PROCESS)

	// 2. 验证过滤器配置
	if err := filter.Validate(); err != nil {
		return fmt.Errorf("seccomp filter validation failed: %w", err)
	}

	// 3. 输出BPF程序反汇编（用于调试）
	// 注意：这里简化处理，实际应该检查日志级别
	if true { // 临时启用调试输出
		// 先构建BPF程序以获取反汇编
		if err := filter.buildBPFProgram(); err != nil {
			logx.Errorf("Failed to build BPF program for debugging: %v", err)
		} else {
			disasm := filter.GetBPFDisassembly()
			logx.Debug("BPF Program Disassembly:")
			for _, line := range disasm {
				logx.Debug(line)
			}
		}
	}

	// 4. 安装seccomp过滤器
	if err := filter.Install(); err != nil {
		return fmt.Errorf("failed to install seccomp filter: %w", err)
	}

	logx.Info("Seccomp-bpf filter installed successfully")
	return nil
}

// 获取系统调用白名单
func GetSyscallWhitelist(language string) []int {
	switch language {
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
	case "java":
		return []int{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 16, 21, 22, 25, 39,
			56, 57, 59, 60, 61, 62, 63, 89, 96, 97, 158, 202, 231, 257, 273, 318,
		}
	case "python":
		return []int{
			0, 1, 2, 3, 4, 5, 6, 8, 9, 10, 11, 12, 13, 16, 21, 22, 39, 59, 60, 61,
			79, 89, 97, 158, 231, 257, 273, 318,
		}
	case "go":
		return []int{
			0, 1, 2, 3, 4, 5, 8, 9, 10, 11, 12, 13, 16, 21, 22, 39, 56, 57, 59, 60,
			61, 62, 89, 96, 97, 158, 202, 231, 257, 273, 318,
		}
	case "javascript":
		return []int{
			0, 1, 2, 3, 4, 5, 6, 8, 9, 10, 11, 12, 13, 16, 21, 22, 39, 59, 60, 61,
			89, 97, 158, 231, 257, 273, 318,
		}
	default:
		// 返回最小权限集合
		return []int{0, 1, 2, 3, 59, 60, 231}
	}
}

// 验证程序路径安全性
func (s *SystemCallSandbox) ValidatePath(path string) error {
	// 检查路径是否在允许的范围内
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 检查路径是否在工作目录内
	workDirAbs, err := filepath.Abs(s.config.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get work directory absolute path: %w", err)
	}

	relPath, err := filepath.Rel(workDirAbs, absPath)
	if err != nil || len(relPath) > 0 && relPath[0] == '.' && relPath[1] == '.' {
		return fmt.Errorf("path outside work directory: %s", path)
	}

	return nil
}

// 构建Clone标志位 - 根据配置动态构建需要的Namespace隔离标志
func (s *SystemCallSandbox) buildCloneFlags() uintptr {
	// 基础的三种Namespace（已实现）
	flags := syscall.CLONE_NEWPID | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS

	// User Namespace隔离
	// 原理：创建新的用户命名空间，实现uid/gid映射，将沙箱内的root权限映射到主机普通用户
	// 作用：防止用户代码获取真实的主机特权，即使在沙箱内获得root也无法影响主机
	if s.config.EnableUserNS {
		flags |= syscall.CLONE_NEWUSER
		logx.Info("Enabled User Namespace isolation - uid/gid mapping will be configured")
	}

	// UTS Namespace隔离
	// 原理：为进程创建独立的主机名和域名空间，进程内修改主机名不影响主机系统
	// 作用：防止用户代码通过修改主机名干扰系统标识或通过主机名判断系统环境
	if s.config.EnableUTSNS {
		flags |= syscall.CLONE_NEWUTS
		logx.Info("Enabled UTS Namespace isolation - hostname/domainname will be isolated")
	}

	// IPC Namespace隔离
	// 原理：创建独立的进程间通信资源池（消息队列、共享内存、信号量）
	// 作用：防止用户代码通过共享内存读取主机进程敏感数据或通过消息队列与其他进程通信
	if s.config.EnableIPCNS {
		flags |= syscall.CLONE_NEWIPC
		logx.Info("Enabled IPC Namespace isolation - IPC resources will be isolated")
	}

	// Cgroup Namespace隔离
	// 原理：创建独立的cgroup视图，进程只能看到自身关联的cgroup子树
	// 作用：防止用户代码修改cgroup配置，限制对控制组的访问权限
	if s.config.EnableCgroupNS {
		flags |= syscall.CLONE_NEWCGROUP
		logx.Info("Enabled Cgroup Namespace isolation - cgroup view will be restricted")
	}

	return uintptr(flags)
}

// 设置User Namespace的uid/gid映射
// 原理：通过写入/proc/PID/uid_map和/proc/PID/gid_map文件实现映射关系
// 映射格式：inside_id outside_id length（沙箱内ID 主机ID 映射长度）
func (s *SystemCallSandbox) setupUserNamespaceMapping(pid int) error {
	if !s.config.EnableUserNS {
		return nil
	}

	logx.Infof("Setting up User Namespace mapping for PID: %d", pid)

	// 设置uid映射 - 将沙箱内的用户ID映射到主机的普通用户ID
	// 例如：沙箱内root(0) -> 主机普通用户(1000)，实现权限降级
	uidMapPath := fmt.Sprintf("/proc/%d/uid_map", pid)
	uidMapping := fmt.Sprintf("%d %d 1", s.config.UidMapInside, s.config.UidMapOutside)
	if err := s.writeToFile(uidMapPath, uidMapping); err != nil {
		return fmt.Errorf("failed to setup uid mapping: %w", err)
	}

	// 禁用setgroups - 在设置gid映射前必须禁用setgroups系统调用
	// 原理：防止通过setgroups绕过gid映射限制获取额外的组权限
	setgroupsPath := fmt.Sprintf("/proc/%d/setgroups", pid)
	if err := s.writeToFile(setgroupsPath, "deny"); err != nil {
		logx.Errorf("Failed to deny setgroups (may not be critical): %v", err)
	}

	// 设置gid映射 - 将沙箱内的组ID映射到主机的普通组ID
	gidMapPath := fmt.Sprintf("/proc/%d/gid_map", pid)
	gidMapping := fmt.Sprintf("%d %d 1", s.config.GidMapInside, s.config.GidMapOutside)
	if err := s.writeToFile(gidMapPath, gidMapping); err != nil {
		return fmt.Errorf("failed to setup gid mapping: %w", err)
	}

	logx.Info("User Namespace uid/gid mapping configured successfully")
	return nil
}

// 设置UTS Namespace的主机名和域名
// 原理：在新的UTS命名空间中，可以独立设置主机名和域名而不影响主机系统
func (s *SystemCallSandbox) setupUTSNamespace() error {
	if !s.config.EnableUTSNS {
		return nil
	}

	logx.Info("Setting up UTS Namespace hostname/domainname")

	// 设置沙箱内的主机名 - 通过sethostname系统调用
	// 作用：为沙箱提供独立的主机标识，防止用户代码通过主机名判断真实环境
	if s.config.Hostname != "" {
		hostname := []byte(s.config.Hostname)
		if err := syscall.Sethostname(hostname); err != nil {
			return fmt.Errorf("failed to set hostname: %w", err)
		}
		logx.Infof("Set sandbox hostname to: %s", s.config.Hostname)
	}

	// 设置沙箱内的域名 - 通过setdomainname系统调用
	// 作用：提供完整的网络标识隔离，防止域名信息泄露
	if s.config.DomainName != "" {
		domainname := []byte(s.config.DomainName)
		if err := syscall.Setdomainname(domainname); err != nil {
			return fmt.Errorf("failed to set domainname: %w", err)
		}
		logx.Infof("Set sandbox domainname to: %s", s.config.DomainName)
	}

	return nil
}

// 验证IPC Namespace隔离效果
// 原理：检查当前进程的IPC资源是否与主机隔离
func (s *SystemCallSandbox) validateIPCNamespace() error {
	if !s.config.EnableIPCNS {
		return nil
	}

	logx.Info("Validating IPC Namespace isolation")

	// 检查消息队列 - 读取/proc/sysvipc/msg获取当前可见的消息队列
	// 在独立的IPC命名空间中，应该看不到主机的IPC资源
	msgPath := "/proc/sysvipc/msg"
	if content, err := os.ReadFile(msgPath); err == nil {
		lines := len(string(content))
		logx.Infof("IPC message queues visible: %d lines", lines)
	}

	// 检查共享内存 - 读取/proc/sysvipc/shm获取当前可见的共享内存段
	shmPath := "/proc/sysvipc/shm"
	if content, err := os.ReadFile(shmPath); err == nil {
		lines := len(string(content))
		logx.Infof("IPC shared memory segments visible: %d lines", lines)
	}

	// 检查信号量 - 读取/proc/sysvipc/sem获取当前可见的信号量集
	semPath := "/proc/sysvipc/sem"
	if content, err := os.ReadFile(semPath); err == nil {
		lines := len(string(content))
		logx.Infof("IPC semaphore sets visible: %d lines", lines)
	}

	logx.Info("IPC Namespace isolation validated")
	return nil
}

// 验证Cgroup Namespace隔离效果
// 原理：检查当前进程的cgroup视图是否被正确限制
func (s *SystemCallSandbox) validateCgroupNamespace() error {
	if !s.config.EnableCgroupNS {
		return nil
	}

	logx.Info("Validating Cgroup Namespace isolation")

	// 检查cgroup根目录 - 在独立的Cgroup命名空间中，/sys/fs/cgroup应该只显示受限的子树
	cgroupRoot := "/sys/fs/cgroup"
	if entries, err := os.ReadDir(cgroupRoot); err == nil {
		logx.Infof("Cgroup controllers visible: %d entries", len(entries))
		for _, entry := range entries {
			logx.Debugf("Visible cgroup controller: %s", entry.Name())
		}
	} else {
		logx.Errorf("Failed to read cgroup root: %v", err)
	}

	// 检查当前进程的cgroup信息
	cgroupPath := "/proc/self/cgroup"
	if content, err := os.ReadFile(cgroupPath); err == nil {
		logx.Infof("Current process cgroup info: %s", string(content))
	}

	logx.Info("Cgroup Namespace isolation validated")
	return nil
}

// 写入文件的辅助函数 - 用于设置Namespace映射文件
func (s *SystemCallSandbox) writeToFile(path, content string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// 创建Namespace包装脚本
// 原理：由于某些Namespace设置需要在子进程中执行，我们创建一个shell脚本
// 该脚本先进行Namespace初始化，再执行实际的用户程序
func (s *SystemCallSandbox) createNamespaceWrapper(executable string, args []string) (string, error) {
	// 创建临时脚本文件
	scriptPath := filepath.Join(s.config.WorkDir, "ns_wrapper.sh")
	script, err := os.Create(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to create wrapper script: %w", err)
	}
	defer script.Close()

	// 构建脚本内容
	var scriptContent strings.Builder
	scriptContent.WriteString("#!/bin/sh\n")
	scriptContent.WriteString("# Namespace initialization wrapper script\n\n")

	// UTS Namespace设置 - 必须在子进程中执行
	if s.config.EnableUTSNS {
		if s.config.Hostname != "" {
			scriptContent.WriteString("# Set hostname in UTS namespace\n")
			scriptContent.WriteString(fmt.Sprintf("hostname '%s' 2>/dev/null || echo 'Warning: failed to set hostname'\n", s.config.Hostname))
		}
		if s.config.DomainName != "" {
			scriptContent.WriteString("# Set domainname in UTS namespace\n")
			scriptContent.WriteString(fmt.Sprintf("domainname '%s' 2>/dev/null || echo 'Warning: failed to set domainname'\n", s.config.DomainName))
		}
	}

	// IPC Namespace验证
	if s.config.EnableIPCNS {
		scriptContent.WriteString("# Validate IPC namespace isolation\n")
		scriptContent.WriteString("echo 'IPC namespace info:' >&2\n")
		scriptContent.WriteString("ls -la /proc/sysvipc/ 2>/dev/null | wc -l >&2 || echo 'IPC validation skipped' >&2\n")
	}

	// Cgroup Namespace验证
	if s.config.EnableCgroupNS {
		scriptContent.WriteString("# Validate Cgroup namespace isolation\n")
		scriptContent.WriteString("echo 'Cgroup namespace info:' >&2\n")
		scriptContent.WriteString("ls -la /sys/fs/cgroup/ 2>/dev/null | wc -l >&2 || echo 'Cgroup validation skipped' >&2\n")
	}

	// 执行实际程序
	scriptContent.WriteString("\n# Execute the actual program\n")
	scriptContent.WriteString("exec ")
	scriptContent.WriteString(fmt.Sprintf("'%s'", executable))
	for _, arg := range args {
		scriptContent.WriteString(fmt.Sprintf(" '%s'", arg))
	}
	scriptContent.WriteString("\n")

	// 写入脚本内容
	if _, err := script.WriteString(scriptContent.String()); err != nil {
		return "", fmt.Errorf("failed to write wrapper script: %w", err)
	}

	// 设置脚本执行权限
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return "", fmt.Errorf("failed to set script permissions: %w", err)
	}

	logx.Infof("Created namespace wrapper script: %s", scriptPath)
	return scriptPath, nil
}

// 获取当前进程的Namespace信息 - 用于调试和验证
func (s *SystemCallSandbox) getNamespaceInfo(pid int) map[string]string {
	nsInfo := make(map[string]string)
	nsTypes := []string{"pid", "net", "mnt", "user", "uts", "ipc", "cgroup"}

	for _, nsType := range nsTypes {
		nsPath := fmt.Sprintf("/proc/%d/ns/%s", pid, nsType)
		if link, err := os.Readlink(nsPath); err == nil {
			nsInfo[nsType] = link
		} else {
			nsInfo[nsType] = "unavailable"
		}
	}

	return nsInfo
}

// 创建默认的沙箱配置 - 启用所有Namespace隔离策略
func NewDefaultSandboxConfig(workDir string) *SandboxConfig {
	return &SandboxConfig{
		// 基础配置 - 使用nobody用户运行，最小权限原则
		UID:     65534, // nobody用户
		GID:     65534, // nobody组
		WorkDir: workDir,

		// 资源限制 - 防止资源耗尽攻击
		TimeLimit:     5000,   // 5秒CPU时间限制
		WallTimeLimit: 10000,  // 10秒墙钟时间限制
		MemoryLimit:   131072, // 128MB内存限制
		StackLimit:    8192,   // 8MB栈限制
		FileSizeLimit: 10240,  // 10MB文件大小限制
		ProcessLimit:  10,     // 最多10个进程

		// 启用所有Namespace隔离策略
		EnableUserNS:   true, // 用户权限隔离
		EnableUTSNS:    true, // 主机名域名隔离
		EnableIPCNS:    true, // 进程间通信隔离
		EnableCgroupNS: true, // 控制组隔离

		// UTS Namespace配置
		Hostname:   "sandbox",     // 沙箱主机名
		DomainName: "judge.local", // 沙箱域名

		// User Namespace uid/gid映射配置
		UidMapInside:  0,     // 沙箱内使用root身份
		UidMapOutside: 65534, // 映射到主机nobody用户
		GidMapInside:  0,     // 沙箱内使用root组
		GidMapOutside: 65534, // 映射到主机nobody组

		// cgroups资源控制配置
		EnableCgroups:   true,      // 启用cgroups资源控制
		CgroupsMode:     "hybrid",  // 混合模式：setrlimit+cgroups
		TaskID:          "default", // 默认任务ID
		Language:        "unknown", // 默认语言
		CPUQuotaPercent: 50,        // 50% CPU配额
		MemorySwapRatio: 1.0,       // 禁用swap（swap=memory）
		IOWeightPercent: 50,        // 50% I/O权重

		// 系统调用控制
		EnableSeccomp: true,

		// 基础环境变量
		Environment: []string{
			"PATH=/usr/bin:/bin",
			"HOME=/tmp",
			"USER=sandbox",
			"SHELL=/bin/sh",
		},
	}
}

// 创建用于特定编程语言的沙箱配置
func NewLanguageSandboxConfig(language, workDir string) *SandboxConfig {
	config := NewDefaultSandboxConfig(workDir)

	// 根据编程语言调整配置
	switch language {
	case "java":
		// Java需要更多内存和时间
		config.MemoryLimit = 262144  // 256MB
		config.TimeLimit = 10000     // 10秒
		config.WallTimeLimit = 20000 // 20秒
		config.AllowedSyscalls = GetSyscallWhitelist("java")

	case "python":
		// Python解释器需要适中资源
		config.MemoryLimit = 131072 // 128MB
		config.TimeLimit = 8000     // 8秒
		config.AllowedSyscalls = GetSyscallWhitelist("python")

	case "cpp", "c":
		// C/C++编译后程序通常资源需求较小
		config.MemoryLimit = 65536 // 64MB
		config.TimeLimit = 3000    // 3秒
		config.AllowedSyscalls = GetSyscallWhitelist("cpp")

	case "go":
		// Go程序需要适中资源
		config.MemoryLimit = 131072 // 128MB
		config.TimeLimit = 5000     // 5秒
		config.AllowedSyscalls = GetSyscallWhitelist("go")

	case "javascript":
		// Node.js需要较多内存
		config.MemoryLimit = 196608 // 192MB
		config.TimeLimit = 8000     // 8秒
		config.AllowedSyscalls = GetSyscallWhitelist("javascript")

	default:
		// 未知语言使用最严格限制
		config.MemoryLimit = 65536 // 64MB
		config.TimeLimit = 2000    // 2秒
		config.AllowedSyscalls = GetSyscallWhitelist("")
	}

	// 确保seccomp过滤器启用并配置了系统调用白名单
	if len(config.AllowedSyscalls) > 0 {
		config.EnableSeccomp = true
		logx.Infof("Created %s sandbox config: memory=%dKB, time=%dms, seccomp enabled with %d syscalls",
			language, config.MemoryLimit, config.TimeLimit, len(config.AllowedSyscalls))
	} else {
		logx.Errorf("Created %s sandbox config without seccomp filtering", language)
	}

	return config
}

// 系统信息获取（用于调试和监控）
func GetSystemInfo() map[string]interface{} {
	var sysinfo syscall.Sysinfo_t
	syscall.Syscall(syscall.SYS_SYSINFO, uintptr(unsafe.Pointer(&sysinfo)), 0, 0)

	return map[string]interface{}{
		"uptime":    sysinfo.Uptime,
		"loads":     [3]uint64{sysinfo.Loads[0], sysinfo.Loads[1], sysinfo.Loads[2]},
		"totalram":  sysinfo.Totalram,
		"freeram":   sysinfo.Freeram,
		"sharedram": sysinfo.Sharedram,
		"bufferram": sysinfo.Bufferram,
		"totalswap": sysinfo.Totalswap,
		"freeswap":  sysinfo.Freeswap,
		"procs":     sysinfo.Procs,
	}
}

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
}

// 系统调用沙箱
type SystemCallSandbox struct {
	config *SandboxConfig
}

// 创建新的沙箱
func NewSystemCallSandbox(config *SandboxConfig) *SystemCallSandbox {
	return &SystemCallSandbox{
		config: config,
	}
}

// 执行程序
func (s *SystemCallSandbox) Execute(ctx context.Context, executable string, args []string) (*ExecuteResult, error) {
	logx.Infof("Starting execution: %s %v", executable, args)

	// 创建命令
	cmd := exec.CommandContext(ctx, executable, args...)

	// 设置进程属性
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// 创建新的命名空间
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS,
		// 设置用户和组
		Credential: &syscall.Credential{
			Uid: uint32(s.config.UID),
			Gid: uint32(s.config.GID),
		},
		// 设置chroot
		Chroot: s.config.Chroot,
	}

	// 设置工作目录
	cmd.Dir = s.config.WorkDir

	// 设置环境变量
	cmd.Env = s.config.Environment

	// 设置输入输出重定向
	if err := s.setupIO(cmd); err != nil {
		return nil, fmt.Errorf("failed to setup IO: %w", err)
	}

	// 设置资源限制
	if err := s.setResourceLimits(); err != nil {
		return nil, fmt.Errorf("failed to set resource limits: %w", err)
	}

	// 启动进程
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	logx.Infof("Process started with PID: %d", cmd.Process.Pid)

	// 监控进程执行
	result, err := s.monitorProcess(cmd.Process.Pid, startTime)
	if err != nil {
		// 确保进程被终止
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to monitor process: %w", err)
	}

	// 等待进程结束
	cmd.Wait()

	logx.Infof("Process execution completed: status=%d, time=%dms, memory=%dKB",
		result.Status, result.TimeUsed, result.MemoryUsed)

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

// 设置资源限制
func (s *SystemCallSandbox) setResourceLimits() error {
	// CPU时间限制
	if s.config.TimeLimit > 0 {
		timeLimit := s.config.TimeLimit / 1000 // 转换为秒
		if err := syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{
			Cur: uint64(timeLimit),
			Max: uint64(timeLimit),
		}); err != nil {
			return fmt.Errorf("failed to set CPU time limit: %w", err)
		}
	}

	// 内存限制
	if s.config.MemoryLimit > 0 {
		memoryLimit := s.config.MemoryLimit * 1024 // 转换为字节
		if err := syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
			Cur: uint64(memoryLimit),
			Max: uint64(memoryLimit),
		}); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
	}

	// 栈大小限制
	if s.config.StackLimit > 0 {
		stackLimit := s.config.StackLimit * 1024 // 转换为字节
		if err := syscall.Setrlimit(syscall.RLIMIT_STACK, &syscall.Rlimit{
			Cur: uint64(stackLimit),
			Max: uint64(stackLimit),
		}); err != nil {
			return fmt.Errorf("failed to set stack limit: %w", err)
		}
	}

	// 文件大小限制
	if s.config.FileSizeLimit > 0 {
		fileSizeLimit := s.config.FileSizeLimit * 1024 // 转换为字节
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{
			Cur: uint64(fileSizeLimit),
			Max: uint64(fileSizeLimit),
		}); err != nil {
			return fmt.Errorf("failed to set file size limit: %w", err)
		}
	}

	// 进程数限制 (在Linux上RLIMIT_NPROC可能不可用，这里注释掉)
	// if s.config.ProcessLimit > 0 {
	// 	if err := syscall.Setrlimit(syscall.RLIMIT_NPROC, &syscall.Rlimit{
	// 		Cur: uint64(s.config.ProcessLimit),
	// 		Max: uint64(s.config.ProcessLimit),
	// 	}); err != nil {
	// 		return fmt.Errorf("failed to set process limit: %w", err)
	// 	}
	// }

	return nil
}

// 监控进程执行
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

// 系统调用过滤器（需要使用libseccomp，这里提供接口定义）
func (s *SystemCallSandbox) setupSeccomp() error {
	// 这里需要使用CGO调用libseccomp库
	// 由于复杂性，这里只提供接口定义
	// 实际实现需要链接libseccomp库

	if !s.config.EnableSeccomp || len(s.config.AllowedSyscalls) == 0 {
		return nil
	}

	logx.Info("Setting up seccomp filter")

	// TODO: 实现seccomp过滤器
	// 1. 创建seccomp上下文
	// 2. 设置默认动作为KILL
	// 3. 为允许的系统调用添加ALLOW规则
	// 4. 加载过滤器

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

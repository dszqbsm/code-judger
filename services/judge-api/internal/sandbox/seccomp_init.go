package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/zeromicro/go-zero/core/logx"
)

// SeccompConfig 用于传递seccomp配置的结构体
type SeccompConfig struct {
	EnableSeccomp   bool   `json:"enable_seccomp"`
	AllowedSyscalls []int  `json:"allowed_syscalls"`
	Language        string `json:"language"`
}

// 创建seccomp初始化程序
// 原理：由于seccomp过滤器需要在目标进程中安装，我们创建一个Go程序作为初始化器
// 该程序先安装seccomp过滤器，再执行实际的用户程序
func (s *SystemCallSandbox) createSeccompInitializer(executable string, args []string, workDir string) (string, error) {
	if !s.config.EnableSeccomp {
		return executable, nil // 不需要seccomp，直接返回原程序
	}

	logx.Info("Creating seccomp initializer program")

	// 1. 创建seccomp配置文件
	seccompConfig := SeccompConfig{
		EnableSeccomp:   s.config.EnableSeccomp,
		AllowedSyscalls: s.config.AllowedSyscalls,
		Language:        "", // 可以从配置中获取，这里暂时为空
	}

	configPath := filepath.Join(workDir, "seccomp_config.json")
	configData, err := json.Marshal(seccompConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal seccomp config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return "", fmt.Errorf("failed to write seccomp config: %w", err)
	}

	// 确保配置文件权限正确
	if err := os.Chown(configPath, 65534, 65534); err != nil {
		logx.Errorf("Failed to change ownership of seccomp config: %v", err)
	}

	// 2. 创建C语言版本的seccomp初始化程序源码
	initializerPath := filepath.Join(workDir, "seccomp_init.c")
	initializerSource := s.generateCSeccompInitializerSource(executable, args, configPath)

	if err := os.WriteFile(initializerPath, []byte(initializerSource), 0644); err != nil {
		return "", fmt.Errorf("failed to write seccomp initializer: %w", err)
	}

	// 确保seccomp初始化程序源码文件权限正确
	if err := os.Chown(initializerPath, 65534, 65534); err != nil {
		logx.Errorf("Failed to change ownership of seccomp initializer source: %v", err)
	}

	// 3. 编译C语言版本的seccomp初始化程序，链接libseccomp库
	binaryPath := filepath.Join(workDir, "seccomp_init")
	cmd := exec.Command("gcc", "-o", binaryPath, initializerPath, "-lseccomp")
	cmd.Dir = workDir

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to compile seccomp initializer: %w, output: %s", err, string(output))
	}

	// 确保编译后的二进制文件权限正确
	if err := os.Chmod(binaryPath, 0755); err != nil {
		logx.Errorf("Failed to change permissions of seccomp initializer binary: %v", err)
	}
	if err := os.Chown(binaryPath, 65534, 65534); err != nil {
		logx.Errorf("Failed to change ownership of seccomp initializer binary: %v", err)
	}

	logx.Infof("Created seccomp initializer: %s", binaryPath)
	return binaryPath, nil
}

// 生成seccomp初始化程序的Go源码
func (s *SystemCallSandbox) generateSeccompInitializerSource(executable string, args []string, configPath string) string {
	// 构建参数列表
	argsStr := "[]string{"
	for _, arg := range args {
		argsStr += fmt.Sprintf(`"%s",`, arg)
	}
	argsStr += "}"

	source := fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"syscall"
	"unsafe"
)

// SeccompConfig 配置结构体
type SeccompConfig struct {
	EnableSeccomp   bool  `+"`json:\"enable_seccomp\"`"+`
	AllowedSyscalls []int `+"`json:\"allowed_syscalls\"`"+`
	Language        string `+"`json:\"language\"`"+`
}

// seccomp常量
const (
	SYS_SECCOMP = 317
	SECCOMP_SET_MODE_FILTER = 1
	SECCOMP_RET_KILL_PROCESS = 0x80000000
	SECCOMP_RET_ALLOW = 0x7fff0000
	
	BPF_LD  = 0x00
	BPF_JMP = 0x05
	BPF_RET = 0x06
	BPF_W   = 0x00
	BPF_ABS = 0x20
	BPF_JEQ = 0x10
	BPF_K   = 0x00
	
	SECCOMP_DATA_NR_OFFSET = 0
	SECCOMP_DATA_ARCH_OFFSET = 4
)

// BPF指令结构体
type BPFInstruction struct {
	Code uint16
	JT   uint8
	JF   uint8
	K    uint32
}

// BPF程序结构体
type BPFProgram struct {
	Len    uint16
	Filter *BPFInstruction
}

func main() {
	// 设置Go运行时内存优化
	debug.SetGCPercent(10)        // 降低GC触发阈值
	debug.SetMemoryLimit(32 << 20) // 设置32MB内存限制
	
	// 1. 读取seccomp配置
	configData, err := os.ReadFile("%s")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read seccomp config: %%v\n", err)
		os.Exit(1)
	}
	
	var config SeccompConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse seccomp config: %%v\n", err)
		os.Exit(1)
	}
	
	// 2. 如果启用了seccomp，安装过滤器
	if config.EnableSeccomp && len(config.AllowedSyscalls) > 0 {
		if err := installSeccompFilter(config.AllowedSyscalls); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to install seccomp filter: %%v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Seccomp filter installed with %%d allowed syscalls\n", len(config.AllowedSyscalls))
	}
	
	// 3. 执行实际程序
	cmd := exec.Command("%s", %s...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Failed to execute program: %%v\n", err)
		os.Exit(1)
	}
}

// 安装seccomp过滤器
func installSeccompFilter(allowedSyscalls []int) error {
	// 构建BPF程序
	var instructions []BPFInstruction
	
	// 1. 验证架构
	instructions = append(instructions, BPFInstruction{
		Code: BPF_LD | BPF_W | BPF_ABS,
		JT:   0,
		JF:   0,
		K:    SECCOMP_DATA_ARCH_OFFSET,
	})
	instructions = append(instructions, BPFInstruction{
		Code: BPF_JMP | BPF_JEQ | BPF_K,
		JT:   0,
		JF:   1,
		K:    0xc000003e, // AUDIT_ARCH_X86_64
	})
	instructions = append(instructions, BPFInstruction{
		Code: BPF_RET | BPF_K,
		JT:   0,
		JF:   0,
		K:    SECCOMP_RET_KILL_PROCESS,
	})
	
	// 2. 加载系统调用号
	instructions = append(instructions, BPFInstruction{
		Code: BPF_LD | BPF_W | BPF_ABS,
		JT:   0,
		JF:   0,
		K:    SECCOMP_DATA_NR_OFFSET,
	})
	
	// 3. 检查允许的系统调用
	for i, syscallNum := range allowedSyscalls {
		if i == len(allowedSyscalls)-1 {
			// 最后一个
			instructions = append(instructions, BPFInstruction{
				Code: BPF_JMP | BPF_JEQ | BPF_K,
				JT:   1,
				JF:   0,
				K:    uint32(syscallNum),
			})
		} else {
			instructions = append(instructions, BPFInstruction{
				Code: BPF_JMP | BPF_JEQ | BPF_K,
				JT:   uint32(len(allowedSyscalls) - i + 1),
				JF:   0,
				K:    uint32(syscallNum),
			})
		}
	}
	
	// 4. 默认动作：杀死进程
	instructions = append(instructions, BPFInstruction{
		Code: BPF_RET | BPF_K,
		JT:   0,
		JF:   0,
		K:    SECCOMP_RET_KILL_PROCESS,
	})
	
	// 5. 允许动作
	instructions = append(instructions, BPFInstruction{
		Code: BPF_RET | BPF_K,
		JT:   0,
		JF:   0,
		K:    SECCOMP_RET_ALLOW,
	})
	
	// 创建BPF程序
	program := BPFProgram{
		Len:    uint16(len(instructions)),
		Filter: &instructions[0],
	}
	
	// 安装过滤器
	ret, _, errno := syscall.Syscall(SYS_SECCOMP, SECCOMP_SET_MODE_FILTER, 0, uintptr(unsafe.Pointer(&program)))
	if ret != 0 {
		return fmt.Errorf("seccomp system call failed: errno=%%d", errno)
	}
	
	return nil
}
`, configPath, executable, argsStr)

	return source
}

// 清理seccomp相关的临时文件
func (s *SystemCallSandbox) cleanupSeccompFiles(workDir string) {
	files := []string{
		"seccomp_config.json",
		"seccomp_init.go",
		"seccomp_init.c",
		"seccomp_init",
	}

	for _, file := range files {
		path := filepath.Join(workDir, file)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			logx.Errorf("Failed to cleanup seccomp file %s: %v", path, err)
		}
	}
}

// 生成C语言版本的seccomp初始化程序源码
func (s *SystemCallSandbox) generateCSeccompInitializerSource(executable string, args []string, configPath string) string {
	// 构建参数列表 - 如果没有参数就不添加额外的参数
	argsStr := ""
	if len(args) > 0 {
		for _, arg := range args {
			argsStr += fmt.Sprintf(`, "%s"`, arg)
		}
	}

	source := fmt.Sprintf(`#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <seccomp.h>

int main() {
    scmp_filter_ctx ctx;
    
    // 创建seccomp上下文，默认动作是杀死进程
    ctx = seccomp_init(SCMP_ACT_KILL);
    if (ctx == NULL) {
        perror("seccomp_init");
        return 1;
    }
    
    // 允许基本的系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(read), 0) < 0) {
        perror("seccomp_rule_add(read)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(write), 0) < 0) {
        perror("seccomp_rule_add(write)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(exit), 0) < 0) {
        perror("seccomp_rule_add(exit)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(exit_group), 0) < 0) {
        perror("seccomp_rule_add(exit_group)");
        goto cleanup;
    }
    
    // 内存管理系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(brk), 0) < 0) {
        perror("seccomp_rule_add(brk)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(mmap), 0) < 0) {
        perror("seccomp_rule_add(mmap)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(munmap), 0) < 0) {
        perror("seccomp_rule_add(munmap)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(mremap), 0) < 0) {
        perror("seccomp_rule_add(mremap)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(mprotect), 0) < 0) {
        perror("seccomp_rule_add(mprotect)");
        goto cleanup;
    }
    
    // 文件和I/O系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(open), 0) < 0) {
        perror("seccomp_rule_add(open)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(openat), 0) < 0) {
        perror("seccomp_rule_add(openat)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(close), 0) < 0) {
        perror("seccomp_rule_add(close)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(lseek), 0) < 0) {
        perror("seccomp_rule_add(lseek)");
        goto cleanup;
    }
    
    // 进程和信号系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(rt_sigaction), 0) < 0) {
        perror("seccomp_rule_add(rt_sigaction)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(rt_sigprocmask), 0) < 0) {
        perror("seccomp_rule_add(rt_sigprocmask)");
        goto cleanup;
    }
    
    // 获取系统信息
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(arch_prctl), 0) < 0) {
        perror("seccomp_rule_add(arch_prctl)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(uname), 0) < 0) {
        perror("seccomp_rule_add(uname)");
        goto cleanup;
    }
    
    // 执行程序系统调用 - 这是关键的缺失调用！
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(execve), 0) < 0) {
        perror("seccomp_rule_add(execve)");
        goto cleanup;
    }
    
    // 文件访问权限检查系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(access), 0) < 0) {
        perror("seccomp_rule_add(access)");
        goto cleanup;
    }
    
    // 获取文件状态信息系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(newfstatat), 0) < 0) {
        perror("seccomp_rule_add(newfstatat)");
        goto cleanup;
    }
    
    // 其他可能需要的文件系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(fstat), 0) < 0) {
        perror("seccomp_rule_add(fstat)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(stat), 0) < 0) {
        perror("seccomp_rule_add(stat)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(lstat), 0) < 0) {
        perror("seccomp_rule_add(lstat)");
        goto cleanup;
    }
    
    // 指定偏移量读取文件系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(pread64), 0) < 0) {
        perror("seccomp_rule_add(pread64)");
        goto cleanup;
    }
    
    // 其他可能需要的读写系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(pwrite64), 0) < 0) {
        perror("seccomp_rule_add(pwrite64)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(readv), 0) < 0) {
        perror("seccomp_rule_add(readv)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(writev), 0) < 0) {
        perror("seccomp_rule_add(writev)");
        goto cleanup;
    }
    
    // 线程相关系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(set_tid_address), 0) < 0) {
        perror("seccomp_rule_add(set_tid_address)");
        goto cleanup;
    }
    
    // 其他常见的程序运行时系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(set_robust_list), 0) < 0) {
        perror("seccomp_rule_add(set_robust_list)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(rseq), 0) < 0) {
        perror("seccomp_rule_add(rseq)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(prlimit64), 0) < 0) {
        perror("seccomp_rule_add(prlimit64)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(getrandom), 0) < 0) {
        perror("seccomp_rule_add(getrandom)");
        goto cleanup;
    }
    
    // 用户空间同步原语系统调用 - 非常重要！
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(futex), 0) < 0) {
        perror("seccomp_rule_add(futex)");
        goto cleanup;
    }
    
    // 其他可能需要的进程管理系统调用
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(getpid), 0) < 0) {
        perror("seccomp_rule_add(getpid)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(gettid), 0) < 0) {
        perror("seccomp_rule_add(gettid)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(getuid), 0) < 0) {
        perror("seccomp_rule_add(getuid)");
        goto cleanup;
    }
    
    if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(getgid), 0) < 0) {
        perror("seccomp_rule_add(getgid)");
        goto cleanup;
    }
    
    // 加载seccomp过滤器
    if (seccomp_load(ctx) < 0) {
        perror("seccomp_load");
        goto cleanup;
    }
    
    // 释放上下文
    seccomp_release(ctx);
    
    // 执行目标程序
    char *argv[] = {"%s"%s, NULL};
    execv("%s", argv);
    
    // 如果execv失败
    perror("execv");
    return 1;
    
cleanup:
    seccomp_release(ctx);
    return 1;
}`, executable, argsStr, executable)

	return source
}

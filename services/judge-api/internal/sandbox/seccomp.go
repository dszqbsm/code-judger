package sandbox

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/zeromicro/go-zero/core/logx"
)

// seccomp-bpf 系统调用过滤器实现
// 原理：通过BPF程序自定义过滤规则，根据过滤规则判断是否允许进程执行该系统调用
// seccomp-bpf是Linux内核提供的一种系统调用过滤机制，能够控制进程可执行的系统调用

// seccomp常量定义
const (
	// seccomp操作模式
	SECCOMP_MODE_DISABLED = 0 // 禁用seccomp
	SECCOMP_MODE_STRICT   = 1 // 严格模式，只允许read、write、exit、sigreturn
	SECCOMP_MODE_FILTER   = 2 // 过滤模式，使用BPF程序自定义规则

	// seccomp系统调用
	SYS_SECCOMP = 317

	// seccomp操作
	SECCOMP_SET_MODE_STRICT  = 0
	SECCOMP_SET_MODE_FILTER  = 1
	SECCOMP_GET_ACTION_AVAIL = 2

	// BPF过滤器返回值
	SECCOMP_RET_KILL_PROCESS = 0x80000000 // 终止进程
	SECCOMP_RET_KILL_THREAD  = 0x00000000 // 终止线程
	SECCOMP_RET_TRAP         = 0x00030000 // 发送SIGSYS信号
	SECCOMP_RET_ERRNO        = 0x00050000 // 返回errno错误码
	SECCOMP_RET_TRACE        = 0x7ff00000 // ptrace跟踪
	SECCOMP_RET_LOG          = 0x7ffc0000 // 记录日志
	SECCOMP_RET_ALLOW        = 0x7fff0000 // 允许执行

	// BPF指令操作码
	BPF_LD   = 0x00 // 加载指令
	BPF_LDX  = 0x01 // 加载索引指令
	BPF_ST   = 0x02 // 存储指令
	BPF_STX  = 0x03 // 存储索引指令
	BPF_ALU  = 0x04 // 算术逻辑指令
	BPF_JMP  = 0x05 // 跳转指令
	BPF_RET  = 0x06 // 返回指令
	BPF_MISC = 0x07 // 其他指令

	// BPF寻址模式
	BPF_W   = 0x00 // 32位字
	BPF_H   = 0x08 // 16位半字
	BPF_B   = 0x10 // 8位字节
	BPF_IMM = 0x00 // 立即数
	BPF_ABS = 0x20 // 绝对偏移
	BPF_IND = 0x40 // 相对偏移
	BPF_MEM = 0x60 // 内存
	BPF_LEN = 0x80 // 数据包长度
	BPF_MSH = 0xa0 // 最高有效半字
	BPF_K   = 0x00 // 常量

	// BPF跳转条件
	BPF_JA   = 0x00 // 无条件跳转
	BPF_JEQ  = 0x10 // 等于跳转
	BPF_JGT  = 0x20 // 大于跳转
	BPF_JGE  = 0x30 // 大于等于跳转
	BPF_JSET = 0x40 // 位与跳转

	// BPF算术操作
	BPF_ADD = 0x00 // 加法
	BPF_SUB = 0x10 // 减法
	BPF_MUL = 0x20 // 乘法
	BPF_DIV = 0x30 // 除法
	BPF_OR  = 0x40 // 或操作
	BPF_AND = 0x50 // 与操作
	BPF_LSH = 0x60 // 左移
	BPF_RSH = 0x70 // 右移
	BPF_NEG = 0x80 // 取负
	BPF_MOD = 0x90 // 取模
	BPF_XOR = 0xa0 // 异或

	// seccomp_data结构体偏移量
	// struct seccomp_data {
	//     int nr;                 /* 系统调用号 */
	//     __u32 arch;            /* 架构 */
	//     __u64 instruction_pointer; /* 指令指针 */
	//     __u64 args[6];         /* 系统调用参数 */
	// };
	SECCOMP_DATA_NR_OFFSET   = 0  // 系统调用号偏移
	SECCOMP_DATA_ARCH_OFFSET = 4  // 架构偏移
	SECCOMP_DATA_IP_OFFSET   = 8  // 指令指针偏移
	SECCOMP_DATA_ARGS_OFFSET = 16 // 参数偏移
)

// BPF指令结构体
type BPFInstruction struct {
	Code uint16 // 操作码
	JT   uint8  // 跳转条件为真时的偏移
	JF   uint8  // 跳转条件为假时的偏移
	K    uint32 // 常量值
}

// BPF程序结构体
type BPFProgram struct {
	Len    uint16          // 指令数量
	Filter *BPFInstruction // 指令数组指针
}

// seccomp过滤器
type SeccompFilter struct {
	allowedSyscalls map[int]bool     // 允许的系统调用集合
	defaultAction   uint32           // 默认动作
	instructions    []BPFInstruction // BPF指令集
}

// 创建新的seccomp过滤器
func NewSeccompFilter(allowedSyscalls []int, defaultAction uint32) *SeccompFilter {
	filter := &SeccompFilter{
		allowedSyscalls: make(map[int]bool),
		defaultAction:   defaultAction,
		instructions:    make([]BPFInstruction, 0),
	}

	// 构建允许的系统调用集合
	for _, syscallNum := range allowedSyscalls {
		filter.allowedSyscalls[syscallNum] = true
	}

	logx.Infof("Created seccomp filter with %d allowed syscalls", len(allowedSyscalls))
	return filter
}

// 构建BPF程序
// 原理：将系统调用白名单转换为BPF指令序列，实现高效的系统调用过滤
func (f *SeccompFilter) buildBPFProgram() error {
	logx.Info("Building BPF program for seccomp filter")

	// 重置指令集
	f.instructions = f.instructions[:0]

	// 1. 验证架构 - 确保是x86_64架构
	// 加载架构字段到累加器
	f.addInstruction(BPF_LD|BPF_W|BPF_ABS, 0, 0, SECCOMP_DATA_ARCH_OFFSET)
	// 比较是否为x86_64架构
	f.addInstruction(BPF_JMP|BPF_JEQ|BPF_K, 0, 1, 0xc000003e) // AUDIT_ARCH_X86_64
	// 架构不匹配则终止进程
	f.addInstruction(BPF_RET|BPF_K, 0, 0, SECCOMP_RET_KILL_PROCESS)

	// 2. 加载系统调用号到累加器
	f.addInstruction(BPF_LD|BPF_W|BPF_ABS, 0, 0, SECCOMP_DATA_NR_OFFSET)

	// 3. 构建系统调用白名单检查逻辑
	// 将允许的系统调用号排序，构建高效的跳转表
	allowedSyscallList := make([]int, 0, len(f.allowedSyscalls))
	for syscallNum := range f.allowedSyscalls {
		allowedSyscallList = append(allowedSyscallList, syscallNum)
	}

	// 简单排序（冒泡排序，适用于小规模数据）
	for i := 0; i < len(allowedSyscallList)-1; i++ {
		for j := 0; j < len(allowedSyscallList)-i-1; j++ {
			if allowedSyscallList[j] > allowedSyscallList[j+1] {
				allowedSyscallList[j], allowedSyscallList[j+1] = allowedSyscallList[j+1], allowedSyscallList[j]
			}
		}
	}

	// 为每个允许的系统调用生成检查指令
	for i, syscallNum := range allowedSyscallList {
		// 检查是否等于当前系统调用号
		if i == len(allowedSyscallList)-1 {
			// 最后一个系统调用，不匹配则执行默认动作
			f.addInstruction(BPF_JMP|BPF_JEQ|BPF_K, 1, 0, uint32(syscallNum))
		} else {
			// 不是最后一个，匹配则允许，不匹配则继续检查下一个
			jumpOffset := len(allowedSyscallList) - i
			if jumpOffset > 255 {
				jumpOffset = 255 // 限制在uint8范围内
			}
			f.addInstruction(BPF_JMP|BPF_JEQ|BPF_K, uint8(jumpOffset), 0, uint32(syscallNum))
		}
	}

	// 4. 默认动作 - 不在白名单中的系统调用
	f.addInstruction(BPF_RET|BPF_K, 0, 0, f.defaultAction)

	// 5. 允许动作 - 在白名单中的系统调用
	f.addInstruction(BPF_RET|BPF_K, 0, 0, SECCOMP_RET_ALLOW)

	logx.Infof("Built BPF program with %d instructions for %d allowed syscalls",
		len(f.instructions), len(allowedSyscallList))

	return nil
}

// 添加BPF指令的辅助函数
func (f *SeccompFilter) addInstruction(code uint16, jt, jf uint8, k uint32) {
	instruction := BPFInstruction{
		Code: code,
		JT:   jt,
		JF:   jf,
		K:    k,
	}
	f.instructions = append(f.instructions, instruction)
}

// 安装seccomp过滤器
// 原理：通过seccomp系统调用将构建的BPF程序加载到内核，启用系统调用过滤
func (f *SeccompFilter) Install() error {
	logx.Info("Installing seccomp filter")

	// 1. 构建BPF程序
	if err := f.buildBPFProgram(); err != nil {
		return fmt.Errorf("failed to build BPF program: %w", err)
	}

	// 2. 创建BPF程序结构体
	program := BPFProgram{
		Len:    uint16(len(f.instructions)),
		Filter: &f.instructions[0],
	}

	// 3. 调用seccomp系统调用安装过滤器
	// seccomp(SECCOMP_SET_MODE_FILTER, 0, &program)
	ret, _, errno := syscall.Syscall(SYS_SECCOMP,
		SECCOMP_SET_MODE_FILTER,
		0,
		uintptr(unsafe.Pointer(&program)))

	if ret != 0 {
		return fmt.Errorf("seccomp system call failed: errno=%d", errno)
	}

	logx.Infof("Seccomp filter installed successfully with %d instructions", len(f.instructions))
	return nil
}

// 验证seccomp过滤器是否正常工作
func (f *SeccompFilter) Validate() error {
	logx.Info("Validating seccomp filter")

	// 检查是否有允许的系统调用
	if len(f.allowedSyscalls) == 0 {
		return fmt.Errorf("no allowed syscalls configured")
	}

	// 检查BPF程序是否已构建
	if len(f.instructions) == 0 {
		return fmt.Errorf("BPF program not built")
	}

	// 验证BPF程序结构
	if len(f.instructions) < 4 {
		return fmt.Errorf("BPF program too short, minimum 4 instructions required")
	}

	// 验证第一条指令是否为架构检查
	firstInst := f.instructions[0]
	if firstInst.Code != (BPF_LD|BPF_W|BPF_ABS) || firstInst.K != SECCOMP_DATA_ARCH_OFFSET {
		return fmt.Errorf("invalid first instruction, should be architecture check")
	}

	logx.Info("Seccomp filter validation passed")
	return nil
}

// 获取BPF程序的人类可读表示（用于调试）
func (f *SeccompFilter) GetBPFDisassembly() []string {
	var disasm []string

	for i, inst := range f.instructions {
		var line string

		// 解析操作码
		switch inst.Code & 0x07 {
		case BPF_LD:
			if inst.Code&BPF_ABS != 0 {
				line = fmt.Sprintf("%3d: LD  [%d]", i, inst.K)
			} else {
				line = fmt.Sprintf("%3d: LD  #%d", i, inst.K)
			}
		case BPF_JMP:
			if inst.Code&BPF_JEQ != 0 {
				line = fmt.Sprintf("%3d: JEQ #%d jt=%d jf=%d", i, inst.K, inst.JT, inst.JF)
			} else if inst.Code&BPF_JGT != 0 {
				line = fmt.Sprintf("%3d: JGT #%d jt=%d jf=%d", i, inst.K, inst.JT, inst.JF)
			} else if inst.Code&BPF_JGE != 0 {
				line = fmt.Sprintf("%3d: JGE #%d jt=%d jf=%d", i, inst.K, inst.JT, inst.JF)
			} else {
				line = fmt.Sprintf("%3d: JMP jt=%d jf=%d", i, inst.JT, inst.JF)
			}
		case BPF_RET:
			var action string
			switch inst.K {
			case SECCOMP_RET_ALLOW:
				action = "ALLOW"
			case SECCOMP_RET_KILL_PROCESS:
				action = "KILL_PROCESS"
			case SECCOMP_RET_KILL_THREAD:
				action = "KILL_THREAD"
			case SECCOMP_RET_TRAP:
				action = "TRAP"
			default:
				action = fmt.Sprintf("0x%x", inst.K)
			}
			line = fmt.Sprintf("%3d: RET %s", i, action)
		default:
			line = fmt.Sprintf("%3d: ??? code=0x%x k=%d", i, inst.Code, inst.K)
		}

		disasm = append(disasm, line)
	}

	return disasm
}

// 创建基于语言的seccomp过滤器
func CreateLanguageSeccompFilter(language string) (*SeccompFilter, error) {
	// 获取语言特定的系统调用白名单
	allowedSyscalls := GetSyscallWhitelist(language)
	if len(allowedSyscalls) == 0 {
		return nil, fmt.Errorf("no syscall whitelist found for language: %s", language)
	}

	// 创建过滤器，默认动作为终止进程
	filter := NewSeccompFilter(allowedSyscalls, SECCOMP_RET_KILL_PROCESS)

	logx.Infof("Created seccomp filter for %s with %d allowed syscalls",
		language, len(allowedSyscalls))

	return filter, nil
}

// 创建严格模式seccomp过滤器（仅允许最基本的系统调用）
func CreateStrictSeccompFilter() (*SeccompFilter, error) {
	// 严格模式只允许最基本的系统调用
	strictSyscalls := []int{
		0,  // read
		1,  // write
		60, // exit
		15, // rt_sigreturn
	}

	filter := NewSeccompFilter(strictSyscalls, SECCOMP_RET_KILL_PROCESS)

	logx.Info("Created strict seccomp filter with minimal syscall set")
	return filter, nil
}

// 创建宽松模式seccomp过滤器（记录违规但不终止进程）
func CreateLoggingSeccompFilter(language string) (*SeccompFilter, error) {
	allowedSyscalls := GetSyscallWhitelist(language)
	if len(allowedSyscalls) == 0 {
		return nil, fmt.Errorf("no syscall whitelist found for language: %s", language)
	}

	// 使用LOG动作而不是KILL，便于调试
	filter := NewSeccompFilter(allowedSyscalls, SECCOMP_RET_LOG)

	logx.Infof("Created logging seccomp filter for %s", language)
	return filter, nil
}

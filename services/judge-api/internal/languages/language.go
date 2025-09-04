package languages

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/config"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/sandbox"
)

// 编译结果
type CompileResult struct {
	Success        bool          // 编译是否成功
	ExecutablePath string        // 可执行文件路径
	CompileTime    time.Duration // 编译时间
	Message        string        // 编译信息（错误或警告）
}

// 语言执行器接口
type LanguageExecutor interface {
	// 获取语言信息
	GetName() string
	GetDisplayName() string
	GetVersion() string
	GetFileExtension() string

	// 编译代码
	Compile(ctx context.Context, code string, workDir string) (*CompileResult, error)

	// 执行代码
	Execute(ctx context.Context, executablePath string, workDir string, config *ExecutionConfig) (*sandbox.ExecuteResult, error)

	// 是否需要编译
	IsCompiled() bool

	// 获取资源限制倍数
	GetTimeMultiplier() float64
	GetMemoryMultiplier() float64
	GetMaxProcesses() int

	// 获取允许的系统调用
	GetAllowedSyscalls() []int
}

// 执行配置
type ExecutionConfig struct {
	TimeLimit   int64    // 时间限制(毫秒)
	MemoryLimit int64    // 内存限制(KB)
	InputFile   string   // 输入文件
	OutputFile  string   // 输出文件
	ErrorFile   string   // 错误输出文件
	Environment []string // 环境变量
}

// 基础语言执行器
type BaseLanguageExecutor struct {
	name             string
	displayName      string
	version          string
	fileExtension    string
	compileCommand   string
	executeCommand   string
	compileTimeout   time.Duration
	timeMultiplier   float64
	memoryMultiplier float64
	maxProcesses     int
	allowedSyscalls  []int
	sandbox          *sandbox.SystemCallSandbox
}

// C++语言执行器
type CppExecutor struct {
	*BaseLanguageExecutor
}

func NewCppExecutor(config config.CompilerConf) *CppExecutor {
	base := &BaseLanguageExecutor{
		name:             "cpp",
		displayName:      "C++",
		version:          config.Version,
		fileExtension:    ".cpp",
		compileCommand:   config.CompileCommand,
		executeCommand:   config.ExecuteCommand,
		compileTimeout:   time.Duration(config.CompileTimeout) * time.Millisecond,
		timeMultiplier:   config.TimeMultiplier,
		memoryMultiplier: config.MemoryMultiplier,
		maxProcesses:     config.MaxProcesses,
		allowedSyscalls:  config.AllowedSyscalls,
	}

	return &CppExecutor{BaseLanguageExecutor: base}
}

func (e *CppExecutor) GetName() string              { return e.name }
func (e *CppExecutor) GetDisplayName() string       { return e.displayName }
func (e *CppExecutor) GetVersion() string           { return e.version }
func (e *CppExecutor) GetFileExtension() string     { return e.fileExtension }
func (e *CppExecutor) IsCompiled() bool             { return true }
func (e *CppExecutor) GetTimeMultiplier() float64   { return e.timeMultiplier }
func (e *CppExecutor) GetMemoryMultiplier() float64 { return e.memoryMultiplier }
func (e *CppExecutor) GetMaxProcesses() int         { return e.maxProcesses }
func (e *CppExecutor) GetAllowedSyscalls() []int    { return e.allowedSyscalls }

func (e *CppExecutor) Compile(ctx context.Context, code string, workDir string) (*CompileResult, error) {
	sourceFile := filepath.Join(workDir, "main.cpp")
	executableFile := filepath.Join(workDir, "main")

	// 写入源代码文件
	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	// 确保源代码文件权限正确，让nobody用户能够读取
	if err := os.Chown(sourceFile, 65534, 65534); err != nil {
		return nil, fmt.Errorf("failed to change source file ownership: %w", err)
	}

	// 替换编译命令中的占位符
	compileCmd := strings.ReplaceAll(e.compileCommand, "{executable}", executableFile)
	compileCmd = strings.ReplaceAll(compileCmd, "{source}", sourceFile)

	// 创建编译沙箱配置
	sandboxConfig := &sandbox.SandboxConfig{
		UID:           65534, // nobody用户
		GID:           65534,
		WorkDir:       workDir,
		TimeLimit:     int64(e.compileTimeout.Milliseconds()),
		WallTimeLimit: int64(e.compileTimeout.Milliseconds()) + 1000,
		MemoryLimit:   512 * 1024, // 512MB编译内存限制
		StackLimit:    8 * 1024,   // 8MB栈限制
		FileSizeLimit: 50 * 1024,  // 50MB文件大小限制
		ProcessLimit:  e.maxProcesses,
		ErrorFile:     filepath.Join(workDir, "compile_error.txt"),
		Environment:   []string{"PATH=/usr/bin:/bin"},
	}

	// 创建编译沙箱
	compileSandbox := sandbox.NewSystemCallSandbox(sandboxConfig)

	// 解析编译命令
	cmdParts := strings.Fields(compileCmd)
	if len(cmdParts) == 0 {
		return nil, fmt.Errorf("empty compile command")
	}

	// 执行编译
	startTime := time.Now()
	result, err := compileSandbox.Execute(ctx, cmdParts[0], cmdParts[1:])
	compileTime := time.Since(startTime)

	// 读取编译错误信息
	var compileMessage string
	if errorData, err := os.ReadFile(sandboxConfig.ErrorFile); err == nil {
		compileMessage = string(errorData)
	}

	compileResult := &CompileResult{
		Success:        result != nil && result.Status == sandbox.StatusAccepted,
		ExecutablePath: executableFile,
		CompileTime:    compileTime,
		Message:        compileMessage,
	}

	if err != nil {
		compileResult.Success = false
		compileResult.Message = fmt.Sprintf("Compile error: %v", err)
	}

	return compileResult, nil
}

func (e *CppExecutor) Execute(ctx context.Context, executablePath string, workDir string, config *ExecutionConfig) (*sandbox.ExecuteResult, error) {
	// 创建执行沙箱配置
	sandboxConfig := &sandbox.SandboxConfig{
		UID:             65534, // nobody用户
		GID:             65534,
		WorkDir:         workDir,
		TimeLimit:       config.TimeLimit,
		WallTimeLimit:   config.TimeLimit + 1000, // 增加1秒容错时间
		MemoryLimit:     config.MemoryLimit,
		StackLimit:      8 * 1024,  // 8MB栈限制
		FileSizeLimit:   10 * 1024, // 10MB输出限制
		ProcessLimit:    e.maxProcesses,
		AllowedSyscalls: e.allowedSyscalls,
		EnableSeccomp:   true, // 启用seccomp安全过滤
		InputFile:       config.InputFile,
		OutputFile:      config.OutputFile,
		ErrorFile:       config.ErrorFile,
		Environment:     config.Environment,
	}

	// 创建执行沙箱
	executeSandbox := sandbox.NewSystemCallSandbox(sandboxConfig)

	// 执行程序
	return executeSandbox.Execute(ctx, executablePath, []string{})
}

// C语言执行器
type CExecutor struct {
	*BaseLanguageExecutor
}

func NewCExecutor(config config.CompilerConf) *CExecutor {
	base := &BaseLanguageExecutor{
		name:             "c",
		displayName:      "C",
		version:          config.Version,
		fileExtension:    ".c",
		compileCommand:   config.CompileCommand,
		executeCommand:   config.ExecuteCommand,
		compileTimeout:   time.Duration(config.CompileTimeout) * time.Millisecond,
		timeMultiplier:   config.TimeMultiplier,
		memoryMultiplier: config.MemoryMultiplier,
		maxProcesses:     config.MaxProcesses,
		allowedSyscalls:  config.AllowedSyscalls,
	}

	return &CExecutor{BaseLanguageExecutor: base}
}

func (e *CExecutor) GetName() string              { return e.name }
func (e *CExecutor) GetDisplayName() string       { return e.displayName }
func (e *CExecutor) GetVersion() string           { return e.version }
func (e *CExecutor) GetFileExtension() string     { return e.fileExtension }
func (e *CExecutor) IsCompiled() bool             { return true }
func (e *CExecutor) GetTimeMultiplier() float64   { return e.timeMultiplier }
func (e *CExecutor) GetMemoryMultiplier() float64 { return e.memoryMultiplier }
func (e *CExecutor) GetMaxProcesses() int         { return e.maxProcesses }
func (e *CExecutor) GetAllowedSyscalls() []int    { return e.allowedSyscalls }

func (e *CExecutor) Compile(ctx context.Context, code string, workDir string) (*CompileResult, error) {
	sourceFile := filepath.Join(workDir, "main.c")
	executableFile := filepath.Join(workDir, "main")

	// 写入源代码文件
	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	// 确保源代码文件权限正确，让nobody用户能够读取
	if err := os.Chown(sourceFile, 65534, 65534); err != nil {
		return nil, fmt.Errorf("failed to change source file ownership: %w", err)
	}

	// 替换编译命令中的占位符
	compileCmd := strings.ReplaceAll(e.compileCommand, "{executable}", executableFile)
	compileCmd = strings.ReplaceAll(compileCmd, "{source}", sourceFile)

	// 创建编译沙箱配置
	sandboxConfig := &sandbox.SandboxConfig{
		UID:           65534,
		GID:           65534,
		WorkDir:       workDir,
		TimeLimit:     int64(e.compileTimeout.Milliseconds()),
		WallTimeLimit: int64(e.compileTimeout.Milliseconds()) + 1000,
		MemoryLimit:   512 * 1024,
		StackLimit:    8 * 1024,
		FileSizeLimit: 50 * 1024,
		ProcessLimit:  e.maxProcesses,
		ErrorFile:     filepath.Join(workDir, "compile_error.txt"),
		Environment:   []string{"PATH=/usr/bin:/bin"},
	}

	compileSandbox := sandbox.NewSystemCallSandbox(sandboxConfig)
	cmdParts := strings.Fields(compileCmd)

	startTime := time.Now()
	result, err := compileSandbox.Execute(ctx, cmdParts[0], cmdParts[1:])
	compileTime := time.Since(startTime)

	var compileMessage string
	if errorData, err := os.ReadFile(sandboxConfig.ErrorFile); err == nil {
		compileMessage = string(errorData)
	}

	compileResult := &CompileResult{
		Success:        result != nil && result.Status == sandbox.StatusAccepted,
		ExecutablePath: executableFile,
		CompileTime:    compileTime,
		Message:        compileMessage,
	}

	if err != nil {
		compileResult.Success = false
		compileResult.Message = fmt.Sprintf("Compile error: %v", err)
	}

	return compileResult, nil
}

func (e *CExecutor) Execute(ctx context.Context, executablePath string, workDir string, config *ExecutionConfig) (*sandbox.ExecuteResult, error) {
	sandboxConfig := &sandbox.SandboxConfig{
		UID:             65534,
		GID:             65534,
		WorkDir:         workDir,
		TimeLimit:       config.TimeLimit,
		WallTimeLimit:   config.TimeLimit + 1000,
		MemoryLimit:     config.MemoryLimit,
		StackLimit:      8 * 1024,
		FileSizeLimit:   10 * 1024,
		ProcessLimit:    e.maxProcesses,
		AllowedSyscalls: e.allowedSyscalls,
		EnableSeccomp:   true, // 启用seccomp安全过滤
		InputFile:       config.InputFile,
		OutputFile:      config.OutputFile,
		ErrorFile:       config.ErrorFile,
		Environment:     config.Environment,
	}

	executeSandbox := sandbox.NewSystemCallSandbox(sandboxConfig)
	return executeSandbox.Execute(ctx, executablePath, []string{})
}

// Java语言执行器
type JavaExecutor struct {
	*BaseLanguageExecutor
}

func NewJavaExecutor(config config.CompilerConf) *JavaExecutor {
	base := &BaseLanguageExecutor{
		name:             "java",
		displayName:      "Java",
		version:          config.Version,
		fileExtension:    ".java",
		compileCommand:   config.CompileCommand,
		executeCommand:   config.ExecuteCommand,
		compileTimeout:   time.Duration(config.CompileTimeout) * time.Millisecond,
		timeMultiplier:   config.TimeMultiplier,
		memoryMultiplier: config.MemoryMultiplier,
		maxProcesses:     config.MaxProcesses,
		allowedSyscalls:  config.AllowedSyscalls,
	}

	return &JavaExecutor{BaseLanguageExecutor: base}
}

func (e *JavaExecutor) GetName() string              { return e.name }
func (e *JavaExecutor) GetDisplayName() string       { return e.displayName }
func (e *JavaExecutor) GetVersion() string           { return e.version }
func (e *JavaExecutor) GetFileExtension() string     { return e.fileExtension }
func (e *JavaExecutor) IsCompiled() bool             { return true }
func (e *JavaExecutor) GetTimeMultiplier() float64   { return e.timeMultiplier }
func (e *JavaExecutor) GetMemoryMultiplier() float64 { return e.memoryMultiplier }
func (e *JavaExecutor) GetMaxProcesses() int         { return e.maxProcesses }
func (e *JavaExecutor) GetAllowedSyscalls() []int    { return e.allowedSyscalls }

func (e *JavaExecutor) Compile(ctx context.Context, code string, workDir string) (*CompileResult, error) {
	sourceFile := filepath.Join(workDir, "Main.java")

	// 写入源代码文件
	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	// 确保源代码文件权限正确，让nobody用户能够读取
	if err := os.Chown(sourceFile, 65534, 65534); err != nil {
		return nil, fmt.Errorf("failed to change source file ownership: %w", err)
	}

	// Java编译命令
	compileCmd := strings.ReplaceAll(e.compileCommand, "{source}", sourceFile)

	sandboxConfig := &sandbox.SandboxConfig{
		UID:           65534,
		GID:           65534,
		WorkDir:       workDir,
		TimeLimit:     int64(e.compileTimeout.Milliseconds()),
		WallTimeLimit: int64(e.compileTimeout.Milliseconds()) + 2000, // Java需要更多时间
		MemoryLimit:   1024 * 1024,                                   // 1GB编译内存限制
		StackLimit:    8 * 1024,
		FileSizeLimit: 50 * 1024,
		ProcessLimit:  e.maxProcesses,
		ErrorFile:     filepath.Join(workDir, "compile_error.txt"),
		Environment:   []string{"PATH=/usr/bin:/bin", "JAVA_HOME=/usr/lib/jvm/default-java"},
	}

	compileSandbox := sandbox.NewSystemCallSandbox(sandboxConfig)
	cmdParts := strings.Fields(compileCmd)

	startTime := time.Now()
	result, err := compileSandbox.Execute(ctx, cmdParts[0], cmdParts[1:])
	compileTime := time.Since(startTime)

	var compileMessage string
	if errorData, err := os.ReadFile(sandboxConfig.ErrorFile); err == nil {
		compileMessage = string(errorData)
	}

	compileResult := &CompileResult{
		Success:        result != nil && result.Status == sandbox.StatusAccepted,
		ExecutablePath: filepath.Join(workDir, "Main.class"),
		CompileTime:    compileTime,
		Message:        compileMessage,
	}

	if err != nil {
		compileResult.Success = false
		compileResult.Message = fmt.Sprintf("Compile error: %v", err)
	}

	return compileResult, nil
}

func (e *JavaExecutor) Execute(ctx context.Context, executablePath string, workDir string, config *ExecutionConfig) (*sandbox.ExecuteResult, error) {
	// Java执行命令，需要替换内存限制
	executeCmd := strings.ReplaceAll(e.executeCommand, "{memory_limit}", fmt.Sprintf("%d", config.MemoryLimit/1024))

	sandboxConfig := &sandbox.SandboxConfig{
		UID:             65534,
		GID:             65534,
		WorkDir:         workDir,
		TimeLimit:       config.TimeLimit,
		WallTimeLimit:   config.TimeLimit + 2000, // Java需要更多启动时间
		MemoryLimit:     config.MemoryLimit,
		StackLimit:      8 * 1024,
		FileSizeLimit:   10 * 1024,
		ProcessLimit:    e.maxProcesses,
		AllowedSyscalls: e.allowedSyscalls,
		EnableSeccomp:   false, // 临时禁用seccomp - Java需要更多系统调用
		InputFile:       config.InputFile,
		OutputFile:      config.OutputFile,
		ErrorFile:       config.ErrorFile,
		Environment:     append(config.Environment, "JAVA_HOME=/usr/lib/jvm/default-java"),
	}

	executeSandbox := sandbox.NewSystemCallSandbox(sandboxConfig)
	cmdParts := strings.Fields(executeCmd)

	return executeSandbox.Execute(ctx, cmdParts[0], cmdParts[1:])
}

// Python语言执行器
type PythonExecutor struct {
	*BaseLanguageExecutor
}

func NewPythonExecutor(config config.CompilerConf) *PythonExecutor {
	base := &BaseLanguageExecutor{
		name:             "python",
		displayName:      "Python",
		version:          config.Version,
		fileExtension:    ".py",
		compileCommand:   config.CompileCommand,
		executeCommand:   config.ExecuteCommand,
		compileTimeout:   time.Duration(config.CompileTimeout) * time.Millisecond,
		timeMultiplier:   config.TimeMultiplier,
		memoryMultiplier: config.MemoryMultiplier,
		maxProcesses:     config.MaxProcesses,
		allowedSyscalls:  config.AllowedSyscalls,
	}

	return &PythonExecutor{BaseLanguageExecutor: base}
}

func (e *PythonExecutor) GetName() string              { return e.name }
func (e *PythonExecutor) GetDisplayName() string       { return e.displayName }
func (e *PythonExecutor) GetVersion() string           { return e.version }
func (e *PythonExecutor) GetFileExtension() string     { return e.fileExtension }
func (e *PythonExecutor) IsCompiled() bool             { return false }
func (e *PythonExecutor) GetTimeMultiplier() float64   { return e.timeMultiplier }
func (e *PythonExecutor) GetMemoryMultiplier() float64 { return e.memoryMultiplier }
func (e *PythonExecutor) GetMaxProcesses() int         { return e.maxProcesses }
func (e *PythonExecutor) GetAllowedSyscalls() []int    { return e.allowedSyscalls }

func (e *PythonExecutor) Compile(ctx context.Context, code string, workDir string) (*CompileResult, error) {
	sourceFile := filepath.Join(workDir, "main.py")

	// 写入源代码文件
	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	// 确保源代码文件权限正确，让nobody用户能够读取
	if err := os.Chown(sourceFile, 65534, 65534); err != nil {
		return nil, fmt.Errorf("failed to change source file ownership: %w", err)
	}

	// Python不需要编译，但可以进行语法检查
	return &CompileResult{
		Success:        true,
		ExecutablePath: sourceFile,
		CompileTime:    0,
		Message:        "Python script syntax check passed",
	}, nil
}

func (e *PythonExecutor) Execute(ctx context.Context, executablePath string, workDir string, config *ExecutionConfig) (*sandbox.ExecuteResult, error) {
	executeCmd := strings.ReplaceAll(e.executeCommand, "{source}", executablePath)

	sandboxConfig := &sandbox.SandboxConfig{
		UID:             65534,
		GID:             65534,
		WorkDir:         workDir,
		TimeLimit:       config.TimeLimit,
		WallTimeLimit:   config.TimeLimit + 1000,
		MemoryLimit:     config.MemoryLimit,
		StackLimit:      8 * 1024,
		FileSizeLimit:   10 * 1024,
		ProcessLimit:    e.maxProcesses,
		AllowedSyscalls: e.allowedSyscalls,
		EnableSeccomp:   false, // 临时禁用seccomp - Python需要更多系统调用
		InputFile:       config.InputFile,
		OutputFile:      config.OutputFile,
		ErrorFile:       config.ErrorFile,
		Environment:     append(config.Environment, "PYTHONPATH=/usr/lib/python3.8"),
	}

	executeSandbox := sandbox.NewSystemCallSandbox(sandboxConfig)
	cmdParts := strings.Fields(executeCmd)

	return executeSandbox.Execute(ctx, cmdParts[0], cmdParts[1:])
}

// 语言执行器管理器
type LanguageManager struct {
	executors map[string]LanguageExecutor
}

func NewLanguageManager(compilers map[string]config.CompilerConf) *LanguageManager {
	manager := &LanguageManager{
		executors: make(map[string]LanguageExecutor),
	}

	// 注册语言执行器
	for lang, conf := range compilers {
		switch lang {
		case "cpp":
			manager.executors[lang] = NewCppExecutor(conf)
		case "c":
			manager.executors[lang] = NewCExecutor(conf)
		case "java":
			manager.executors[lang] = NewJavaExecutor(conf)
		case "python":
			manager.executors[lang] = NewPythonExecutor(conf)
			// TODO: 添加更多语言支持
		}
	}

	return manager
}

func (m *LanguageManager) GetExecutor(language string) (LanguageExecutor, error) {
	executor, exists := m.executors[language]
	if !exists {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
	return executor, nil
}

func (m *LanguageManager) GetSupportedLanguages() []string {
	languages := make([]string, 0, len(m.executors))
	for lang := range m.executors {
		languages = append(languages, lang)
	}
	return languages
}

func (m *LanguageManager) GetLanguageConfigs() []LanguageConfigInfo {
	configs := make([]LanguageConfigInfo, 0, len(m.executors))
	for _, executor := range m.executors {
		configs = append(configs, LanguageConfigInfo{
			Name:             executor.GetName(),
			DisplayName:      executor.GetDisplayName(),
			Version:          executor.GetVersion(),
			FileExtension:    executor.GetFileExtension(),
			TimeMultiplier:   executor.GetTimeMultiplier(),
			MemoryMultiplier: executor.GetMemoryMultiplier(),
			IsEnabled:        true,
		})
	}
	return configs
}

type LanguageConfigInfo struct {
	Name             string  `json:"name"`
	DisplayName      string  `json:"display_name"`
	Version          string  `json:"version"`
	FileExtension    string  `json:"file_extension"`
	TimeMultiplier   float64 `json:"time_multiplier"`
	MemoryMultiplier float64 `json:"memory_multiplier"`
	IsEnabled        bool    `json:"is_enabled"`
}

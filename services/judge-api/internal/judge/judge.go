package judge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/config"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/languages"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/sandbox"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// 判题请求
type JudgeRequest struct {
	SubmissionID int64             `json:"submission_id"`
	ProblemID    int64             `json:"problem_id"`
	UserID       int64             `json:"user_id"`
	Language     string            `json:"language"`
	Code         string            `json:"code"`
	TimeLimit    int               `json:"time_limit"`   // 毫秒
	MemoryLimit  int               `json:"memory_limit"` // MB
	TestCases    []*types.TestCase `json:"test_cases"`
}

// 判题引擎
type JudgeEngine struct {
	config          *config.JudgeEngineConf
	languageManager *languages.LanguageManager
	workDir         string
	tempDir         string
}

func NewJudgeEngine(config *config.JudgeEngineConf) *JudgeEngine {
	// 创建语言管理器
	languageManager := languages.NewLanguageManager(config.Compilers)

	return &JudgeEngine{
		config:          config,
		languageManager: languageManager,
		workDir:         config.WorkDir,
		tempDir:         config.TempDir,
	}
}

// 执行判题
func (je *JudgeEngine) Judge(ctx context.Context, req *JudgeRequest) (*types.JudgeResult, error) {
	logx.Infof("Starting judge for submission %d", req.SubmissionID)

	// 验证请求参数
	if err := je.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// 获取语言执行器
	executor, err := je.languageManager.GetExecutor(req.Language)
	if err != nil {
		return nil, fmt.Errorf("unsupported language: %w", err)
	}

	// 创建临时工作目录
	tempDir, err := je.createTempDir(req.SubmissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer je.cleanupTempDir(tempDir)

	// 初始化判题结果
	result := &types.JudgeResult{
		SubmissionId: req.SubmissionID,
		Status:       "judging",
		TestCases:    make([]types.TestCaseResult, 0, len(req.TestCases)),
		JudgeInfo: types.JudgeInfo{
			JudgeServer:     je.getServerID(),
			JudgeTime:       time.Now().Format(time.RFC3339),
			LanguageVersion: executor.GetVersion(),
		},
	}

	// 1. 编译代码
	compileResult, err := je.compileCode(ctx, executor, req.Code, tempDir)
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	result.CompileInfo = types.CompileInfo{
		Success: compileResult.Success,
		Message: compileResult.Message,
		Time:    int(compileResult.CompileTime.Milliseconds()),
	}

	if !compileResult.Success {
		result.Status = "compile_error"
		return result, nil
	}

	// 2. 执行测试用例
	totalScore := 0
	maxTimeUsed := 0
	maxMemoryUsed := 0

	for i, testCase := range req.TestCases {
		logx.Infof("Executing test case %d for submission %d", i+1, req.SubmissionID)

		testResult, err := je.runTestCase(ctx, executor, compileResult.ExecutablePath,
			testCase, tempDir, req.TimeLimit, req.MemoryLimit)
		if err != nil {
			logx.Errorf("Failed to run test case %d: %v", i+1, err)
			testResult = &types.TestCaseResult{
				CaseId:      testCase.CaseId,
				Status:      "system_error",
				Input:       testCase.Input,
				Expected:    testCase.ExpectedOutput,
				Output:      "",
				ErrorOutput: err.Error(),
			}
		}

		result.TestCases = append(result.TestCases, *testResult)

		// 更新最大资源使用
		if testResult.TimeUsed > maxTimeUsed {
			maxTimeUsed = testResult.TimeUsed
		}
		if testResult.MemoryUsed > maxMemoryUsed {
			maxMemoryUsed = testResult.MemoryUsed
		}

		// 计算分数
		if testResult.Status == "accepted" {
			totalScore += 100 / len(req.TestCases) // 平均分配分数
		}

		// 如果不是AC，可以选择提前结束（根据配置）
		if testResult.Status != "accepted" {
			// 这里可以根据配置决定是否继续执行剩余测试用例
			break
		}
	}

	// 3. 设置最终结果
	result.Score = totalScore
	result.TimeUsed = maxTimeUsed
	result.MemoryUsed = maxMemoryUsed

	// 确定最终状态
	result.Status = je.determineFinalStatus(result.TestCases)

	logx.Infof("Judge completed for submission %d: status=%s, score=%d",
		req.SubmissionID, result.Status, result.Score)

	return result, nil
}

// 验证请求参数
func (je *JudgeEngine) validateRequest(req *JudgeRequest) error {
	if req.SubmissionID <= 0 {
		return fmt.Errorf("invalid submission ID")
	}

	if req.Language == "" {
		return fmt.Errorf("language is required")
	}

	if req.Code == "" {
		return fmt.Errorf("code is required")
	}

	if len(req.Code) > je.config.Security.MaxCodeLength {
		return fmt.Errorf("code length exceeds limit")
	}

	if len(req.TestCases) == 0 {
		return fmt.Errorf("test cases are required")
	}

	if req.TimeLimit <= 0 || req.TimeLimit > je.config.ResourceLimits.MaxTimeLimit {
		return fmt.Errorf("invalid time limit")
	}

	if req.MemoryLimit <= 0 || req.MemoryLimit > je.config.ResourceLimits.MaxMemoryLimit {
		return fmt.Errorf("invalid memory limit")
	}

	// 检查禁止的代码模式
	for _, pattern := range je.config.Security.ForbiddenPatterns {
		if strings.Contains(req.Code, pattern) {
			return fmt.Errorf("code contains forbidden pattern: %s", pattern)
		}
	}

	return nil
}

// 编译代码
func (je *JudgeEngine) compileCode(ctx context.Context, executor languages.LanguageExecutor,
	code string, workDir string) (*languages.CompileResult, error) {

	if !executor.IsCompiled() {
		// 解释型语言，不需要编译
		sourceFile := filepath.Join(workDir, "main"+executor.GetFileExtension())
		if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
			return nil, err
		}

		return &languages.CompileResult{
			Success:        true,
			ExecutablePath: sourceFile,
			CompileTime:    0,
			Message:        "No compilation required",
		}, nil
	}

	// 编译型语言，需要编译
	return executor.Compile(ctx, code, workDir)
}

// 执行测试用例
func (je *JudgeEngine) runTestCase(ctx context.Context, executor languages.LanguageExecutor,
	executablePath string, testCase *types.TestCase, workDir string,
	timeLimit int, memoryLimit int) (*types.TestCaseResult, error) {

	// 创建输入输出文件
	inputFile := filepath.Join(workDir, fmt.Sprintf("input_%d.txt", testCase.CaseId))
	outputFile := filepath.Join(workDir, fmt.Sprintf("output_%d.txt", testCase.CaseId))
	errorFile := filepath.Join(workDir, fmt.Sprintf("error_%d.txt", testCase.CaseId))

	// 写入测试输入
	if err := os.WriteFile(inputFile, []byte(testCase.Input), 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// 应用语言特定的资源限制倍数
	adjustedTimeLimit := int64(float64(timeLimit) * executor.GetTimeMultiplier())
	adjustedMemoryLimit := int64(float64(memoryLimit) * executor.GetMemoryMultiplier() * 1024) // 转换为KB

	// 配置执行参数
	execConfig := &languages.ExecutionConfig{
		TimeLimit:   adjustedTimeLimit,
		MemoryLimit: adjustedMemoryLimit,
		InputFile:   inputFile,
		OutputFile:  outputFile,
		ErrorFile:   errorFile,
		Environment: []string{"PATH=/usr/bin:/bin"},
	}

	// 执行程序
	execResult, err := executor.Execute(ctx, executablePath, workDir, execConfig)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// 读取程序输出
	var output, errorOutput string
	if outputData, err := os.ReadFile(outputFile); err == nil {
		output = string(outputData)
	}
	if errorData, err := os.ReadFile(errorFile); err == nil {
		errorOutput = string(errorData)
	}

	// 添加调试日志
	logx.Infof("Debug: ErrorOutput length=%d, content='%s'", len(errorOutput), errorOutput)
	logx.Infof("Debug: ExecResult.ErrorOutput length=%d, content='%s'", len(execResult.ErrorOutput), execResult.ErrorOutput)

	// 创建测试用例结果
	result := &types.TestCaseResult{
		CaseId:      testCase.CaseId,
		TimeUsed:    int(execResult.TimeUsed),
		MemoryUsed:  int(execResult.MemoryUsed),
		Input:       testCase.Input,
		Output:      strings.TrimSpace(output),
		Expected:    strings.TrimSpace(testCase.ExpectedOutput),
		ErrorOutput: errorOutput,
	}

	// 确定执行状态
	result.Status = je.determineTestCaseStatus(execResult, result)

	return result, nil
}

// 确定测试用例状态
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

	case sandbox.StatusCompileError:
		return "compile_error"

	default:
		return "system_error"
	}
}

// 比较输出结果
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

// 确定最终状态
func (je *JudgeEngine) determineFinalStatus(testCases []types.TestCaseResult) string {
	if len(testCases) == 0 {
		return "system_error"
	}

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
		case "runtime_error":
			hasRuntimeError = true
		case "time_limit_exceeded":
			hasTimeLimitExceeded = true
		case "memory_limit_exceeded":
			hasMemoryLimitExceeded = true
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
	if hasTimeLimitExceeded {
		return "time_limit_exceeded"
	}
	if hasMemoryLimitExceeded {
		return "memory_limit_exceeded"
	}
	if hasWrongAnswer {
		return "wrong_answer"
	}

	return "system_error"
}

// 创建临时目录
func (je *JudgeEngine) createTempDir(submissionID int64) (string, error) {
	tempDir := filepath.Join(je.tempDir, fmt.Sprintf("judge_%d_%d",
		submissionID, time.Now().UnixNano()))

	if err := os.MkdirAll(tempDir, 0777); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// 将目录所有权改为nobody用户，确保沙箱环境中的编译器能够写入
	if err := os.Chown(tempDir, 65534, 65534); err != nil {
		return "", fmt.Errorf("failed to change temp directory ownership: %w", err)
	}

	return tempDir, nil
}

// 清理临时目录
func (je *JudgeEngine) cleanupTempDir(tempDir string) {
	if err := os.RemoveAll(tempDir); err != nil {
		logx.Errorf("Failed to cleanup temp directory %s: %v", tempDir, err)
	}
}

// 获取服务器ID
func (je *JudgeEngine) getServerID() string {
	// TODO: 从配置或环境变量获取
	hostname, _ := os.Hostname()
	return hostname
}

// 健康检查
func (je *JudgeEngine) HealthCheck() error {
	// 检查工作目录是否可写
	testFile := filepath.Join(je.tempDir, "health_check")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("work directory not writable: %w", err)
	}
	os.Remove(testFile)

	// 检查语言执行器
	supportedLanguages := je.languageManager.GetSupportedLanguages()
	if len(supportedLanguages) == 0 {
		return fmt.Errorf("no language executors available")
	}

	return nil
}

// 获取系统信息
func (je *JudgeEngine) GetSystemInfo() map[string]interface{} {
	return map[string]interface{}{
		"work_dir":            je.workDir,
		"temp_dir":            je.tempDir,
		"supported_languages": je.languageManager.GetSupportedLanguages(),
		"language_configs":    je.languageManager.GetLanguageConfigs(),
		"system_info":         sandbox.GetSystemInfo(),
	}
}

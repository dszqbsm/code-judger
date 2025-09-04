package judge

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/scheduler"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitJudgeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSubmitJudgeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitJudgeLogic {
	return &SubmitJudgeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubmitJudgeLogic) SubmitJudge(req *types.SubmitJudgeReq) (resp *types.SubmitJudgeResp, err error) {
	l.Logger.Infof("开始处理判题请求: SubmissionID=%d, ProblemID=%d, UserID=%d, Language=%s",
		req.SubmissionId, req.ProblemId, req.UserId, req.Language)

	// 1. 验证基本请求参数
	if err := l.validateBasicRequest(req); err != nil {
		l.Logger.Errorf("请求参数验证失败: %v", err)
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 2. 从题目服务获取题目详细信息（真实业务逻辑）
	problemInfo, err := l.getProblemDetailFromService(req.ProblemId)
	if err != nil {
		l.Logger.Errorf("获取题目信息失败: ProblemID=%d, Error=%v", req.ProblemId, err)
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: fmt.Sprintf("获取题目信息失败: %s", err.Error()),
			},
		}, nil
	}

	l.Logger.Infof("成功获取题目信息: ProblemID=%d, Title=%s, TimeLimit=%dms, MemoryLimit=%dMB, TestCases=%d",
		problemInfo.ProblemId, problemInfo.Title, problemInfo.TimeLimit, problemInfo.MemoryLimit, len(problemInfo.TestCases))

	// 3. 验证编程语言支持（题目业务限制 + 系统技术限制）
	if err := l.validateLanguageSupport(req.Language, problemInfo); err != nil {
		l.Logger.Errorf("语言支持验证失败: Language=%s, Error=%v", req.Language, err)
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 4. 验证代码安全性
	if err := l.validateCodeSecurity(req.Code, req.Language); err != nil {
		l.Logger.Errorf("代码安全验证失败: Language=%s, Error=%v", req.Language, err)
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 5. 转换测试用例（值类型 -> 指针类型）
	testCases := l.convertTestCases(problemInfo.TestCases)

	// 6. 创建判题任务（使用题目的时间和内存限制）
	task := &scheduler.JudgeTask{
		SubmissionID: req.SubmissionId,
		ProblemID:    req.ProblemId,
		UserID:       req.UserId,
		Language:     req.Language,
		Code:         req.Code,
		TimeLimit:    problemInfo.TimeLimit,   // 从题目服务获取
		MemoryLimit:  problemInfo.MemoryLimit, // 从题目服务获取
		TestCases:    testCases,               // 从题目服务获取
		Priority:     l.determinePriority(req.UserId),
	}

	// 7. 提交任务到调度器
	if err := l.svcCtx.TaskScheduler.SubmitTask(task); err != nil {
		l.Logger.Errorf("提交判题任务失败: %v", err)
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "提交判题任务失败，请稍后重试",
			},
		}, nil
	}

	// 8. 从调度器获取任务在队列中的实际位置
	taskPosition, err := l.getTaskQueuePositionFromScheduler(task.ID)
	if err != nil {
		l.Logger.Errorf("获取任务队列位置失败: TaskID=%s, Error=%v", task.ID, err)
		// 使用队列长度作为备用值
		queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()
		taskPosition = queueStatus.QueueLength
	}

	// 9. 计算预估等待时间
	estimatedTime := l.calculateEstimatedWaitTime(taskPosition)

	// 10. 记录成功日志
	l.Logger.Infof("判题任务提交成功: SubmissionID=%d, TaskID=%s, ProblemID=%d, Language=%s, QueuePosition=%d, EstimatedTime=%ds",
		req.SubmissionId, task.ID, req.ProblemId, req.Language, taskPosition, estimatedTime)

	return &types.SubmitJudgeResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "判题任务已提交",
		},
		Data: types.SubmitJudgeData{
			SubmissionId:  req.SubmissionId,
			Status:        "pending",
			QueuePosition: taskPosition,
			EstimatedTime: estimatedTime,
		},
	}, nil
}

// getProblemDetailFromService 从题目服务获取题目详细信息（真实业务逻辑）
func (l *SubmitJudgeLogic) getProblemDetailFromService(problemId int64) (*types.ProblemInfo, error) {
	// 记录开始时间
	startTime := time.Now()

	// 调用题目服务客户端
	problemInfo, err := l.svcCtx.ProblemClient.GetProblemDetail(l.ctx, problemId)
	if err != nil {
		return nil, fmt.Errorf("调用题目服务失败: %w", err)
	}

	// 记录调用耗时
	duration := time.Since(startTime)
	l.Logger.Infof("题目服务调用成功: ProblemID=%d, Duration=%v", problemId, duration)

	// 验证题目信息的完整性和合理性
	if err := l.validateProblemInfo(problemInfo); err != nil {
		return nil, fmt.Errorf("题目信息验证失败: %w", err)
	}

	return problemInfo, nil
}

// validateProblemInfo 验证题目信息的完整性和合理性
func (l *SubmitJudgeLogic) validateProblemInfo(problemInfo *types.ProblemInfo) error {
	if problemInfo == nil {
		return fmt.Errorf("题目信息为空")
	}

	if problemInfo.ProblemId <= 0 {
		return fmt.Errorf("无效的题目ID: %d", problemInfo.ProblemId)
	}

	if problemInfo.Title == "" {
		return fmt.Errorf("题目标题为空")
	}

	if problemInfo.TimeLimit <= 0 || problemInfo.TimeLimit > 30000 {
		return fmt.Errorf("时间限制不合理: %dms（应在1-30000ms之间）", problemInfo.TimeLimit)
	}

	if problemInfo.MemoryLimit <= 0 || problemInfo.MemoryLimit > 1024 {
		return fmt.Errorf("内存限制不合理: %dMB（应在1-1024MB之间）", problemInfo.MemoryLimit)
	}

	if len(problemInfo.Languages) == 0 {
		return fmt.Errorf("支持的编程语言列表为空")
	}

	if len(problemInfo.TestCases) == 0 {
		return fmt.Errorf("测试用例为空")
	}

	// 验证测试用例
	for i, testCase := range problemInfo.TestCases {
		if testCase.Input == "" && testCase.ExpectedOutput == "" {
			return fmt.Errorf("测试用例 %d 的输入和输出都为空", i+1)
		}
	}

	return nil
}

// getTaskQueuePositionFromScheduler 从调度器获取任务位置的方法（真实业务逻辑）
func (l *SubmitJudgeLogic) getTaskQueuePositionFromScheduler(taskID string) (int, error) {
	// 方法1: 直接从调度器获取任务位置（如果调度器提供了这个方法）
	if position, err := l.svcCtx.TaskScheduler.GetTaskPosition(taskID); err == nil {
		return position, nil
	}

	// 方法2: 通过队列状态查找任务位置
	queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()

	// 解析任务ID中的SubmissionID
	submissionID, err := l.extractSubmissionIDFromTaskID(taskID)
	if err != nil {
		return 0, fmt.Errorf("解析任务ID失败: %w", err)
	}

	// 在队列中查找任务位置
	for i, queueItem := range queueStatus.QueueItems {
		if queueItem.SubmissionID == submissionID {
			return i + 1, nil // 返回1基的位置（第1个、第2个...）
		}
	}

	// 如果在队列中没找到，可能任务已经开始执行或者刚刚提交还未进入队列
	// 返回队列长度+1作为估算位置
	return queueStatus.QueueLength + 1, nil
}

// extractSubmissionIDFromTaskID 从任务ID中提取提交ID
func (l *SubmitJudgeLogic) extractSubmissionIDFromTaskID(taskID string) (int64, error) {
	// 任务ID格式：task_123_1234567890 或 submission_123
	parts := strings.Split(taskID, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("无效的任务ID格式: %s", taskID)
	}

	// 尝试解析第二部分作为SubmissionID
	submissionID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析SubmissionID失败: %w", err)
	}

	return submissionID, nil
}

// calculateEstimatedWaitTime 计算预估等待时间（基于真实队列状态）
func (l *SubmitJudgeLogic) calculateEstimatedWaitTime(queuePosition int) int {
	if queuePosition <= 0 {
		return 0
	}

	// 获取系统配置
	avgTaskTime := l.svcCtx.Config.TaskQueue.AverageTaskTime
	if avgTaskTime <= 0 {
		avgTaskTime = 30 // 默认30秒
	}

	workerCount := l.svcCtx.Config.TaskQueue.MaxWorkers
	if workerCount <= 0 {
		workerCount = 1
	}

	// 获取当前系统负载情况
	queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()
	systemLoad := float64(queueStatus.RunningTasks) / float64(workerCount)

	// 根据系统负载调整预估时间
	loadMultiplier := 1.0
	if systemLoad > 0.8 {
		loadMultiplier = 1.5 // 高负载时增加50%的时间
	} else if systemLoad > 0.5 {
		loadMultiplier = 1.2 // 中负载时增加20%的时间
	}

	// 计算基础等待时间：(位置-1) / 工作器数 * 平均时间
	baseWaitTime := ((queuePosition - 1) / workerCount) * avgTaskTime

	// 应用负载调整
	estimatedTime := int(float64(baseWaitTime) * loadMultiplier)

	// 最小等待时间为平均任务时间
	if estimatedTime < avgTaskTime {
		estimatedTime = avgTaskTime
	}

	return estimatedTime
}

// validateBasicRequest 验证基本请求参数
func (l *SubmitJudgeLogic) validateBasicRequest(req *types.SubmitJudgeReq) error {
	if req.SubmissionId <= 0 {
		return fmt.Errorf("无效的提交ID: %d", req.SubmissionId)
	}

	if req.ProblemId <= 0 {
		return fmt.Errorf("无效的题目ID: %d", req.ProblemId)
	}

	if req.UserId <= 0 {
		return fmt.Errorf("无效的用户ID: %d", req.UserId)
	}

	if req.Language == "" {
		return fmt.Errorf("编程语言不能为空")
	}

	if req.Code == "" {
		return fmt.Errorf("代码不能为空")
	}

	maxCodeLength := l.svcCtx.Config.JudgeEngine.Security.MaxCodeLength
	if len(req.Code) > maxCodeLength {
		return fmt.Errorf("代码长度超出限制: %d > %d", len(req.Code), maxCodeLength)
	}

	return nil
}

// validateLanguageSupport 验证编程语言支持（包含题目业务限制和系统技术限制）
func (l *SubmitJudgeLogic) validateLanguageSupport(language string, problemInfo *types.ProblemInfo) error {
	// 1. 题目业务限制验证（优先检查，提供更具体的错误信息）
	if !l.isLanguageSupportedByProblem(language, problemInfo.Languages) {
		return fmt.Errorf("该题目不支持 %s 语言，支持的语言：%v", language, problemInfo.Languages)
	}

	// 2. 系统技术限制验证
	systemInfo := l.svcCtx.JudgeEngine.GetSystemInfo()
	supportedLanguages, ok := systemInfo["supported_languages"].([]string)
	if !ok {
		l.Logger.Errorf("获取系统支持的语言列表失败")
		// 如果无法获取系统支持的语言，只进行题目级别的验证
		return nil
	}

	if !l.isLanguageSupported(language, supportedLanguages) {
		return fmt.Errorf("判题系统暂不支持 %s 语言（可能在维护中），系统支持的语言：%v", language, supportedLanguages)
	}

	return nil
}

// validateCodeSecurity 验证代码安全性
func (l *SubmitJudgeLogic) validateCodeSecurity(code, language string) error {
	// 检查禁止的代码模式
	forbiddenPatterns := l.svcCtx.Config.JudgeEngine.Security.ForbiddenPatterns
	for _, pattern := range forbiddenPatterns {
		if pattern == "" {
			continue
		}

		// 简单的字符串包含检查（实际应该使用正则表达式）
		if strings.Contains(strings.ToLower(code), strings.ToLower(pattern)) {
			return fmt.Errorf("代码包含禁止的模式: %s", pattern)
		}
	}

	// 语言特定的安全检查
	switch language {
	case "c", "cpp":
		if err := l.validateCCppSecurity(code); err != nil {
			return err
		}
	case "java":
		if err := l.validateJavaSecurity(code); err != nil {
			return err
		}
	case "python":
		if err := l.validatePythonSecurity(code); err != nil {
			return err
		}
	}

	return nil
}

// validateCCppSecurity C/C++代码安全检查
func (l *SubmitJudgeLogic) validateCCppSecurity(code string) error {
	dangerousPatterns := []string{
		"system(",
		"exec(",
		"fork(",
		"#include <unistd.h>",
		"#include <sys/",
		"asm(",
		"__asm__",
	}

	lowerCode := strings.ToLower(code)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerCode, strings.ToLower(pattern)) {
			return fmt.Errorf("C/C++代码包含危险操作: %s", pattern)
		}
	}

	return nil
}

// validateJavaSecurity Java代码安全检查
func (l *SubmitJudgeLogic) validateJavaSecurity(code string) error {
	dangerousPatterns := []string{
		"Runtime.getRuntime()",
		"ProcessBuilder",
		"System.exit(",
		"java.lang.reflect",
		"java.io.File",
		"java.net.Socket",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(code, pattern) {
			return fmt.Errorf("Java代码包含危险操作: %s", pattern)
		}
	}

	return nil
}

// validatePythonSecurity Python代码安全检查
func (l *SubmitJudgeLogic) validatePythonSecurity(code string) error {
	dangerousPatterns := []string{
		"import os",
		"import subprocess",
		"__import__",
		"eval(",
		"exec(",
		"compile(",
		"open(",
	}

	lowerCode := strings.ToLower(code)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerCode, strings.ToLower(pattern)) {
			return fmt.Errorf("Python代码包含危险操作: %s", pattern)
		}
	}

	return nil
}

// convertTestCases 转换测试用例（值类型 -> 指针类型）
func (l *SubmitJudgeLogic) convertTestCases(testCases []types.TestCase) []*types.TestCase {
	result := make([]*types.TestCase, len(testCases))
	for i, tc := range testCases {
		result[i] = &types.TestCase{
			CaseId:         tc.CaseId,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			TimeLimit:      tc.TimeLimit,
			MemoryLimit:    tc.MemoryLimit,
		}
	}
	return result
}

// isLanguageSupportedByProblem 检查语言是否被题目支持
func (l *SubmitJudgeLogic) isLanguageSupportedByProblem(language string, problemLanguages []string) bool {
	for _, lang := range problemLanguages {
		if lang == language {
			return true
		}
	}
	return false
}

// isLanguageSupported 检查语言是否被判题引擎支持
func (l *SubmitJudgeLogic) isLanguageSupported(language string, supportedLanguages []string) bool {
	for _, lang := range supportedLanguages {
		if lang == language {
			return true
		}
	}
	return false
}

// determinePriority 确定任务优先级
func (l *SubmitJudgeLogic) determinePriority(userID int64) int {
	// TODO: 根据用户类型、VIP状态等确定优先级
	// 可以调用用户服务获取用户信息
	// userInfo, err := l.svcCtx.UserClient.GetUser(l.ctx, userID)
	// if err == nil {
	//     if userInfo.IsVIP {
	//         return scheduler.PriorityNormal
	//     }
	//     if userInfo.Role == "admin" || userInfo.Role == "teacher" {
	//         return scheduler.PriorityNormal
	//     }
	// }

	// 默认普通优先级
	return scheduler.PriorityLow
}

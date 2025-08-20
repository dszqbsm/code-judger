package judge

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/online-judge/code-judger/services/judge-api/internal/scheduler"
	"github.com/online-judge/code-judger/services/judge-api/internal/svc"
	"github.com/online-judge/code-judger/services/judge-api/internal/types"

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
	// 1. 验证基本请求参数
	if err := l.validateBasicRequest(req); err != nil {
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 2. 从题目服务获取题目详细信息
	problemInfo, err := l.svcCtx.ProblemClient.GetProblemDetail(l.ctx, req.ProblemId)
	if err != nil {
		logx.Errorf("Failed to get problem detail for problem_id=%d: %v", req.ProblemId, err)
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: fmt.Sprintf("获取题目信息失败: %s", err.Error()),
			},
		}, nil
	}

	// 3. 验证编程语言支持（题目业务限制 + 系统技术限制）
	if err := l.validateLanguageSupport(req.Language, problemInfo); err != nil {
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 5. 转换测试用例（值类型 -> 指针类型）
	// 目的：
	// 1. 类型适配：ProblemInfo.TestCases([]TestCase) -> JudgeTask.TestCases([]*TestCase)
	// 2. 内存优化：指针类型避免大量数据复制
	// 3. 状态更新：判题过程中可以直接修改测试用例状态
	testCases := make([]*types.TestCase, len(problemInfo.TestCases))
	for i, tc := range problemInfo.TestCases {
		testCases[i] = &types.TestCase{
			CaseId:         tc.CaseId,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			TimeLimit:      tc.TimeLimit,
			MemoryLimit:    tc.MemoryLimit,
		}
	}

	// 6. 创建判题任务（使用题目的时间和内存限制）
	task := &scheduler.JudgeTask{
		SubmissionID: req.SubmissionId,
		ProblemID:    req.ProblemId,
		UserID:       req.UserId,
		Language:     req.Language,
		Code:         req.Code,
		TimeLimit:    problemInfo.TimeLimit,   // 从题目信息获取
		MemoryLimit:  problemInfo.MemoryLimit, // 从题目信息获取
		TestCases:    testCases,               // 从题目信息获取
		Priority:     l.determinePriority(req.UserId),
	}

	// 7. 提交任务到调度器
	if err := l.svcCtx.TaskScheduler.SubmitTask(task); err != nil {
		logx.Errorf("Failed to submit judge task: %v", err)
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "提交判题任务失败",
			},
		}, nil
	}

	// 8. 获取当前任务在队列中的实际位置
	taskPosition, err := l.getTaskQueuePosition(task.ID)
	if err != nil {
		// 如果无法获取精确位置，使用队列长度作为近似值
		queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()
		taskPosition = queueStatus.QueueLength
		logx.WithContext(l.ctx).Infof("Warning: Cannot get exact task position, using queue length: %d", taskPosition)
	}

	// 9. 记录操作日志
	logx.Infof("Judge task submitted: submission_id=%d, problem_id=%d, language=%s, time_limit=%dms, memory_limit=%dMB, test_cases=%d, queue_position=%d",
		req.SubmissionId, req.ProblemId, req.Language, problemInfo.TimeLimit, problemInfo.MemoryLimit, len(testCases), taskPosition)

	return &types.SubmitJudgeResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "判题任务已提交",
		},
		Data: types.SubmitJudgeData{
			SubmissionId:  req.SubmissionId,
			Status:        "pending",
			QueuePosition: taskPosition,                     // 使用实际队列位置
			EstimatedTime: l.estimateWaitTime(taskPosition), // 基于实际位置计算等待时间
		},
	}, nil
}

// 验证基本请求参数
func (l *SubmitJudgeLogic) validateBasicRequest(req *types.SubmitJudgeReq) error {
	if req.SubmissionId <= 0 {
		return fmt.Errorf("无效的提交ID")
	}

	if req.ProblemId <= 0 {
		return fmt.Errorf("无效的题目ID")
	}

	if req.UserId <= 0 {
		return fmt.Errorf("无效的用户ID")
	}

	if req.Language == "" {
		return fmt.Errorf("编程语言不能为空")
	}

	if req.Code == "" {
		return fmt.Errorf("代码不能为空")
	}

	if len(req.Code) > l.svcCtx.Config.JudgeEngine.Security.MaxCodeLength {
		return fmt.Errorf("代码长度超出限制")
	}

	// 检查禁止的代码模式
	for _, pattern := range l.svcCtx.Config.JudgeEngine.Security.ForbiddenPatterns {
		if len(pattern) > 0 && len(req.Code) > 0 {
			// 这里应该使用正则表达式匹配，简化处理
			// TODO: 实现正则表达式匹配
		}
	}

	return nil
}

// 检查语言是否被判题引擎支持
func (l *SubmitJudgeLogic) isLanguageSupported(language string, supportedLanguages []string) bool {
	for _, lang := range supportedLanguages {
		if lang == language {
			return true
		}
	}
	return false
}

// 验证编程语言支持（包含题目业务限制和系统技术限制）
func (l *SubmitJudgeLogic) validateLanguageSupport(language string, problemInfo *types.ProblemInfo) error {
	// 1. 题目业务限制验证（优先检查，提供更具体的错误信息）
	if !l.isLanguageSupportedByProblem(language, problemInfo.Languages) {
		return fmt.Errorf("该题目不支持 %s 语言，支持的语言：%v",
			language, problemInfo.Languages)
	}

	// 2. 系统技术限制验证
	supportedLanguages := l.svcCtx.JudgeEngine.GetSystemInfo()["supported_languages"].([]string)
	if !l.isLanguageSupported(language, supportedLanguages) {
		return fmt.Errorf("判题系统暂不支持 %s 语言（可能在维护中），系统支持的语言：%v",
			language, supportedLanguages)
	}

	return nil
}

// 检查语言是否被题目支持
func (l *SubmitJudgeLogic) isLanguageSupportedByProblem(language string, problemLanguages []string) bool {
	for _, lang := range problemLanguages {
		if lang == language {
			return true
		}
	}
	return false
}

// 确定任务优先级
func (l *SubmitJudgeLogic) determinePriority(userID int64) int {
	// TODO: 根据用户类型、VIP状态等确定优先级
	// 这里简化处理，所有任务都是普通优先级
	return scheduler.PriorityNormal
}

// 获取任务在队列中的实际位置
func (l *SubmitJudgeLogic) getTaskQueuePosition(taskID string) (int, error) {
	// 注意：这里需要调度器提供获取任务位置的方法
	// 当前简化实现，实际应该在scheduler中添加GetTaskPosition方法

	// 临时方案：通过遍历队列状态查找任务位置
	queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()

	for i, queueItem := range queueStatus.QueueItems {
		if queueItem.SubmissionID == extractSubmissionID(taskID) {
			return i + 1, nil // 返回1基的位置（第1个、第2个...）
		}
	}

	return queueStatus.QueueLength, fmt.Errorf("task not found in queue, using queue length as fallback")
}

// 从任务ID提取提交ID的辅助函数
func extractSubmissionID(taskID string) int64 {
	// 任务ID格式：task_123_1234567890
	// 提取submission_id部分
	parts := strings.Split(taskID, "_")
	if len(parts) >= 2 {
		if submissionID, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			return submissionID
		}
	}
	return 0
}

// 估算等待时间
func (l *SubmitJudgeLogic) estimateWaitTime(queuePosition int) int {
	// 假设每个任务平均执行时间为30秒
	avgTaskTime := 30
	workerCount := l.svcCtx.Config.TaskQueue.MaxWorkers

	if workerCount <= 0 {
		workerCount = 1
	}

	// 修正计算公式：考虑并行处理
	if queuePosition <= 0 {
		return 0
	}

	// 计算等待时间：(位置-1) / 工作器数 * 平均时间
	// 例：第4个任务，3个工作器 → (4-1)/3 * 30 = 30秒
	return ((queuePosition - 1) / workerCount) * avgTaskTime
}

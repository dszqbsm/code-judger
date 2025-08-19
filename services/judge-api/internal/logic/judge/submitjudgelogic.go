package judge

import (
	"context"
	"fmt"

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
	// 验证请求参数
	if err := l.validateRequest(req); err != nil {
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 检查语言是否支持
	supportedLanguages := l.svcCtx.JudgeEngine.GetSystemInfo()["supported_languages"].([]string)
	if !l.isLanguageSupported(req.Language, supportedLanguages) {
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: fmt.Sprintf("不支持的编程语言: %s", req.Language),
			},
		}, nil
	}

	// 转换测试用例
	testCases := make([]*types.TestCase, len(req.TestCases))
	for i, tc := range req.TestCases {
		testCases[i] = &types.TestCase{
			CaseId:         tc.CaseId,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			TimeLimit:      tc.TimeLimit,
			MemoryLimit:    tc.MemoryLimit,
		}
	}

	// 创建判题任务
	task := &scheduler.JudgeTask{
		SubmissionID: req.SubmissionId,
		ProblemID:    req.ProblemId,
		UserID:       req.UserId,
		Language:     req.Language,
		Code:         req.Code,
		TimeLimit:    req.TimeLimit,
		MemoryLimit:  req.MemoryLimit,
		TestCases:    testCases,
		Priority:     l.determinePriority(req.UserId), // 根据用户类型确定优先级
	}

	// 提交任务到调度器
	if err := l.svcCtx.TaskScheduler.SubmitTask(task); err != nil {
		logx.Errorf("Failed to submit judge task: %v", err)
		return &types.SubmitJudgeResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "提交判题任务失败",
			},
		}, nil
	}

	// 获取队列状态以计算等待时间
	queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()

	// 记录操作日志
	logx.Infof("Judge task submitted: submission_id=%d, language=%s, queue_position=%d",
		req.SubmissionId, req.Language, queueStatus.QueueLength)

	return &types.SubmitJudgeResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "判题任务已提交",
		},
		Data: types.SubmitJudgeData{
			SubmissionId:  req.SubmissionId,
			Status:        "pending",
			QueuePosition: queueStatus.QueueLength,
			EstimatedTime: l.estimateWaitTime(queueStatus.QueueLength),
		},
	}, nil
}

// 验证请求参数
func (l *SubmitJudgeLogic) validateRequest(req *types.SubmitJudgeReq) error {
	if req.SubmissionId <= 0 {
		return fmt.Errorf("无效的提交ID")
	}

	if req.ProblemId <= 0 {
		return fmt.Errorf("无效的题目ID")
	}

	if req.UserId <= 0 {
		return fmt.Errorf("无效的用户ID")
	}

	if req.Code == "" {
		return fmt.Errorf("代码不能为空")
	}

	if len(req.Code) > l.svcCtx.Config.JudgeEngine.Security.MaxCodeLength {
		return fmt.Errorf("代码长度超出限制")
	}

	if req.TimeLimit < 100 || req.TimeLimit > 10000 {
		return fmt.Errorf("时间限制必须在100-10000毫秒之间")
	}

	if req.MemoryLimit < 16 || req.MemoryLimit > 512 {
		return fmt.Errorf("内存限制必须在16-512MB之间")
	}

	if len(req.TestCases) == 0 {
		return fmt.Errorf("测试用例不能为空")
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

// 检查语言是否支持
func (l *SubmitJudgeLogic) isLanguageSupported(language string, supportedLanguages []string) bool {
	for _, lang := range supportedLanguages {
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

// 估算等待时间
func (l *SubmitJudgeLogic) estimateWaitTime(queuePosition int) int {
	// 假设每个任务平均执行时间为30秒
	avgTaskTime := 30
	workerCount := l.svcCtx.Config.TaskQueue.MaxWorkers

	if workerCount <= 0 {
		workerCount = 1
	}

	return (queuePosition / workerCount) * avgTaskTime
}

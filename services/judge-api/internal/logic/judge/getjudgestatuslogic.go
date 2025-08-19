package judge

import (
	"context"
	"fmt"

	"github.com/online-judge/code-judger/services/judge-api/internal/scheduler"
	"github.com/online-judge/code-judger/services/judge-api/internal/svc"
	"github.com/online-judge/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetJudgeStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetJudgeStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetJudgeStatusLogic {
	return &GetJudgeStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetJudgeStatusLogic) GetJudgeStatus(req *types.GetJudgeStatusReq) (resp *types.GetJudgeStatusResp, err error) {
	// 验证提交ID
	if req.SubmissionId <= 0 {
		return &types.GetJudgeStatusResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: "无效的提交ID",
			},
		}, nil
	}

	// 查找任务
	task, err := l.findTaskBySubmissionId(req.SubmissionId)
	if err != nil {
		logx.Errorf("Failed to find task for submission %d: %v", req.SubmissionId, err)
		return &types.GetJudgeStatusResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "未找到判题任务",
			},
		}, nil
	}

	// 计算进度
	progress := l.calculateProgress(task)

	// 确定当前测试用例和总数
	currentTestCase := 0
	totalTestCases := len(task.TestCases)

	if task.Result != nil {
		currentTestCase = len(task.Result.TestCases)
	}

	// 生成状态消息
	message := l.generateStatusMessage(task.Status, currentTestCase, totalTestCases)

	logx.Infof("Retrieved judge status for submission %d: status=%s, progress=%d%%",
		req.SubmissionId, task.Status, progress)

	return &types.GetJudgeStatusResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "获取成功",
		},
		Data: types.JudgeStatus{
			SubmissionId:    req.SubmissionId,
			Status:          task.Status,
			Progress:        progress,
			CurrentTestCase: currentTestCase,
			TotalTestCases:  totalTestCases,
			Message:         message,
		},
	}, nil
}

// 根据提交ID查找任务
func (l *GetJudgeStatusLogic) findTaskBySubmissionId(submissionId int64) (*scheduler.JudgeTask, error) {
	// TODO: 实现高效的任务查找
	// 这里简化实现，返回模拟数据
	return &scheduler.JudgeTask{
		SubmissionID: submissionId,
		Status:       "running",
		TestCases: []*types.TestCase{
			{CaseId: 1, Input: "test1", ExpectedOutput: "output1"},
			{CaseId: 2, Input: "test2", ExpectedOutput: "output2"},
			{CaseId: 3, Input: "test3", ExpectedOutput: "output3"},
		},
		Result: &types.JudgeResult{
			TestCases: []types.TestCaseResult{
				{CaseId: 1, Status: "accepted"},
			},
		},
	}, nil
}

// 计算判题进度
func (l *GetJudgeStatusLogic) calculateProgress(task *scheduler.JudgeTask) int {
	switch task.Status {
	case "pending":
		return 0
	case "running":
		if task.Result != nil && len(task.TestCases) > 0 {
			completed := len(task.Result.TestCases)
			total := len(task.TestCases)
			return (completed * 100) / total
		}
		return 10 // 至少10%表示已开始
	case "completed":
		return 100
	case "failed", "cancelled":
		return 100
	default:
		return 0
	}
}

// 生成状态消息
func (l *GetJudgeStatusLogic) generateStatusMessage(status string, current, total int) string {
	switch status {
	case "pending":
		return "等待判题中..."
	case "running":
		if current > 0 {
			return fmt.Sprintf("正在执行测试用例 %d/%d", current, total)
		}
		return "正在编译代码..."
	case "completed":
		return "判题完成"
	case "failed":
		return "判题失败"
	case "cancelled":
		return "判题已取消"
	default:
		return "未知状态"
	}
}

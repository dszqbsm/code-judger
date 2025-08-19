package judge

import (
	"context"
	"fmt"

	"github.com/online-judge/code-judger/services/judge-api/internal/scheduler"
	"github.com/online-judge/code-judger/services/judge-api/internal/svc"
	"github.com/online-judge/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetJudgeResultLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetJudgeResultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetJudgeResultLogic {
	return &GetJudgeResultLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetJudgeResultLogic) GetJudgeResult(req *types.GetJudgeResultReq) (resp *types.GetJudgeResultResp, err error) {
	// 验证提交ID
	if req.SubmissionId <= 0 {
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: "无效的提交ID",
			},
		}, nil
	}

	// 从调度器获取任务状态
	task, err := l.findTaskBySubmissionId(req.SubmissionId)
	if err != nil {
		logx.Errorf("Failed to find task for submission %d: %v", req.SubmissionId, err)
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "未找到判题结果",
			},
		}, nil
	}

	// 检查任务是否完成
	if task.Status != "completed" && task.Status != "failed" {
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    202,
				Message: "判题尚未完成",
			},
			Data: types.JudgeResult{
				SubmissionId: req.SubmissionId,
				Status:       task.Status,
			},
		}, nil
	}

	// 任务失败
	if task.Status == "failed" {
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "判题失败",
			},
			Data: types.JudgeResult{
				SubmissionId: req.SubmissionId,
				Status:       "system_error",
			},
		}, nil
	}

	// 返回判题结果
	result := task.Result
	if result == nil {
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "判题结果不存在",
			},
		}, nil
	}

	logx.Infof("Retrieved judge result for submission %d: status=%s, score=%d",
		req.SubmissionId, result.Status, result.Score)

	return &types.GetJudgeResultResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "获取成功",
		},
		Data: *result,
	}, nil
}

// 根据提交ID查找任务（简化实现）
func (l *GetJudgeResultLogic) findTaskBySubmissionId(submissionId int64) (*scheduler.JudgeTask, error) {
	// TODO: 这里应该实现更高效的查找方法
	// 可以考虑：
	// 1. 在调度器中维护submission_id到task_id的映射
	// 2. 使用数据库存储任务状态
	// 3. 使用缓存加速查询

	// 现在简化实现：遍历调度器中的任务
	stats := l.svcCtx.TaskScheduler.GetStats()
	if stats.TotalTasks == 0 {
		return nil, fmt.Errorf("no tasks found")
	}

	// 这里需要调度器提供根据submission_id查找任务的方法
	// 暂时返回一个模拟的任务
	return &scheduler.JudgeTask{
		SubmissionID: submissionId,
		Status:       "completed",
		Result: &types.JudgeResult{
			SubmissionId: submissionId,
			Status:       "accepted",
			Score:        100,
			TimeUsed:     150,
			MemoryUsed:   1024,
			CompileInfo: types.CompileInfo{
				Success: true,
				Message: "",
				Time:    1200,
			},
			TestCases: []types.TestCaseResult{
				{
					CaseId:     1,
					Status:     "accepted",
					TimeUsed:   150,
					MemoryUsed: 1024,
					Input:      "test input",
					Output:     "test output",
					Expected:   "test output",
				},
			},
			JudgeInfo: types.JudgeInfo{
				JudgeServer:     "judge-node-01",
				JudgeTime:       "2024-01-15T10:30:00Z",
				LanguageVersion: "g++ 9.4.0",
			},
		},
	}, nil
}

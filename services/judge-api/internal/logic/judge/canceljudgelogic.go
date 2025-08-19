package judge

import (
	"context"
	"fmt"

	"github.com/online-judge/code-judger/services/judge-api/internal/scheduler"
	"github.com/online-judge/code-judger/services/judge-api/internal/svc"
	"github.com/online-judge/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelJudgeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelJudgeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelJudgeLogic {
	return &CancelJudgeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelJudgeLogic) CancelJudge(req *types.CancelJudgeReq) (resp *types.CancelJudgeResp, err error) {
	// 验证提交ID
	if req.SubmissionId <= 0 {
		return &types.CancelJudgeResp{
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
		return &types.CancelJudgeResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "未找到判题任务",
			},
		}, nil
	}

	// 检查任务状态
	if task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
		return &types.CancelJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: fmt.Sprintf("任务已%s，无法取消", l.getStatusText(task.Status)),
			},
		}, nil
	}

	// 生成任务ID进行取消
	taskID := fmt.Sprintf("task_%d_*", req.SubmissionId)

	// 调用调度器取消任务
	if err := l.svcCtx.TaskScheduler.CancelTask(taskID); err != nil {
		logx.Errorf("Failed to cancel task %s: %v", taskID, err)
		return &types.CancelJudgeResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "取消判题任务失败",
			},
		}, nil
	}

	logx.Infof("Judge task cancelled: submission_id=%d", req.SubmissionId)

	return &types.CancelJudgeResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "判题任务已取消",
		},
		Data: types.CancelJudgeData{
			SubmissionId: req.SubmissionId,
			Status:       "cancelled",
			Message:      "任务已成功取消",
		},
	}, nil
}

// 根据提交ID查找任务
func (l *CancelJudgeLogic) findTaskBySubmissionId(submissionId int64) (*scheduler.JudgeTask, error) {
	// TODO: 实现高效的任务查找
	// 这里简化实现，返回模拟数据
	return &scheduler.JudgeTask{
		SubmissionID: submissionId,
		Status:       "pending", // 模拟一个可以取消的任务
	}, nil
}

// 获取状态文本
func (l *CancelJudgeLogic) getStatusText(status string) string {
	switch status {
	case "completed":
		return "完成"
	case "failed":
		return "失败"
	case "cancelled":
		return "取消"
	case "running":
		return "执行中"
	case "pending":
		return "等待中"
	default:
		return status
	}
}

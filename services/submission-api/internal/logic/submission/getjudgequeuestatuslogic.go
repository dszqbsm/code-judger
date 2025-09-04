package submission

import (
	"context"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetJudgeQueueStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetJudgeQueueStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetJudgeQueueStatusLogic {
	return &GetJudgeQueueStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetJudgeQueueStatusLogic) GetJudgeQueueStatus() (resp *types.GetJudgeQueueStatusResp, err error) {
	l.Logger.Info("获取判题队列状态")

	// 1. 权限检查 - 只有管理员可以查看队列状态
	userRole := l.ctx.Value("role").(string)
	if userRole != "admin" && userRole != "teacher" {
		l.Logger.Errorf("权限不足: Role=%s", userRole)
		return &types.GetJudgeQueueStatusResp{
			Code:    403,
			Message: "无权查看判题队列状态",
		}, nil
	}

	// 2. 调用判题服务获取队列状态
	queueStatus, err := l.svcCtx.JudgeClient.GetJudgeQueue(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用判题服务失败: %v", err)
		return &types.GetJudgeQueueStatusResp{
			Code:    500,
			Message: "获取判题队列状态失败",
		}, nil
	}

	// 3. 检查判题服务返回的状态
	if queueStatus.Code != 200 {
		l.Logger.Errorf("判题服务返回错误: Code=%d, Message=%s", queueStatus.Code, queueStatus.Message)
		return &types.GetJudgeQueueStatusResp{
			Code:    queueStatus.Code,
			Message: queueStatus.Message,
		}, nil
	}

	// 4. 转换数据格式并返回
	queueData := &types.JudgeQueueData{
		QueueLength:    queueStatus.Data.QueueLength,
		PendingTasks:   queueStatus.Data.PendingTasks,
		RunningTasks:   queueStatus.Data.RunningTasks,
		CompletedTasks: queueStatus.Data.CompletedTasks,
		FailedTasks:    queueStatus.Data.FailedTasks,
	}

	// 转换队列项目
	queueData.QueueItems = make([]types.QueueItem, len(queueStatus.Data.QueueItems))
	for i, item := range queueStatus.Data.QueueItems {
		queueData.QueueItems[i] = types.QueueItem{
			SubmissionId:  item.SubmissionId,
			UserId:        item.UserId,
			ProblemId:     item.ProblemId,
			Language:      item.Language,
			Priority:      item.Priority,
			QueueTime:     item.QueueTime,
			EstimatedTime: item.EstimatedTime,
		}
	}

	l.Logger.Infof("成功获取判题队列状态: QueueLength=%d, PendingTasks=%d, RunningTasks=%d",
		queueData.QueueLength, queueData.PendingTasks, queueData.RunningTasks)

	return &types.GetJudgeQueueStatusResp{
		Code:    200,
		Message: "获取成功",
		Data:    *queueData,
	}, nil
}





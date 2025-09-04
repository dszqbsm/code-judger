package system

import (
	"context"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetJudgeQueueLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetJudgeQueueLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetJudgeQueueLogic {
	return &GetJudgeQueueLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetJudgeQueueLogic) GetJudgeQueue(req *types.GetJudgeQueueReq) (resp *types.GetJudgeQueueResp, err error) {
	// 从任务调度器获取队列状态
	queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()

	// 转换队列项格式
	queueItems := make([]types.QueueItem, len(queueStatus.QueueItems))
	for i, item := range queueStatus.QueueItems {
		queueItems[i] = types.QueueItem{
			SubmissionId:  item.SubmissionID,
			UserId:        item.UserID,
			ProblemId:     item.ProblemID,
			Language:      item.Language,
			Priority:      item.Priority,
			QueueTime:     item.QueueTime,
			EstimatedTime: item.EstimatedTime,
		}
	}

	logx.Infof("Retrieved judge queue status: queue_length=%d, running_tasks=%d",
		queueStatus.QueueLength, queueStatus.RunningTasks)

	return &types.GetJudgeQueueResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "获取成功",
		},
		Data: types.JudgeQueueData{
			QueueLength:    queueStatus.QueueLength,
			PendingTasks:   queueStatus.PendingTasks,
			RunningTasks:   queueStatus.RunningTasks,
			CompletedTasks: queueStatus.CompletedTasks,
			FailedTasks:    queueStatus.FailedTasks,
			QueueItems:     queueItems,
		},
	}, nil
}

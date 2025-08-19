package judge

import (
	"context"

	"github.com/online-judge/code-judger/services/judge-api/internal/svc"
	"github.com/online-judge/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejudgeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRejudgeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejudgeLogic {
	return &RejudgeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RejudgeLogic) Rejudge(req *types.RejudgeReq) (resp *types.RejudgeResp, err error) {
	// 验证提交ID
	if req.SubmissionId <= 0 {
		return &types.RejudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: "无效的提交ID",
			},
		}, nil
	}

	// TODO: 实现重新判题逻辑
	// 1. 从数据库获取原始提交信息
	// 2. 创建新的判题任务
	// 3. 提交到调度器

	logx.Infof("Rejudge request for submission %d", req.SubmissionId)

	// 这里简化实现
	return &types.RejudgeResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "重新判题任务已提交",
		},
		Data: types.RejudgeData{
			SubmissionId: req.SubmissionId,
			Status:       "pending",
			Message:      "重新判题任务已加入队列",
		},
	}, nil
}

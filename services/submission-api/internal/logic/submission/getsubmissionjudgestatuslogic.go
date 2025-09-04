package submission

import (
	"context"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSubmissionJudgeStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSubmissionJudgeStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubmissionJudgeStatusLogic {
	return &GetSubmissionJudgeStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubmissionJudgeStatusLogic) GetSubmissionJudgeStatus(req *types.GetSubmissionJudgeStatusReq) (resp *types.GetSubmissionJudgeStatusResp, err error) {
	l.Logger.Infof("获取提交判题状态: SubmissionID=%d", req.SubmissionID)

	// 1. 验证提交是否存在且属于当前用户
	submission, err := l.svcCtx.SubmissionModel.FindOne(l.ctx, req.SubmissionID)
	if err != nil {
		l.Logger.Errorf("查询提交记录失败: %v", err)
		return &types.GetSubmissionJudgeStatusResp{
			Code:    404,
			Message: "提交记录不存在",
		}, nil
	}

	// 2. 权限检查 - 只能查看自己的提交状态
	userID := l.ctx.Value("user_id").(int64)
	if submission.UserID != userID {
		l.Logger.Errorf("权限不足: UserID=%d, SubmissionUserID=%d", userID, submission.UserID)
		return &types.GetSubmissionJudgeStatusResp{
			Code:    403,
			Message: "无权查看此提交的判题状态",
		}, nil
	}

	// 3. 调用判题服务获取实时状态
	judgeStatus, err := l.svcCtx.JudgeClient.GetJudgeStatus(l.ctx, req.SubmissionID)
	if err != nil {
		l.Logger.Errorf("调用判题服务失败: %v", err)
		return &types.GetSubmissionJudgeStatusResp{
			Code:    500,
			Message: "获取判题状态失败",
		}, nil
	}

	// 4. 检查判题服务返回的状态
	if judgeStatus.Code != 200 {
		l.Logger.Errorf("判题服务返回错误: Code=%d, Message=%s", judgeStatus.Code, judgeStatus.Message)
		return &types.GetSubmissionJudgeStatusResp{
			Code:    judgeStatus.Code,
			Message: judgeStatus.Message,
		}, nil
	}

	// 5. 转换数据格式并返回
	status := &types.JudgeStatus{
		SubmissionId:    judgeStatus.Data.SubmissionId,
		Status:          judgeStatus.Data.Status,
		Progress:        judgeStatus.Data.Progress,
		CurrentTestCase: judgeStatus.Data.CurrentTestCase,
		TotalTestCases:  judgeStatus.Data.TotalTestCases,
		Message:         judgeStatus.Data.Message,
	}

	l.Logger.Infof("成功获取判题状态: SubmissionID=%d, Status=%s, Progress=%d%%",
		req.SubmissionID, status.Status, status.Progress)

	return &types.GetSubmissionJudgeStatusResp{
		Code:    200,
		Message: "获取成功",
		Data:    *status,
	}, nil
}

package submission

import (
	"context"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSubmissionJudgeResultLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSubmissionJudgeResultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubmissionJudgeResultLogic {
	return &GetSubmissionJudgeResultLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubmissionJudgeResultLogic) GetSubmissionJudgeResult(req *types.GetSubmissionJudgeResultReq) (resp *types.GetSubmissionJudgeResultResp, err error) {
	l.Logger.Infof("获取提交判题结果: SubmissionID=%d", req.SubmissionID)

	// 1. 验证提交是否存在且属于当前用户
	submission, err := l.svcCtx.SubmissionModel.FindOne(l.ctx, req.SubmissionID)
	if err != nil {
		l.Logger.Errorf("查询提交记录失败: %v", err)
		return &types.GetSubmissionJudgeResultResp{
			Code:    404,
			Message: "提交记录不存在",
		}, nil
	}

	// 2. 权限检查 - 只能查看自己的提交结果
	userID := l.ctx.Value("user_id").(int64)
	if submission.UserID != userID {
		l.Logger.Errorf("权限不足: UserID=%d, SubmissionUserID=%d", userID, submission.UserID)
		return &types.GetSubmissionJudgeResultResp{
			Code:    403,
			Message: "无权查看此提交的判题结果",
		}, nil
	}

	// 3. 直接从数据库构建结果（判题结果已通过Kafka+WebSocket实时推送）
	result := &types.JudgeResult{
		SubmissionId: submission.ID,
		Status:       submission.Status,
		Score:        int(submission.Score.Int32),
		TimeUsed:     int(submission.TimeUsed.Int32),
		MemoryUsed:   int(submission.MemoryUsed.Int32),
		CompileInfo: types.CompileInfo{
			Success: submission.CompileInfo.String == "success",
			Message: submission.CompileInfo.String,
			Time:    0, // 编译时间字段暂不可用，使用默认值
		},
		JudgeInfo: types.JudgeInfo{
			JudgeServer:     submission.JudgeServer.String,
			JudgeTime:       submission.JudgedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			LanguageVersion: submission.Language, // 简化版本信息
		},
	}

	// TODO: 如果需要详细的测试用例结果，可以从专门的测试用例结果表查询
	// 当前简化实现返回基本信息
	result.TestCases = []types.TestCaseResult{}

	l.Logger.Infof("成功获取判题结果: SubmissionID=%d, Status=%s, Score=%d",
		req.SubmissionID, result.Status, result.Score)

	return &types.GetSubmissionJudgeResultResp{
		Code:    200,
		Message: "获取成功",
		Data:    *result,
	}, nil
}

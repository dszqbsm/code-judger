package submission

import (
	"context"
	"time"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejudgeSubmissionProxyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRejudgeSubmissionProxyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejudgeSubmissionProxyLogic {
	return &RejudgeSubmissionProxyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RejudgeSubmissionProxyLogic) RejudgeSubmission(req *types.RejudgeSubmissionProxyReq) (resp *types.RejudgeSubmissionProxyResp, err error) {
	l.Logger.Infof("重新判题请求: SubmissionID=%d", req.SubmissionID)

	// 1. 验证提交是否存在
	submission, err := l.svcCtx.SubmissionModel.FindOne(l.ctx, req.SubmissionID)
	if err != nil {
		l.Logger.Errorf("查询提交记录失败: %v", err)
		return &types.RejudgeSubmissionProxyResp{
			Code:    404,
			Message: "提交记录不存在",
		}, nil
	}

	// 2. 权限检查 - 只能重判自己的提交，或管理员可以重判所有提交
	userID := l.ctx.Value("user_id").(int64)
	userRole := l.ctx.Value("role").(string)

	if submission.UserID != userID && userRole != "admin" && userRole != "teacher" {
		l.Logger.Errorf("权限不足: UserID=%d, SubmissionUserID=%d, Role=%s", userID, submission.UserID, userRole)
		return &types.RejudgeSubmissionProxyResp{
			Code:    403,
			Message: "无权重新判题此提交",
		}, nil
	}

	// 3. 检查提交状态 - 只有已完成的提交才能重判
	if submission.Status == "pending" || submission.Status == "judging" {
		l.Logger.Errorf("提交状态不允许重判: Status=%s", submission.Status)
		return &types.RejudgeSubmissionProxyResp{
			Code:    400,
			Message: "当前提交正在判题中，无法重新判题",
		}, nil
	}

	// 4. 调用判题服务进行重新判题
	rejudgeResult, err := l.svcCtx.JudgeClient.RejudgeSubmission(l.ctx, req.SubmissionID)
	if err != nil {
		l.Logger.Errorf("调用判题服务失败: %v", err)
		return &types.RejudgeSubmissionProxyResp{
			Code:    500,
			Message: "重新判题请求失败",
		}, nil
	}

	// 5. 检查判题服务返回的状态
	if rejudgeResult.Code != 200 {
		l.Logger.Errorf("判题服务返回错误: Code=%d, Message=%s", rejudgeResult.Code, rejudgeResult.Message)
		return &types.RejudgeSubmissionProxyResp{
			Code:    rejudgeResult.Code,
			Message: rejudgeResult.Message,
		}, nil
	}

	// 6. 更新提交状态为重新判题中
	err = l.svcCtx.SubmissionModel.UpdateStatus(l.ctx, req.SubmissionID, "rejudging")
	if err != nil {
		l.Logger.Errorf("更新提交状态失败: %v", err)
		// 这里不返回错误，因为判题任务已经提交成功
	}

	// 7. 记录重判操作日志
	l.logRejudgeOperation(req.SubmissionID, userID, userRole)

	// 8. 转换数据格式并返回
	result := &types.RejudgeData{
		SubmissionId: rejudgeResult.Data.SubmissionId,
		Status:       rejudgeResult.Data.Status,
		Message:      rejudgeResult.Data.Message,
	}

	l.Logger.Infof("重新判题请求成功: SubmissionID=%d, Status=%s",
		req.SubmissionID, result.Status)

	return &types.RejudgeSubmissionProxyResp{
		Code:    200,
		Message: "重新判题任务已提交",
		Data:    *result,
	}, nil
}

// 记录重判操作日志
func (l *RejudgeSubmissionProxyLogic) logRejudgeOperation(submissionID, userID int64, userRole string) {
	// 这里可以记录到操作日志表
	l.Logger.Infof("重判操作记录: SubmissionID=%d, OperatorID=%d, OperatorRole=%s, Time=%s",
		submissionID, userID, userRole, time.Now().Format("2006-01-02 15:04:05"))

	// 可以扩展为写入专门的操作日志表
	// operationLog := &models.OperationLog{
	//     SubmissionID: submissionID,
	//     OperatorID:   userID,
	//     OperatorRole: userRole,
	//     Operation:    "rejudge",
	//     CreatedAt:    time.Now(),
	// }
	// l.svcCtx.OperationLogModel.Insert(l.ctx, operationLog)
}

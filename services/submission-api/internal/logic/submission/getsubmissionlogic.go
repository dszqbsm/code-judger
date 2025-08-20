package submission

import (
	"context"
	"fmt"

	"github.com/online-judge/code-judger/services/submission-api/internal/svc"
	"github.com/online-judge/code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSubmissionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSubmissionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubmissionLogic {
	return &GetSubmissionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubmissionLogic) GetSubmission(req *types.GetSubmissionReq) (resp *types.GetSubmissionResp, err error) {
	// TODO: 临时跳过用户验证

	// TODO: 生产环境中需要恢复认证检查
	// user, ok := middleware.GetUserFromContext(l.ctx)
	// if !ok {
	//     return nil, fmt.Errorf("用户信息不存在")
	// }

	// 直接使用SQL查询提交记录
	var submission struct {
		Id         int64   `db:"id"`
		UserId     int64   `db:"user_id"`
		ProblemId  int64   `db:"problem_id"`
		ContestId  *int64  `db:"contest_id"`
		Language   string  `db:"language"`
		Code       string  `db:"code"`
		Status     string  `db:"status"`
		TimeUsed   *int64  `db:"time_used"`
		MemoryUsed *int64  `db:"memory_used"`
		Score      *int64  `db:"score"`
		CreatedAt  string  `db:"created_at"`
		JudgedAt   *string `db:"judged_at"`
	}

	l.Infof("开始查询提交记录，ID: %d", req.SubmissionID)

	// 添加详细日志：检查数据库连接
	if l.svcCtx.DB == nil {
		l.Logger.Errorf("数据库连接为空")
		return &types.GetSubmissionResp{
			Code:    500,
			Message: "数据库连接错误",
		}, nil
	}
	l.Infof("数据库连接正常")

	// 添加详细日志：检查SubmissionModel
	if l.svcCtx.SubmissionModel == nil {
		l.Logger.Errorf("SubmissionModel为空")
		return &types.GetSubmissionResp{
			Code:    500,
			Message: "模型初始化错误",
		}, nil
	}
	l.Infof("SubmissionModel正常")

	// 使用SubmissionModel的FindOne方法
	l.Infof("调用SubmissionModel.FindOne，参数 ID=%d", req.SubmissionID)
	submissionRecord, err := l.svcCtx.SubmissionModel.FindOne(l.ctx, req.SubmissionID)
	if err != nil {
		l.Logger.Errorf("查询提交记录失败，详细错误: %+v, 错误类型: %T", err, err)
		l.Logger.Errorf("查询参数: ID=%d", req.SubmissionID)

		// 检查是否是记录不存在的错误
		if err.Error() == "record not found" || err.Error() == "sql: no rows in result set" {
			return &types.GetSubmissionResp{
				Code:    404,
				Message: "提交记录不存在",
			}, nil
		}

		return &types.GetSubmissionResp{
			Code:    400,
			Message: fmt.Sprintf("查询失败: %v", err),
		}, nil
	}

	l.Infof("成功查询到提交记录: ID=%d", submissionRecord.Id)

	// 填充submission结构体
	submission.Id = submissionRecord.Id
	submission.UserId = submissionRecord.UserId
	submission.ProblemId = submissionRecord.ProblemId

	// 处理ContestId（sql.NullInt64类型）
	if submissionRecord.ContestId.Valid {
		submission.ContestId = &submissionRecord.ContestId.Int64
	}

	submission.Language = submissionRecord.Language
	submission.Code = submissionRecord.Code
	submission.Status = submissionRecord.Status

	// 处理指针类型字段
	submission.TimeUsed = &submissionRecord.TimeUsed
	submission.MemoryUsed = &submissionRecord.MemoryUsed
	submission.Score = &submissionRecord.Score

	submission.CreatedAt = submissionRecord.CreatedAt.String()
	if submissionRecord.JudgedAt.Valid {
		judgedAt := submissionRecord.JudgedAt.Time.String()
		submission.JudgedAt = &judgedAt
	}

	l.Infof("成功查询到提交记录: ID=%d, Status=%s", submission.Id, submission.Status)

	// TODO: 权限检查（暂时跳过）
	// if !l.canViewSubmission(user, submission) {
	//     return nil, fmt.Errorf("权限不足，无法查看此提交")
	// }

	// 构建响应数据
	respData := &types.GetSubmissionRespData{
		SubmissionID: submission.Id,
		ProblemID:    submission.ProblemId,
		UserID:       submission.UserId,
		Username:     "test_user", // 临时固定值
		Language:     submission.Language,
		Code:         submission.Code,
		Status:       submission.Status,
		CreatedAt:    submission.CreatedAt,
		UpdatedAt:    submission.CreatedAt, // 使用创建时间
	}

	// TODO: 解析判题结果和编译信息（暂时跳过）

	// 设置判题完成时间
	if submission.JudgedAt != nil {
		respData.JudgedAt = submission.JudgedAt
	}

	return &types.GetSubmissionResp{
		Code:    200,
		Message: "获取成功",
		Data:    *respData,
	}, nil
}

// TODO: 删除不需要的辅助函数

package submission

import (
	"context"
	"fmt"
	"strconv"

	"github.com/online-judge/code-judger/services/submission-api/internal/middleware"
	"github.com/online-judge/code-judger/services/submission-api/internal/svc"
	"github.com/online-judge/code-judger/services/submission-api/internal/types"
	"github.com/online-judge/code-judger/services/submission-api/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSubmissionListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSubmissionListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubmissionListLogic {
	return &GetSubmissionListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubmissionListLogic) GetSubmissionList(req *types.GetSubmissionListReq) (resp *types.GetSubmissionListResp, err error) {
	l.Infof("GetSubmissionList请求参数: Page=%d, PageSize=%d, ProblemID=%d, Status=%s, Language=%s, UserID=%d",
		req.Page, req.PageSize, req.ProblemID, req.Status, req.Language, req.UserID)

	// 临时：使用模拟用户信息进行测试
	user := &middleware.UserInfo{
		UserID:   1001,
		Username: "test_user",
		Role:     "admin", // 使用admin权限以便查看所有提交
		TokenID:  "test_token",
	}

	// TODO: 生产环境中需要恢复认证检查
	// user, ok := middleware.GetUserFromContext(l.ctx)
	// if !ok {
	//     return nil, fmt.Errorf("用户信息不存在")
	// }

	// 验证分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > l.svcCtx.Config.Business.MaxPageSize {
		req.PageSize = l.svcCtx.Config.Business.DefaultPageSize
	}

	// 构建搜索条件
	condition := &models.SearchCondition{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// 权限检查和条件设置
	if user.Role == "admin" || user.Role == "teacher" {
		// 管理员和教师可以查看所有提交或指定用户的提交
		if req.UserID > 0 {
			condition.UserID = &req.UserID
		}
	} else {
		// 普通用户只能查看自己的提交
		condition.UserID = &user.UserID
	}

	// 设置其他查询条件
	if req.ProblemID > 0 {
		condition.ProblemID = &req.ProblemID
	}
	if req.Status != "" {
		condition.Status = req.Status
	}
	if req.Language != "" {
		condition.Language = req.Language
	}

	// 查询提交列表
	submissions, total, err := l.svcCtx.SubmissionModel.Search(l.ctx, condition)
	if err != nil {
		l.Logger.Errorf("查询提交列表失败: %v", err)
		return nil, fmt.Errorf("查询失败，请稍后重试")
	}

	// 构建响应数据
	summaryList := make([]types.SubmissionSummary, 0, len(submissions))
	for _, submission := range submissions {
		summary := types.SubmissionSummary{
			SubmissionID: submission.Id,
			ProblemID:    submission.ProblemId,
			ProblemTitle: l.getProblemTitleByID(submission.ProblemId), // 这里应该调用题目服务获取题目标题
			UserID:       submission.UserId,
			Username:     l.getUsernameByID(submission.UserId), // 这里应该调用用户服务获取用户名
			Language:     submission.Language,
			Status:       submission.Status,
			Score:        int(submission.Score),
			TimeUsed:     int(submission.TimeUsed),
			MemoryUsed:   int(submission.MemoryUsed),
			CreatedAt:    submission.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// 设置判题完成时间
		if submission.JudgedAt.Valid {
			judgedAt := submission.JudgedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			summary.JudgedAt = &judgedAt
		}

		summaryList = append(summaryList, summary)
	}

	return &types.GetSubmissionListResp{
		Code:    200,
		Message: "获取成功",
		Data: types.GetSubmissionListRespData{
			Submissions: summaryList,
			Total:       total,
			Page:        req.Page,
			PageSize:    req.PageSize,
		},
	}, nil
}

// getUsernameByID 根据用户ID获取用户名
func (l *GetSubmissionListLogic) getUsernameByID(userID int64) string {
	// 这里应该调用用户服务获取用户名
	// 为了简化，这里返回一个格式化的用户名
	return "user_" + strconv.FormatInt(userID, 10)
}

// getProblemTitleByID 根据题目ID获取题目标题
func (l *GetSubmissionListLogic) getProblemTitleByID(problemID int64) string {
	// 这里应该调用题目服务获取题目标题
	// 为了简化，这里返回一个格式化的题目标题
	return "题目_" + strconv.FormatInt(problemID, 10)
}

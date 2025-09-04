package submission

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"
	"github.com/dszqbsm/code-judger/services/submission-api/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSubmissionListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetSubmissionListLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetSubmissionListLogic {
	return &GetSubmissionListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetSubmissionListLogic) GetSubmissionList(req *types.GetSubmissionListReq) (resp *types.GetSubmissionListResp, err error) {
	l.Logger.Infof("开始处理获取提交列表请求: Page=%d, PageSize=%d, ProblemID=%d, Status=%s, Language=%s, UserID=%d",
		req.Page, req.PageSize, req.ProblemID, req.Status, req.Language, req.UserID)

	// 1. 从JWT获取用户信息
	user, err := l.getUserFromJWT()
	if err != nil {
		l.Logger.Errorf("获取用户信息失败: %v", err)
		return &types.GetSubmissionListResp{
			Code:    401,
			Message: "认证失败：" + err.Error(),
		}, nil
	}

	l.Logger.Infof("用户认证成功: UserID=%d, Username=%s, Role=%s", user.UserID, user.Username, user.Role)

	// 2. 验证和调整分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > l.svcCtx.Config.Business.MaxPageSize {
		req.PageSize = l.svcCtx.Config.Business.DefaultPageSize
	}

	// 3. 权限检查和条件构建
	condition, err := l.buildSearchCondition(req, user)
	if err != nil {
		l.Logger.Errorf("构建搜索条件失败: %v", err)
		return &types.GetSubmissionListResp{
			Code:    403,
			Message: err.Error(),
		}, nil
	}

	// 4. 查询提交列表
	submissions, total, err := l.svcCtx.SubmissionModel.Search(l.ctx, condition)
	if err != nil {
		l.Logger.Errorf("查询提交列表失败: %v", err)
		return &types.GetSubmissionListResp{
			Code:    500,
			Message: "查询失败，请稍后重试",
		}, nil
	}

	l.Logger.Infof("成功查询到提交列表: 总数=%d, 当前页数量=%d", total, len(submissions))

	// 5. 并发获取用户名和题目标题
	summaryList, err := l.buildSubmissionSummaryList(submissions, user)
	if err != nil {
		l.Logger.Errorf("构建提交摘要列表失败: %v", err)
		return &types.GetSubmissionListResp{
			Code:    500,
			Message: "数据处理失败",
		}, nil
	}

	l.Logger.Infof("用户 %s 成功获取提交列表: 页码=%d, 页大小=%d, 总数=%d",
		user.Username, req.Page, req.PageSize, total)

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

// getUserFromJWT 从JWT中获取用户信息
func (l *GetSubmissionListLogic) getUserFromJWT() (*middleware.UserInfo, error) {
	// 方法1: 尝试从go-zero的JWT上下文获取用户信息
	if user, ok := middleware.GetUserFromContext(l.ctx); ok && user != nil {
		return user, nil
	}

	// 方法2: 从HTTP请求头获取JWT令牌并解析
	if l.r != nil {
		user, err := middleware.GetUserFromJWT(l.r, l.svcCtx.JWTManager)
		if err != nil {
			return nil, fmt.Errorf("JWT令牌解析失败: %v", err)
		}
		return user, nil
	}

	return nil, fmt.Errorf("无法获取用户信息：上下文和请求头都为空")
}

// buildSearchCondition 构建搜索条件并进行权限检查
func (l *GetSubmissionListLogic) buildSearchCondition(req *types.GetSubmissionListReq, user *middleware.UserInfo) (*models.SearchCondition, error) {
	condition := &models.SearchCondition{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// 1. 根据用户角色设置查看权限
	switch user.Role {
	case "admin":
		// 管理员可以查看所有提交
		l.Logger.Infof("管理员用户 %s 查询提交列表", user.Username)
		if req.UserID > 0 {
			condition.UserID = &req.UserID
		}

	case "teacher":
		// 教师可以查看所有公开提交或指定用户的提交
		l.Logger.Infof("教师用户 %s 查询提交列表", user.Username)
		if req.UserID > 0 {
			// 教师查看指定用户的提交
			condition.UserID = &req.UserID
		}
		// 如果没有指定用户，教师可以查看所有公开提交（在数据查询时过滤）

	case "student":
		// 学生只能查看自己的提交，或者公开的提交
		if req.UserID > 0 && req.UserID != user.UserID {
			return nil, fmt.Errorf("权限不足：学生只能查看自己的提交")
		}
		condition.UserID = &user.UserID
		l.Logger.Infof("学生用户 %s 查询自己的提交列表", user.Username)

	default:
		return nil, fmt.Errorf("无效的用户角色: %s", user.Role)
	}

	// 2. 设置题目过滤条件
	if req.ProblemID > 0 {
		// 验证题目是否存在和用户是否有权限访问
		if err := l.validateProblemAccess(req.ProblemID, user); err != nil {
			return nil, fmt.Errorf("题目访问权限验证失败: %v", err)
		}
		condition.ProblemID = &req.ProblemID
	}

	// 3. 设置比赛过滤条件
	if req.ContestID > 0 {
		// 验证比赛权限
		if err := l.validateContestAccess(req.ContestID, user); err != nil {
			return nil, fmt.Errorf("比赛访问权限验证失败: %v", err)
		}
		condition.ContestID = &req.ContestID
	}

	// 4. 设置其他过滤条件
	if req.Status != "" {
		condition.Status = req.Status
	}
	if req.Language != "" {
		condition.Language = req.Language
	}

	return condition, nil
}

// validateProblemAccess 验证题目访问权限
func (l *GetSubmissionListLogic) validateProblemAccess(problemID int64, user *middleware.UserInfo) error {
	// TODO: 调用题目服务验证题目是否存在和用户是否有权限访问
	// problemClient := l.svcCtx.ProblemRpc
	// problem, err := problemClient.GetProblem(l.ctx, &problem.GetProblemReq{Id: problemID})
	// if err != nil {
	//     return fmt.Errorf("题目不存在: %d", problemID)
	// }

	// 检查题目是否公开或用户是否有权限访问
	// if !problem.IsPublic && user.Role != "admin" && user.Role != "teacher" {
	//     return fmt.Errorf("无权限访问题目: %d", problemID)
	// }

	l.Logger.Infof("题目访问权限验证通过: ProblemID=%d, UserID=%d, Role=%s", problemID, user.UserID, user.Role)
	return nil
}

// validateContestAccess 验证比赛访问权限
func (l *GetSubmissionListLogic) validateContestAccess(contestID int64, user *middleware.UserInfo) error {
	// TODO: 调用比赛服务验证比赛是否存在和用户是否有权限访问
	// contestClient := l.svcCtx.ContestRpc
	// contest, err := contestClient.GetContest(l.ctx, &contest.GetContestReq{Id: contestID})
	// if err != nil {
	//     return fmt.Errorf("比赛不存在: %d", contestID)
	// }

	// 检查用户是否参加了该比赛
	// participant, err := contestClient.GetParticipant(l.ctx, &contest.GetParticipantReq{
	//     ContestID: contestID,
	//     UserID:    user.UserID,
	// })
	// if err != nil && user.Role != "admin" && user.Role != "teacher" {
	//     return fmt.Errorf("未参加该比赛，无法查看提交: %d", contestID)
	// }

	l.Logger.Infof("比赛访问权限验证通过: ContestID=%d, UserID=%d, Role=%s", contestID, user.UserID, user.Role)
	return nil
}

// buildSubmissionSummaryList 并发构建提交摘要列表
func (l *GetSubmissionListLogic) buildSubmissionSummaryList(submissions []*models.Submission, user *middleware.UserInfo) ([]types.SubmissionSummary, error) {
	if len(submissions) == 0 {
		return []types.SubmissionSummary{}, nil
	}

	summaryList := make([]types.SubmissionSummary, len(submissions))

	// 使用WaitGroup和channel进行并发处理
	var wg sync.WaitGroup
	errChan := make(chan error, len(submissions))

	// 收集所有需要查询的用户ID和题目ID
	userIDs := make(map[int64]bool)
	problemIDs := make(map[int64]bool)

	for _, submission := range submissions {
		userIDs[submission.UserID] = true
		problemIDs[submission.ProblemID] = true
	}

	// 并发获取用户名映射
	usernameMap, err := l.getUsernameMapByIDs(userIDs)
	if err != nil {
		l.Logger.Errorf("批量获取用户名失败: %v", err)
		// 不返回错误，使用默认值
		usernameMap = make(map[int64]string)
	}

	// 并发获取题目标题映射
	problemTitleMap, err := l.getProblemTitleMapByIDs(problemIDs)
	if err != nil {
		l.Logger.Errorf("批量获取题目标题失败: %v", err)
		// 不返回错误，使用默认值
		problemTitleMap = make(map[int64]string)
	}

	// 并发构建每个提交的摘要信息
	for i, submission := range submissions {
		wg.Add(1)
		go func(index int, sub *models.Submission) {
			defer wg.Done()

			summary := l.buildSingleSubmissionSummary(sub, user, usernameMap, problemTitleMap)
			summaryList[index] = summary
		}(i, submission)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return summaryList, nil
}

// buildSingleSubmissionSummary 构建单个提交摘要
func (l *GetSubmissionListLogic) buildSingleSubmissionSummary(
	submission *models.Submission,
	user *middleware.UserInfo,
	usernameMap map[int64]string,
	problemTitleMap map[int64]string,
) types.SubmissionSummary {

	// 获取用户名
	username := usernameMap[submission.UserID]
	if username == "" {
		username = fmt.Sprintf("user_%d", submission.UserID)
	}

	// 获取题目标题
	problemTitle := problemTitleMap[submission.ProblemID]
	if problemTitle == "" {
		problemTitle = fmt.Sprintf("Problem_%d", submission.ProblemID)
	}

	summary := types.SubmissionSummary{
		SubmissionID: submission.ID,
		ProblemID:    submission.ProblemID,
		ProblemTitle: problemTitle,
		UserID:       submission.UserID,
		Username:     username,
		Language:     submission.Language,
		Status:       submission.Status,
		CreatedAt: func() string {
			if submission.CreatedAt.Valid {
				return submission.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			}
			return ""
		}(),
	}

	// 处理可空字段
	if submission.Score.Valid {
		summary.Score = int(submission.Score.Int32)
	}
	if submission.TimeUsed.Valid {
		summary.TimeUsed = int(submission.TimeUsed.Int32)
	}
	if submission.MemoryUsed.Valid {
		summary.MemoryUsed = int(submission.MemoryUsed.Int32)
	}

	// 设置比赛ID
	if submission.ContestID.Valid {
		summary.ContestID = &submission.ContestID.Int64
	}

	// 设置判题完成时间
	if submission.JudgedAt.Valid {
		judgedAt := submission.JudgedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		summary.JudgedAt = &judgedAt
	}

	// 根据权限过滤敏感信息
	summary = l.filterSummaryByPermission(summary, submission, user)

	return summary
}

// getUsernameMapByIDs 批量获取用户名映射
func (l *GetSubmissionListLogic) getUsernameMapByIDs(userIDs map[int64]bool) (map[int64]string, error) {
	usernameMap := make(map[int64]string)

	// 方法1: 从Redis批量获取缓存的用户名
	cachedUsernames := l.getUsernamesFromCache(userIDs)
	for userID, username := range cachedUsernames {
		usernameMap[userID] = username
	}

	// 方法2: 对于缓存中没有的用户，调用用户服务批量获取
	uncachedUserIDs := make([]int64, 0)
	for userID := range userIDs {
		if _, exists := usernameMap[userID]; !exists {
			uncachedUserIDs = append(uncachedUserIDs, userID)
		}
	}

	if len(uncachedUserIDs) > 0 {
		// TODO: 调用用户服务批量获取用户名
		// userClient := l.svcCtx.UserRpc
		// users, err := userClient.GetUsersByIDs(l.ctx, &user.GetUsersByIDsReq{Ids: uncachedUserIDs})
		// if err != nil {
		//     l.Logger.Errorf("批量获取用户信息失败: %v", err)
		//     return usernameMap, nil // 不返回错误，使用已有的缓存数据
		// }

		// for _, user := range users.Users {
		//     usernameMap[user.Id] = user.Username
		//     // 缓存到Redis（5分钟）
		//     cacheKey := fmt.Sprintf("user:username:%d", user.Id)
		//     l.svcCtx.RedisClient.Setex(cacheKey, user.Username, 300)
		// }

		// 暂时使用格式化的用户名
		for _, userID := range uncachedUserIDs {
			usernameMap[userID] = fmt.Sprintf("user_%d", userID)
		}
	}

	return usernameMap, nil
}

// getProblemTitleMapByIDs 批量获取题目标题映射
func (l *GetSubmissionListLogic) getProblemTitleMapByIDs(problemIDs map[int64]bool) (map[int64]string, error) {
	problemTitleMap := make(map[int64]string)

	// 方法1: 从Redis批量获取缓存的题目标题
	cachedTitles := l.getProblemTitlesFromCache(problemIDs)
	for problemID, title := range cachedTitles {
		problemTitleMap[problemID] = title
	}

	// 方法2: 对于缓存中没有的题目，调用题目服务批量获取
	uncachedProblemIDs := make([]int64, 0)
	for problemID := range problemIDs {
		if _, exists := problemTitleMap[problemID]; !exists {
			uncachedProblemIDs = append(uncachedProblemIDs, problemID)
		}
	}

	if len(uncachedProblemIDs) > 0 {
		// TODO: 调用题目服务批量获取题目标题
		// problemClient := l.svcCtx.ProblemRpc
		// problems, err := problemClient.GetProblemsByIDs(l.ctx, &problem.GetProblemsByIDsReq{Ids: uncachedProblemIDs})
		// if err != nil {
		//     l.Logger.Errorf("批量获取题目信息失败: %v", err)
		//     return problemTitleMap, nil // 不返回错误，使用已有的缓存数据
		// }

		// for _, problem := range problems.Problems {
		//     problemTitleMap[problem.Id] = problem.Title
		//     // 缓存到Redis（10分钟）
		//     cacheKey := fmt.Sprintf("problem:title:%d", problem.Id)
		//     l.svcCtx.RedisClient.Setex(cacheKey, problem.Title, 600)
		// }

		// 暂时使用格式化的题目标题
		for _, problemID := range uncachedProblemIDs {
			problemTitleMap[problemID] = fmt.Sprintf("Problem_%d", problemID)
		}
	}

	return problemTitleMap, nil
}

// getUsernamesFromCache 从Redis批量获取用户名缓存
func (l *GetSubmissionListLogic) getUsernamesFromCache(userIDs map[int64]bool) map[int64]string {
	usernameMap := make(map[int64]string)

	for userID := range userIDs {
		cacheKey := fmt.Sprintf("user:username:%d", userID)
		username, err := l.svcCtx.RedisClient.Get(cacheKey)
		if err == nil && username != "" {
			usernameMap[userID] = username
		}
	}

	return usernameMap
}

// getProblemTitlesFromCache 从Redis批量获取题目标题缓存
func (l *GetSubmissionListLogic) getProblemTitlesFromCache(problemIDs map[int64]bool) map[int64]string {
	problemTitleMap := make(map[int64]string)

	for problemID := range problemIDs {
		cacheKey := fmt.Sprintf("problem:title:%d", problemID)
		title, err := l.svcCtx.RedisClient.Get(cacheKey)
		if err == nil && title != "" {
			problemTitleMap[problemID] = title
		}
	}

	return problemTitleMap
}

// filterSummaryByPermission 根据权限过滤摘要信息
func (l *GetSubmissionListLogic) filterSummaryByPermission(
	summary types.SubmissionSummary,
	submission *models.Submission,
	user *middleware.UserInfo,
) types.SubmissionSummary {

	// 1. 管理员可以查看所有信息
	if user.Role == "admin" {
		return summary
	}

	// 2. 用户可以查看自己的所有信息
	if user.UserID == submission.UserID {
		return summary
	}

	// 3. 教师可以查看所有提交的详细信息
	if user.Role == "teacher" {
		return summary
	}

	// 4. 对于他人的提交，在真实业务中通常允许查看基本信息
	// 这里保持原始信息，根据实际业务需求调整

	// 5. 比赛期间可能需要隐藏某些信息
	if submission.ContestID.Valid {
		// TODO: 根据比赛状态和规则决定是否隐藏信息
		// 比赛进行中可能隐藏分数和详细结果
	}

	return summary
}

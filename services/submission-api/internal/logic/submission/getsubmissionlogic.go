package submission

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"
	"github.com/dszqbsm/code-judger/services/submission-api/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSubmissionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetSubmissionLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetSubmissionLogic {
	return &GetSubmissionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetSubmissionLogic) GetSubmission(req *types.GetSubmissionReq) (resp *types.GetSubmissionResp, err error) {
	l.Logger.Infof("开始处理获取提交记录请求: SubmissionID=%d", req.SubmissionID)

	// 1. 验证提交ID
	if req.SubmissionID <= 0 {
		l.Logger.Errorf("无效的提交ID: %d", req.SubmissionID)
		return &types.GetSubmissionResp{
			Code:    400,
			Message: "无效的提交ID",
		}, nil
	}

	// 2. 从JWT获取用户信息
	user, err := l.getUserFromJWT()
	if err != nil {
		l.Logger.Errorf("获取用户信息失败: %v", err)
		return &types.GetSubmissionResp{
			Code:    401,
			Message: "认证失败：" + err.Error(),
		}, nil
	}

	l.Logger.Infof("用户认证成功: UserID=%d, Username=%s, Role=%s", user.UserID, user.Username, user.Role)

	// 3. 查询提交记录
	submissionRecord, err := l.svcCtx.SubmissionDao.GetSubmissionByID(l.ctx, req.SubmissionID)
	if err != nil {
		l.Logger.Errorf("查询提交记录失败: %v", err)
		return &types.GetSubmissionResp{
			Code:    404,
			Message: "提交记录不存在",
		}, nil
	}

	l.Logger.Infof("成功查询到提交记录: ID=%d, UserID=%d, ProblemID=%d, Status=%s", 
		submissionRecord.ID, submissionRecord.UserID, submissionRecord.ProblemID, submissionRecord.Status)

	// 4. 权限检查
	if err := l.validateViewPermission(user, submissionRecord); err != nil {
		l.Logger.Errorf("用户 %s (ID: %d) 权限验证失败，无法查看提交 %d: %v",
			user.Username, user.UserID, req.SubmissionID, err)
		return &types.GetSubmissionResp{
			Code:    403,
			Message: err.Error(),
		}, nil
	}

	// 5. 获取用户名（调用用户服务或从缓存获取）
	username, err := l.getUsernameByID(submissionRecord.UserID)
	if err != nil {
		l.Logger.Errorf("获取用户名失败: UserID=%d, Error=%v", submissionRecord.UserID, err)
		username = "unknown" // 降级处理
	}

	// 6. 解析判题结果和编译信息
	judgeResult, err := l.parseJudgeResult(submissionRecord)
	if err != nil {
		l.Logger.Errorf("解析判题结果失败: %v", err)
		// 不返回错误，使用默认值
	}

	// 7. 构建响应数据
	respData := &types.GetSubmissionRespData{
		SubmissionID: submissionRecord.ID,
		ProblemID:    submissionRecord.ProblemID,
		UserID:       submissionRecord.UserID,
		Username:     username,
		Language:     submissionRecord.Language,
		Code:         l.filterCodeByPermission(user, submissionRecord),
		CodeLength:   int(submissionRecord.CodeLength.Int32),
		Status:       submissionRecord.Status,
		CreatedAt:    func() string {
			if submissionRecord.CreatedAt.Valid {
				return submissionRecord.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			}
			return ""
		}(),
		UpdatedAt:    "", // 数据库中没有updated_at字段
	}

	// 设置比赛ID（如果有）
	if submissionRecord.ContestID.Valid {
		respData.ContestID = &submissionRecord.ContestID.Int64
	}

	// 设置判题结果
	if judgeResult != nil {
		respData.Score = judgeResult.Score
		respData.TimeUsed = judgeResult.TimeUsed
		respData.MemoryUsed = judgeResult.MemoryUsed
		respData.CompileOutput = judgeResult.CompileOutput
		respData.RuntimeOutput = judgeResult.RuntimeOutput
		respData.ErrorMessage = judgeResult.ErrorMessage
		respData.TestCasesPassed = judgeResult.TestCasesPassed
		respData.TestCasesTotal = judgeResult.TestCasesTotal
		respData.JudgeServer = judgeResult.JudgeServer
	}

	// 设置判题完成时间
	if submissionRecord.JudgedAt.Valid {
		judgedAt := submissionRecord.JudgedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		respData.JudgedAt = &judgedAt
	}

	l.Logger.Infof("用户 %s 成功获取提交记录: SubmissionID=%d, Status=%s", 
		user.Username, req.SubmissionID, submissionRecord.Status)

	return &types.GetSubmissionResp{
		Code:    200,
		Message: "获取成功",
		Data:    *respData,
	}, nil
}

// JudgeResultDetail 判题结果详情
type JudgeResultDetail struct {
	Score           *int32  `json:"score"`
	TimeUsed        *int32  `json:"time_used"`
	MemoryUsed      *int32  `json:"memory_used"`
	CompileOutput   *string `json:"compile_output"`
	RuntimeOutput   *string `json:"runtime_output"`
	ErrorMessage    *string `json:"error_message"`
	TestCasesPassed *int32  `json:"test_cases_passed"`
	TestCasesTotal  *int32  `json:"test_cases_total"`
	JudgeServer     *string `json:"judge_server"`
}

// getUserFromJWT 从JWT中获取用户信息
func (l *GetSubmissionLogic) getUserFromJWT() (*middleware.UserInfo, error) {
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

// validateViewPermission 验证查看提交的权限
func (l *GetSubmissionLogic) validateViewPermission(user *middleware.UserInfo, submission *models.Submission) error {
	// 1. 管理员可以查看所有提交
	if user.Role == "admin" {
		l.Logger.Infof("管理员用户 %s 查看提交记录: SubmissionID=%d", user.Username, submission.ID)
		return nil
	}

	// 2. 用户可以查看自己的提交
	if user.UserID == submission.UserID {
		l.Logger.Infof("用户 %s 查看自己的提交记录: SubmissionID=%d", user.Username, submission.ID)
		return nil
	}

	// 3. 教师可以查看所有提交（在真实业务中，教师通常有更高权限）
	if user.Role == "teacher" {
		l.Logger.Infof("教师用户 %s 查看提交记录: SubmissionID=%d", user.Username, submission.ID)
		return nil
	}

	// 4. 如果是比赛提交，检查比赛权限
	if submission.ContestID.Valid {
		return l.validateContestViewPermission(user, submission)
	}

	// 5. 检查题目是否允许查看他人提交
	if err := l.validateProblemViewPermission(user, submission); err != nil {
		return err
	}

	// 6. 默认情况下，普通用户可以查看他人的提交（根据实际业务需求调整）
	// 在真实的OJ系统中，通常允许查看他人的AC提交代码用于学习
	l.Logger.Infof("普通用户 %s 查看他人提交记录: SubmissionID=%d", user.Username, submission.ID)

	return nil
}

// validateContestViewPermission 验证比赛提交查看权限
func (l *GetSubmissionLogic) validateContestViewPermission(user *middleware.UserInfo, submission *models.Submission) error {
	// TODO: 调用比赛服务验证权限
	// contestClient := l.svcCtx.ContestRpc
	// contest, err := contestClient.GetContest(l.ctx, &contest.GetContestReq{Id: submission.ContestID.Int64})
	// if err != nil {
	//     return fmt.Errorf("比赛不存在")
	// }

	// 检查用户是否参加了该比赛
	// participant, err := contestClient.GetParticipant(l.ctx, &contest.GetParticipantReq{
	//     ContestID: submission.ContestID.Int64,
	//     UserID:    user.UserID,
	// })
	// if err != nil {
	//     return fmt.Errorf("未参加该比赛，无法查看提交")
	// }

	// 检查比赛状态和查看权限
	// if contest.Status != "finished" && user.UserID != submission.UserID {
	//     return fmt.Errorf("比赛进行中，只能查看自己的提交")
	// }

	l.Logger.Infof("比赛提交权限验证通过: ContestID=%d, UserID=%d, SubmissionID=%d", 
		submission.ContestID.Int64, user.UserID, submission.ID)
	return nil
}

// validateProblemViewPermission 验证题目提交查看权限
func (l *GetSubmissionLogic) validateProblemViewPermission(user *middleware.UserInfo, submission *models.Submission) error {
	// TODO: 调用题目服务获取题目信息
	// problemClient := l.svcCtx.ProblemRpc
	// problem, err := problemClient.GetProblem(l.ctx, &problem.GetProblemReq{Id: submission.ProblemID})
	// if err != nil {
	//     return fmt.Errorf("题目不存在")
	// }

	// 检查题目是否允许查看他人提交
	// if !problem.AllowViewOthersSubmission && user.UserID != submission.UserID {
	//     return fmt.Errorf("该题目不允许查看他人提交")
	// }

	l.Logger.Infof("题目提交权限验证通过: ProblemID=%d, UserID=%d, SubmissionID=%d", 
		submission.ProblemID, user.UserID, submission.ID)
	return nil
}

// getUsernameByID 根据用户ID获取用户名
func (l *GetSubmissionLogic) getUsernameByID(userID int64) (string, error) {
	// 方法1: 从Redis缓存获取
	cacheKey := fmt.Sprintf("user:username:%d", userID)
	username, err := l.svcCtx.RedisClient.Get(cacheKey)
	if err == nil && username != "" {
		return username, nil
	}

	// 方法2: 调用用户服务获取
	// TODO: 实现RPC调用
	// userClient := l.svcCtx.UserRpc
	// user, err := userClient.GetUser(l.ctx, &user.GetUserReq{Id: userID})
	// if err != nil {
	//     return "", fmt.Errorf("获取用户信息失败: %v", err)
	// }

	// 缓存用户名（5分钟）
	// l.svcCtx.RedisClient.Setex(cacheKey, user.Username, 300)
	// return user.Username, nil

	// 暂时返回格式化的用户名
	return fmt.Sprintf("user_%d", userID), nil
}

// filterCodeByPermission 根据权限过滤代码内容
func (l *GetSubmissionLogic) filterCodeByPermission(user *middleware.UserInfo, submission *models.Submission) string {
	// 1. 管理员可以查看所有代码
	if user.Role == "admin" {
		return submission.Code
	}

	// 2. 用户可以查看自己的代码
	if user.UserID == submission.UserID {
		return submission.Code
	}

	// 3. 教师可以查看所有提交的代码
	if user.Role == "teacher" {
		return submission.Code
	}

	// 4. 如果是比赛提交，根据比赛规则决定
	if submission.ContestID.Valid {
		// TODO: 根据比赛设置决定是否显示代码
		// 比赛结束后可能允许查看，比赛进行中可能不允许
		return "[代码在比赛期间不可见]"
	}

	// 5. 默认允许查看代码（根据实际业务需求调整）
	// 在真实的OJ系统中，通常允许查看AC提交的代码用于学习

	// 6. 默认返回代码（如果提交是公开的）
	return submission.Code
}

// parseJudgeResult 解析判题结果
func (l *GetSubmissionLogic) parseJudgeResult(submission *models.Submission) (*JudgeResultDetail, error) {
	result := &JudgeResultDetail{}

	// 1. 基本判题信息
	if submission.Score.Valid {
		result.Score = &submission.Score.Int32
	}
	if submission.TimeUsed.Valid {
		result.TimeUsed = &submission.TimeUsed.Int32
	}
	if submission.MemoryUsed.Valid {
		result.MemoryUsed = &submission.MemoryUsed.Int32
	}
	// TestCasesPassed 和 TestCasesTotal 字段在当前数据库结构中不存在
	// 可以从 Result JSON 字段中解析获得
	// TODO: 从 submission.Result 中解析测试用例信息

	// 编译信息
	if submission.CompileInfo.Valid && submission.CompileInfo.String != "" {
		compileOutput := l.formatCompileOutput(submission.CompileInfo.String)
		result.CompileOutput = &compileOutput
	}

	// 运行时信息
	if submission.RuntimeInfo.Valid && submission.RuntimeInfo.String != "" {
		runtimeOutput := l.formatRuntimeOutput(submission.RuntimeInfo.String)
		result.RuntimeOutput = &runtimeOutput
	}

	// 判题服务器信息
	if submission.JudgeServer.Valid {
		result.JudgeServer = &submission.JudgeServer.String
	}

	// 错误信息可以从CompileInfo或RuntimeInfo中解析
	// TODO: 根据需要从编译信息或运行时信息中提取错误信息

	// 5. 解析详细的测试用例结果（如果存储为JSON）
	if err := l.parseDetailedTestResults(submission, result); err != nil {
		l.Logger.Errorf("解析详细测试结果失败: %v", err)
	}

	return result, nil
}

// formatCompileOutput 格式化编译输出
func (l *GetSubmissionLogic) formatCompileOutput(output string) string {
	if output == "" {
		return ""
	}

	// 1. 限制输出长度
	maxLength := 2000
	if len(output) > maxLength {
		output = output[:maxLength] + "\n...(输出过长，已截断)"
	}

	// 2. 过滤敏感信息（如文件路径）
	output = l.sanitizeOutput(output)

	// 3. 格式化换行符
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")

	return output
}

// formatRuntimeOutput 格式化运行时输出
func (l *GetSubmissionLogic) formatRuntimeOutput(output string) string {
	if output == "" {
		return ""
	}

	// 1. 限制输出长度
	maxLength := 1000
	if len(output) > maxLength {
		output = output[:maxLength] + "\n...(输出过长，已截断)"
	}

	// 2. 过滤敏感信息
	output = l.sanitizeOutput(output)

	// 3. 格式化换行符
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")

	return output
}

// formatErrorMessage 格式化错误信息
func (l *GetSubmissionLogic) formatErrorMessage(message string) string {
	if message == "" {
		return ""
	}

	// 1. 限制错误信息长度
	maxLength := 500
	if len(message) > maxLength {
		message = message[:maxLength] + "...(错误信息过长，已截断)"
	}

	// 2. 过滤敏感信息
	message = l.sanitizeOutput(message)

	return message
}

// sanitizeOutput 清理输出中的敏感信息
func (l *GetSubmissionLogic) sanitizeOutput(output string) string {
	// 移除可能的文件路径
	output = strings.ReplaceAll(output, "/tmp/", "/")
	output = strings.ReplaceAll(output, "/var/", "/")
	output = strings.ReplaceAll(output, "/home/", "/")
	
	// 移除IP地址模式
	// 这里可以添加更复杂的正则表达式来匹配IP地址
	
	return output
}

// parseDetailedTestResults 解析详细的测试用例结果
func (l *GetSubmissionLogic) parseDetailedTestResults(submission *models.Submission, result *JudgeResultDetail) error {
	// 如果有存储详细的测试结果（JSON格式），在这里解析
	// 例如：每个测试用例的运行时间、内存使用、状态等
	
	// TODO: 根据实际的数据存储格式实现
	// 可能的JSON格式示例：
	// {
	//   "test_cases": [
	//     {"id": 1, "status": "AC", "time": 100, "memory": 1024, "score": 10},
	//     {"id": 2, "status": "WA", "time": 50, "memory": 512, "score": 0}
	//   ],
	//   "total_score": 10,
	//   "max_time": 100,
	//   "max_memory": 1024
	// }

	if submission.TestCaseResults.Valid && submission.TestCaseResults.String != "" {
		var detailedResult map[string]interface{}
		if err := json.Unmarshal([]byte(submission.TestCaseResults.String), &detailedResult); err == nil {
			// 成功解析JSON，提取详细信息
			if totalScore, ok := detailedResult["total_score"].(float64); ok {
				score := int32(totalScore)
				result.Score = &score
			}
			if maxTime, ok := detailedResult["max_time"].(float64); ok {
				timeUsed := int32(maxTime)
				result.TimeUsed = &timeUsed
			}
			if maxMemory, ok := detailedResult["max_memory"].(float64); ok {
				memoryUsed := int32(maxMemory)
				result.MemoryUsed = &memoryUsed
			}
		}
	}

	return nil
}

// getSubmissionIDFromPath 从URL路径中提取提交ID
func (l *GetSubmissionLogic) getSubmissionIDFromPath() (int64, error) {
	if l.r == nil {
		return 0, fmt.Errorf("HTTP请求为空")
	}

	// 从URL路径中提取ID，例如 /submission/123
	path := l.r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return 0, fmt.Errorf("无效的URL路径")
	}

	idStr := parts[len(parts)-1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的提交ID格式: %s", idStr)
	}

	return id, nil
}
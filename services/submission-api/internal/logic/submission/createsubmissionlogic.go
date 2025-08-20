package submission

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"code-judger/services/submission-api/internal/middleware"
	"code-judger/services/submission-api/internal/svc"
	"code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSubmissionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateSubmissionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSubmissionLogic {
	return &CreateSubmissionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSubmissionLogic) CreateSubmission(req *types.CreateSubmissionReq) (resp *types.CreateSubmissionResp, err error) {
	l.Logger.Infof("开始处理创建提交请求: ProblemID=%d, Language=%s", req.ProblemID, req.Language)

	// 临时：使用模拟用户信息进行测试
	user := &middleware.UserInfo{
		UserID:   1001,
		Username: "test_user",
		Role:     "student",
		TokenID:  "test_token",
	}

	l.Logger.Infof("使用模拟用户: UserID=%d, Username=%s", user.UserID, user.Username)

	// TODO: 生产环境中需要恢复认证检查
	// user, ok := middleware.GetUserFromContext(l.ctx)
	// if !ok {
	//     return nil, fmt.Errorf("用户信息不存在")
	// }

	// 验证编程语言是否支持
	if !l.isLanguageSupported(req.Language) {
		return nil, fmt.Errorf("不支持的编程语言: %s", req.Language)
	}

	// 验证代码长度
	if len(req.Code) > l.svcCtx.Config.Business.MaxCodeLength {
		return nil, fmt.Errorf("代码长度超出限制，最大允许 %d 字符", l.svcCtx.Config.Business.MaxCodeLength)
	}

	// 验证题目是否存在（这里简化处理，实际应该调用题目服务）
	if req.ProblemID <= 0 {
		return nil, fmt.Errorf("无效的题目ID")
	}

	// 准备插入数据库
	l.Logger.Infof("准备插入数据库")

	// 直接使用SQL插入数据库(临时解决方案)
	query := "INSERT INTO submissions (user_id, problem_id, contest_id, language, code, code_length, status, created_at) VALUES (?, ?, ?, ?, ?, ?, 'pending', NOW())"
	contestID := func() interface{} {
		if req.ContestID > 0 {
			return req.ContestID
		}
		return nil
	}()

	l.Logger.Infof("执行SQL插入: UserID=%d, ProblemID=%d, ContestID=%v, Language=%s, CodeLength=%d",
		user.UserID, req.ProblemID, contestID, req.Language, len(req.Code))

	result, err := l.svcCtx.DB.ExecCtx(l.ctx, query,
		user.UserID,
		req.ProblemID,
		contestID,
		req.Language,
		req.Code,
		len(req.Code))
	if err != nil {
		l.Logger.Errorf("创建提交记录失败: %v", err)
		return nil, fmt.Errorf("提交失败，请稍后重试")
	}

	l.Logger.Infof("数据库插入成功")

	// 获取插入的ID
	submissionID, err := result.LastInsertId()
	if err != nil {
		l.Logger.Errorf("获取提交ID失败: %v", err)
		return nil, fmt.Errorf("提交失败，请稍后重试")
	}

	// 提交记录已创建，ID为submissionID

	// TODO: 发送判题任务到消息队列（暂时跳过以测试基础功能）
	// judgeTask := &JudgeTask{
	//     SubmissionID: submissionID,
	//     UserID:       user.UserID,
	//     ProblemID:    req.ProblemID,
	//     Language:     req.Language,
	//     Code:         req.Code,
	//     ContestID:    &req.ContestID,
	//     Priority:     l.calculatePriority(user.Role, &req.ContestID),
	//     CreatedAt:    time.Now(),
	// }

	// err = l.publishJudgeTask(judgeTask)
	// if err != nil {
	//     l.Logger.Errorf("发送判题任务失败: %v", err)
	//     return nil, fmt.Errorf("提交失败，系统繁忙")
	// }

	// 获取队列位置和预估时间
	queuePosition := l.getQueuePosition()
	estimatedTime := l.getEstimatedTime(queuePosition)

	l.Logger.Infof("用户 %s 提交代码成功, 提交ID: %d, 题目ID: %d, 语言: %s", user.Username, submissionID, req.ProblemID, req.Language)

	return &types.CreateSubmissionResp{
		Code:    200,
		Message: "提交成功",
		Data: types.CreateSubmissionRespData{
			SubmissionID:  submissionID,
			Status:        "pending",
			QueuePosition: queuePosition,
			EstimatedTime: estimatedTime,
			CreatedAt:     time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}, nil
}

// JudgeTask 判题任务
type JudgeTask struct {
	SubmissionID int64     `json:"submission_id"`
	UserID       int64     `json:"user_id"`
	ProblemID    int64     `json:"problem_id"`
	Language     string    `json:"language"`
	Code         string    `json:"code"`
	ContestID    *int64    `json:"contest_id,omitempty"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
}

// isLanguageSupported 检查编程语言是否支持
func (l *CreateSubmissionLogic) isLanguageSupported(language string) bool {
	supportedLanguages := []string{"cpp", "c", "java", "python", "go", "javascript"}
	for _, lang := range supportedLanguages {
		if lang == language {
			return true
		}
	}
	return false
}

// getClientIP 获取客户端IP地址
func (l *CreateSubmissionLogic) getClientIP() string {
	// 这里应该从HTTP请求中获取真实IP
	// 由于go-zero的限制，这里简化处理
	return "127.0.0.1"
}

// getUserAgent 获取用户代理
func (l *CreateSubmissionLogic) getUserAgent() string {
	// 这里应该从HTTP请求头中获取User-Agent
	// 由于go-zero的限制，这里简化处理
	return "unknown"
}

// calculatePriority 计算任务优先级
func (l *CreateSubmissionLogic) calculatePriority(role string, contestID *int64) int {
	// 比赛提交优先级最高
	if contestID != nil {
		return 1
	}

	// 管理员和教师次高优先级
	if role == "admin" || role == "teacher" {
		return 2
	}

	// 普通用户
	return 3
}

// publishJudgeTask 发布判题任务到消息队列
func (l *CreateSubmissionLogic) publishJudgeTask(task *JudgeTask) error {
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化判题任务失败: %v", err)
	}

	return l.svcCtx.MessageQueue.PublishJudgeTask(l.ctx, taskData)
}

// getQueuePosition 获取队列位置
func (l *CreateSubmissionLogic) getQueuePosition() int {
	// 这里应该查询当前队列中的任务数量
	// 简化处理，返回一个估算值
	return 5
}

// getEstimatedTime 获取预估等待时间
func (l *CreateSubmissionLogic) getEstimatedTime(queuePosition int) int {
	// 根据队列位置估算等待时间（秒）
	// 假设每个任务平均耗时6秒
	return queuePosition * 6
}

// 辅助函数
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

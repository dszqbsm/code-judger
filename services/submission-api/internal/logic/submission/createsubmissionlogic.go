package submission

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/queue"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"
	"github.com/dszqbsm/code-judger/services/submission-api/models"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSubmissionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewCreateSubmissionLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *CreateSubmissionLogic {
	return &CreateSubmissionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *CreateSubmissionLogic) CreateSubmission(req *types.CreateSubmissionReq) (resp *types.CreateSubmissionResp, err error) {
	l.Logger.Infof("开始处理创建提交请求: ProblemID=%d, Language=%s, CodeLength=%d", req.ProblemID, req.Language, len(req.Code))

	// 1. 从JWT获取用户信息
	l.Logger.Infof("步骤1: 开始JWT认证")
	user, err := l.getUserFromJWT()
	if err != nil {
		l.Logger.Errorf("获取用户信息失败: %v", err)
		return &types.CreateSubmissionResp{
			Code:    401,
			Message: "认证失败：" + err.Error(),
		}, nil
	}

	l.Logger.Infof("步骤1完成: 用户认证成功: UserID=%d, Username=%s, Role=%s", user.UserID, user.Username, user.Role)

	// 2. 验证提交请求
	l.Logger.Infof("步骤2: 开始验证提交请求")
	if err := l.validateSubmissionRequest(req); err != nil {
		l.Logger.Errorf("提交请求验证失败: %v", err)
		return &types.CreateSubmissionResp{
			Code:    400,
			Message: err.Error(),
		}, nil
	}
	l.Logger.Infof("步骤2完成: 提交请求验证通过")

	// 3. 基本题目ID验证（权限验证已在题目服务完成）
	l.Logger.Infof("步骤3: 开始基本题目验证")
	if req.ProblemID <= 0 {
		l.Logger.Errorf("无效的题目ID: %d", req.ProblemID)
		return &types.CreateSubmissionResp{
			Code:    400,
			Message: "无效的题目ID",
		}, nil
	}
	l.Logger.Infof("步骤3完成: 题目ID验证通过")

	// 4. 检查提交频率限制
	l.Logger.Infof("步骤4: 开始检查提交频率限制")
	if err := l.checkSubmissionRateLimit(user.UserID); err != nil {
		l.Logger.Errorf("提交频率限制检查失败: %v", err)
		return &types.CreateSubmissionResp{
			Code:    429,
			Message: err.Error(),
		}, nil
	}
	l.Logger.Infof("步骤4完成: 提交频率限制检查通过")

	// 5. 获取客户端信息
	clientIP := l.getClientIP()
	_ = l.getUserAgent() // 保留以备将来使用

	// 6. 创建提交记录
	l.Logger.Infof("步骤6: 开始创建提交记录")
	submission := &models.Submission{
		UserID:     user.UserID,
		ProblemID:  req.ProblemID,
		Language:   req.Language,
		Code:       req.Code,
		CodeLength: sql.NullInt32{Int32: int32(len(req.Code)), Valid: true},
		Status:     "pending",
		IPAddress:  sql.NullString{String: clientIP, Valid: true},
		// 其他字段保持默认值（NULL）
	}

	// 设置比赛ID（如果有）
	if req.ContestID > 0 {
		submission.ContestID = sql.NullInt64{Int64: req.ContestID, Valid: true}
	}

	// 7. 通过DAO层创建提交记录
	submissionID, err := l.svcCtx.SubmissionDao.CreateSubmission(l.ctx, submission)
	if err != nil {
		l.Logger.Errorf("创建提交记录失败: %v", err)
		return &types.CreateSubmissionResp{
			Code:    500,
			Message: "提交失败，请稍后重试",
		}, nil
	}

	l.Logger.Infof("步骤6完成: 提交记录创建成功: ID=%d, UserID=%d, ProblemID=%d", submissionID, user.UserID, req.ProblemID)

	// 8. 创建判题任务信息
	l.Logger.Infof("步骤8: 开始创建判题任务信息")
	judgeTaskInfo := &queue.JudgeTaskInfo{
		SubmissionID:  submissionID,
		UserID:        user.UserID,
		ProblemID:     req.ProblemID,
		Priority:      l.calculatePriority(user.Role, req.ContestID),
		CreatedAt:     time.Now(),
		EstimatedTime: 6, // 默认预估6秒
	}
	l.Logger.Infof("步骤8完成: 判题任务信息创建完成")

	// 9. 添加任务到队列管理器
	l.Logger.Infof("步骤9: 开始添加任务到队列管理器")
	queuePosition, err := l.svcCtx.QueueManager.AddTask(l.ctx, judgeTaskInfo)
	if err != nil {
		l.Logger.Errorf("添加任务到队列失败: %v", err)
		l.svcCtx.SubmissionDao.UpdateSubmissionStatus(l.ctx, submissionID, "system_error")
		queuePosition = 1 // 使用默认值
	}
	l.Logger.Infof("步骤9完成: 任务已添加到队列，位置: %d", queuePosition)

	// 10. 创建简化的判题任务并发送到消息队列（题目信息由判题服务获取）
	l.Logger.Infof("步骤10: 开始创建判题任务并发送到Kafka")
	judgeTask := &JudgeTask{
		SubmissionID: submissionID,
		UserID:       user.UserID,
		ProblemID:    req.ProblemID,
		Language:     req.Language,
		Code:         req.Code,
		Priority:     judgeTaskInfo.Priority,
		CreatedAt:    time.Now(),
	}

	l.Logger.Infof("步骤10.1: 开始发布Kafka消息")
	// 使用独立的context避免HTTP请求超时影响Kafka操作
	kafkaCtx, kafkaCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer kafkaCancel()

	err = l.publishJudgeTaskWithContext(kafkaCtx, judgeTask)
	if err != nil {
		l.Logger.Errorf("发送判题任务失败: %v", err)
		// 注意：这里不返回错误，因为提交记录已经创建成功
		// 可以考虑后续补偿机制或者设置状态为failed
		l.svcCtx.SubmissionDao.UpdateSubmissionStatus(l.ctx, submissionID, "system_error")
	} else {
		l.Logger.Infof("步骤10完成: Kafka消息发布成功")
	}

	// 11. 获取预估等待时间
	estimatedTime, err := l.svcCtx.QueueManager.GetEstimatedWaitTime(l.ctx, queuePosition)
	if err != nil {
		l.Logger.Errorf("获取预估等待时间失败: %v", err)
		estimatedTime = l.getEstimatedTime(queuePosition) // 使用备用计算方法
	}

	l.Logger.Infof("用户 %s 提交代码成功: 提交ID=%d, 题目ID=%d, 语言=%s, 队列位置=%d, 预估时间=%ds",
		user.Username, submissionID, req.ProblemID, req.Language, queuePosition, estimatedTime)

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

// JudgeTask 判题任务（简化版，题目详细信息由判题服务获取）
type JudgeTask struct {
	SubmissionID int64     `json:"submission_id"`
	UserID       int64     `json:"user_id"`
	ProblemID    int64     `json:"problem_id"`
	Language     string    `json:"language"`
	Code         string    `json:"code"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
}

// TestCase 测试用例结构体（保留给rejudge逻辑兼容）
type TestCase struct {
	CaseID   int    `json:"case_id"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

// ProblemInfo 题目信息结构体（保留给rejudge逻辑兼容）
type ProblemInfo struct {
	TimeLimit   int        `json:"time_limit"`   // 毫秒
	MemoryLimit int        `json:"memory_limit"` // MB
	TestCases   []TestCase `json:"test_cases"`
}

// getUserFromJWT 从JWT中获取用户信息
func (l *CreateSubmissionLogic) getUserFromJWT() (*middleware.UserInfo, error) {
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

// validateSubmissionRequest 验证提交请求
func (l *CreateSubmissionLogic) validateSubmissionRequest(req *types.CreateSubmissionReq) error {
	// 验证题目ID
	if req.ProblemID <= 0 {
		return fmt.Errorf("无效的题目ID")
	}

	// 验证编程语言是否支持
	if !l.isLanguageSupported(req.Language) {
		return fmt.Errorf("不支持的编程语言: %s", req.Language)
	}

	// 验证代码长度
	if len(req.Code) == 0 {
		return fmt.Errorf("代码不能为空")
	}

	if len(req.Code) > l.svcCtx.Config.Business.MaxCodeLength {
		return fmt.Errorf("代码长度超出限制，最大允许 %d 字符", l.svcCtx.Config.Business.MaxCodeLength)
	}

	// 验证代码内容（基本安全检查）
	if err := l.validateCodeContent(req.Code); err != nil {
		return err
	}

	return nil
}

// 移除validateProblemAccess函数，权限验证已在题目服务完成

// checkSubmissionRateLimit 检查提交频率限制
func (l *CreateSubmissionLogic) checkSubmissionRateLimit(userID int64) error {
	// 使用Redis检查提交频率限制
	key := fmt.Sprintf("submission_rate_limit:%d", userID)

	// 检查一分钟内的提交次数
	count, err := l.svcCtx.RedisClient.Incr(key)
	if err != nil {
		l.Logger.Errorf("检查提交频率限制失败: %v", err)
		// 如果Redis出错，允许提交但记录日志
		return nil
	}

	// 设置过期时间为60秒
	if count == 1 {
		l.svcCtx.RedisClient.Expire(key, 60)
	}

	// 检查是否超过限制
	maxSubmissions := l.svcCtx.Config.Business.MaxSubmissionPerMinute
	if int(count) > maxSubmissions {
		return fmt.Errorf("提交过于频繁，请等待 %d 秒后再试", 60)
	}

	return nil
}

// isLanguageSupported 检查编程语言是否支持
func (l *CreateSubmissionLogic) isLanguageSupported(language string) bool {
	supportedLanguages := l.svcCtx.Config.Submission.SupportedLanguages
	if len(supportedLanguages) == 0 {
		// 如果没有配置支持的语言，返回false
		return false
	}

	for _, langConf := range supportedLanguages {
		if langConf.Name == language && langConf.Enabled {
			return true
		}
	}
	return false
}

// validateCodeContent 验证代码内容
func (l *CreateSubmissionLogic) validateCodeContent(code string) error {
	// 检查恶意代码模式
	maliciousPatterns := []string{
		"system(",
		"exec(",
		"eval(",
		"__import__",
		"subprocess",
		"os.system",
		"Runtime.getRuntime",
	}

	lowerCode := strings.ToLower(code)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(lowerCode, strings.ToLower(pattern)) {
			return fmt.Errorf("代码包含不被允许的系统调用")
		}
	}

	return nil
}

// getClientIP 获取客户端真实IP地址
func (l *CreateSubmissionLogic) getClientIP() string {
	if l.r == nil {
		return "unknown"
	}

	// 优先级顺序获取真实IP
	headers := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Client-IP",
		"CF-Connecting-IP", // Cloudflare
	}

	for _, header := range headers {
		ip := l.r.Header.Get(header)
		if ip != "" && ip != "unknown" {
			// X-Forwarded-For 可能包含多个IP，取第一个
			if header == "X-Forwarded-For" {
				ips := strings.Split(ip, ",")
				if len(ips) > 0 {
					return strings.TrimSpace(ips[0])
				}
			}
			return ip
		}
	}

	// 最后使用RemoteAddr
	if l.r.RemoteAddr != "" {
		ip := strings.Split(l.r.RemoteAddr, ":")[0]
		return ip
	}

	return "unknown"
}

// getUserAgent 获取用户代理
func (l *CreateSubmissionLogic) getUserAgent() string {
	if l.r == nil {
		return "unknown"
	}

	userAgent := l.r.Header.Get("User-Agent")
	if userAgent == "" {
		return "unknown"
	}

	// 限制User-Agent长度
	if len(userAgent) > 500 {
		return userAgent[:500]
	}

	return userAgent
}

// calculatePriority 计算任务优先级
func (l *CreateSubmissionLogic) calculatePriority(role string, contestID int64) int {
	// 比赛提交优先级最高
	if contestID > 0 {
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
	return l.svcCtx.MessageQueue.PublishJudgeTask(l.ctx, task)
}

// publishJudgeTaskWithContext 使用指定context发布判题任务到消息队列
func (l *CreateSubmissionLogic) publishJudgeTaskWithContext(ctx context.Context, task *JudgeTask) error {
	return l.svcCtx.MessageQueue.PublishJudgeTask(ctx, task)
}

// getQueuePosition 获取真实的队列位置
func (l *CreateSubmissionLogic) getQueuePosition() (int, error) {
	// 从Redis获取当前队列长度
	queueKey := "judge_queue_length"
	length, err := l.svcCtx.RedisClient.Llen(queueKey)
	if err != nil {
		l.Logger.Errorf("获取队列长度失败: %v", err)
		return 1, err
	}

	// 队列位置 = 当前队列长度 + 1
	position := int(length) + 1

	// 确保位置至少为1
	if position < 1 {
		position = 1
	}

	return position, nil
}

// getEstimatedTime 获取预估等待时间
func (l *CreateSubmissionLogic) getEstimatedTime(queuePosition int) int {
	// 从配置或Redis获取平均判题时间
	avgJudgeTime := l.svcCtx.Config.Business.AverageJudgeTime
	if avgJudgeTime <= 0 {
		avgJudgeTime = 6 // 默认6秒
	}

	// 考虑并发判题服务器数量
	concurrentJudges := l.svcCtx.Config.Business.ConcurrentJudges
	if concurrentJudges <= 0 {
		concurrentJudges = 1
	}

	// 预估时间 = (队列位置 / 并发数) * 平均判题时间
	estimatedTime := (queuePosition / concurrentJudges) * avgJudgeTime

	// 至少需要平均判题时间
	if estimatedTime < avgJudgeTime {
		estimatedTime = avgJudgeTime
	}

	return estimatedTime
}

// 移除getProblemInfo和getMockProblemInfo函数，题目信息获取由判题服务负责

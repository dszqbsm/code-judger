package submission

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/queue"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejudgeSubmissionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewRejudgeSubmissionLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *RejudgeSubmissionLogic {
	return &RejudgeSubmissionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *RejudgeSubmissionLogic) RejudgeSubmission(req *types.RejudgeSubmissionReq) (resp *types.RejudgeSubmissionResp, err error) {
	l.Logger.Infof("开始处理重新判题请求: SubmissionID=%d", req.SubmissionID)

	// 1. 验证提交ID
	if req.SubmissionID <= 0 {
		l.Logger.Errorf("无效的提交ID: %d", req.SubmissionID)
		return &types.RejudgeSubmissionResp{
			Code:    400,
			Message: "无效的提交ID",
		}, nil
	}

	// 2. 从JWT获取用户信息并验证权限
	user, err := l.getUserFromJWT()
	if err != nil {
		l.Logger.Errorf("获取用户信息失败: %v", err)
		return &types.RejudgeSubmissionResp{
			Code:    401,
			Message: "认证失败：" + err.Error(),
		}, nil
	}

	// 3. 权限检查：只有管理员可以重新判题
	if user.Role != "admin" {
		l.Logger.Errorf("用户 %s (Role: %s) 权限不足，无法重新判题", user.Username, user.Role)
		return &types.RejudgeSubmissionResp{
			Code:    403,
			Message: "权限不足：只有管理员可以重新判题",
		}, nil
	}

	l.Logger.Infof("管理员用户 %s 请求重新判题: SubmissionID=%d", user.Username, req.SubmissionID)

	// 4. 验证提交记录是否存在
	submission, err := l.svcCtx.SubmissionDao.GetSubmissionByID(l.ctx, req.SubmissionID)
	if err != nil {
		l.Logger.Errorf("查询提交记录失败: %v", err)
		return &types.RejudgeSubmissionResp{
			Code:    404,
			Message: "提交记录不存在",
		}, nil
	}

	l.Logger.Infof("找到提交记录: ID=%d, UserID=%d, ProblemID=%d, Status=%s",
		submission.ID, submission.UserID, submission.ProblemID, submission.Status)

	// 5. 检查提交状态
	if submission.Status == "pending" || submission.Status == "judging" {
		l.Logger.Infof("提交正在判题中，无需重新判题: SubmissionID=%d, Status=%s", req.SubmissionID, submission.Status)
		return &types.RejudgeSubmissionResp{
			Code:    400,
			Message: "提交正在判题中，无需重新判题",
		}, nil
	}

	// 6. 清理原有判题结果
	err = l.clearPreviousJudgeResult(req.SubmissionID)
	if err != nil {
		l.Logger.Errorf("清理原有判题结果失败: %v", err)
		// 不返回错误，继续执行
	}

	// 7. 更新提交状态为pending
	err = l.svcCtx.SubmissionDao.UpdateSubmissionStatus(l.ctx, req.SubmissionID, "pending")
	if err != nil {
		l.Logger.Errorf("更新提交状态失败: %v", err)
		return &types.RejudgeSubmissionResp{
			Code:    500,
			Message: "重新判题失败，请稍后重试",
		}, nil
	}

	// 8. 创建重新判题任务信息
	judgeTaskInfo := &queue.JudgeTaskInfo{
		SubmissionID:  req.SubmissionID,
		UserID:        submission.UserID,
		ProblemID:     submission.ProblemID,
		Priority:      1, // 重新判题任务优先级最高
		CreatedAt:     time.Now(),
		EstimatedTime: 6, // 默认预估6秒
	}

	// 9. 添加任务到队列管理器
	queuePosition, err := l.svcCtx.QueueManager.AddTask(l.ctx, judgeTaskInfo)
	if err != nil {
		l.Logger.Errorf("添加重新判题任务到队列失败: %v", err)
		// 恢复原状态
		l.svcCtx.SubmissionDao.UpdateSubmissionStatus(l.ctx, req.SubmissionID, submission.Status)
		return &types.RejudgeSubmissionResp{
			Code:    500,
			Message: "重新判题失败，系统繁忙",
		}, nil
	}

	// 10. 创建简化的重新判题任务（题目信息由判题服务获取）
	l.Logger.Infof("步骤10: 开始创建重新判题任务")
	judgeTask := &JudgeTask{
		SubmissionID: req.SubmissionID,
		UserID:       submission.UserID,
		ProblemID:    submission.ProblemID,
		Language:     submission.Language,
		Code:         submission.Code,
		Priority:     1, // 重新判题任务优先级最高
		CreatedAt:    time.Now(),
	}

	// 11. 发送到消息队列
	err = l.publishRejudgeTask(judgeTask)
	if err != nil {
		l.Logger.Errorf("发送重新判题任务失败: %v", err)
		// 恢复原状态
		l.svcCtx.SubmissionDao.UpdateSubmissionStatus(l.ctx, req.SubmissionID, submission.Status)
		return &types.RejudgeSubmissionResp{
			Code:    500,
			Message: "重新判题失败，系统繁忙",
		}, nil
	}

	// 12. 获取预估等待时间
	estimatedTime, err := l.svcCtx.QueueManager.GetEstimatedWaitTime(l.ctx, queuePosition)
	if err != nil {
		l.Logger.Errorf("获取预估等待时间失败: %v", err)
		estimatedTime = queuePosition * 6 // 使用备用计算方法
	}

	l.Logger.Infof("管理员 %s 重新判题任务创建成功: SubmissionID=%d, QueuePosition=%d, EstimatedTime=%ds",
		user.Username, req.SubmissionID, queuePosition, estimatedTime)

	return &types.RejudgeSubmissionResp{
		Code:    200,
		Message: "重新判题任务已提交",
		Data: types.RejudgeSubmissionRespData{
			SubmissionID:  req.SubmissionID,
			Status:        "pending",
			Message:       "已提交重新判题任务，请稍后查看结果",
			QueuePosition: queuePosition,
			EstimatedTime: estimatedTime,
		},
	}, nil
}

// getUserFromJWT 从JWT中获取用户信息
func (l *RejudgeSubmissionLogic) getUserFromJWT() (*middleware.UserInfo, error) {
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

// clearPreviousJudgeResult 清理原有判题结果
func (l *RejudgeSubmissionLogic) clearPreviousJudgeResult(submissionID int64) error {
	// 简化实现：通过更新状态来标记需要重新判题
	// 具体的结果清理可以在判题服务中处理
	l.Logger.Infof("标记提交 %d 需要重新判题", submissionID)

	// TODO: 如果需要清理具体的判题结果字段，可以调用DAO方法
	// 这里暂时只记录日志，实际清理在判题开始时进行

	return nil
}

// publishRejudgeTask 发布重新判题任务到消息队列
func (l *RejudgeSubmissionLogic) publishRejudgeTask(task *JudgeTask) error {
	// 重新判题任务使用特殊的主题或标记
	return l.svcCtx.MessageQueue.PublishJudgeTask(l.ctx, task, "rejudge")
}

// getClientIP 获取客户端IP地址
func (l *RejudgeSubmissionLogic) getClientIP() string {
	if l.r == nil {
		return "admin_console"
	}

	// 优先级顺序获取真实IP
	headers := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Client-IP",
		"CF-Connecting-IP",
	}

	for _, header := range headers {
		ip := l.r.Header.Get(header)
		if ip != "" && ip != "unknown" {
			if header == "X-Forwarded-For" {
				// X-Forwarded-For 可能包含多个IP，取第一个
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

	return "admin_console"
}

// getUserAgent 获取用户代理
func (l *RejudgeSubmissionLogic) getUserAgent() string {
	if l.r == nil {
		return "admin_console"
	}

	userAgent := l.r.Header.Get("User-Agent")
	if userAgent == "" {
		return "admin_console"
	}

	// 限制User-Agent长度
	if len(userAgent) > 500 {
		return userAgent[:500]
	}

	return userAgent
}



// 移除getProblemInfo函数，题目信息获取由判题服务负责

package submission

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/middleware"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetQueueStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetQueueStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetQueueStatsLogic {
	return &GetQueueStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetQueueStatsLogic) GetQueueStats(req *types.GetQueueStatsReq) (resp *types.GetQueueStatsResp, err error) {
	l.Logger.Infof("开始处理获取队列统计请求")

	// 1. 从JWT获取用户信息
	user, err := l.getUserFromJWT()
	if err != nil {
		l.Logger.Errorf("获取用户信息失败: %v", err)
		return &types.GetQueueStatsResp{
			Code:    401,
			Message: "认证失败：" + err.Error(),
		}, nil
	}

	// 2. 权限检查：只有管理员和教师可以查看队列统计
	if user.Role != "admin" && user.Role != "teacher" {
		l.Logger.Errorf("用户 %s (Role: %s) 权限不足，无法查看队列统计", user.Username, user.Role)
		return &types.GetQueueStatsResp{
			Code:    403,
			Message: "权限不足：只有管理员和教师可以查看队列统计",
		}, nil
	}

	// 3. 获取队列统计信息
	stats, err := l.svcCtx.QueueManager.GetQueueStats(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取队列统计失败: %v", err)
		return &types.GetQueueStatsResp{
			Code:    500,
			Message: "获取队列统计失败",
		}, nil
	}

	// 4. 获取当前队列长度
	currentQueueLength, err := l.svcCtx.QueueManager.GetCurrentQueueLength(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取当前队列长度失败: %v", err)
		currentQueueLength = 0
	}

	l.Logger.Infof("用户 %s 成功获取队列统计: 总任务=%d, 待处理=%d, 处理中=%d, 已完成=%d", 
		user.Username, stats.TotalTasks, stats.PendingTasks, stats.ProcessingTasks, stats.CompletedTasks)

	return &types.GetQueueStatsResp{
		Code:    200,
		Message: "获取成功",
		Data: types.GetQueueStatsRespData{
			TotalTasks:        stats.TotalTasks,
			PendingTasks:      stats.PendingTasks,
			ProcessingTasks:   stats.ProcessingTasks,
			CompletedTasks:    stats.CompletedTasks,
			CurrentQueueLength: int64(currentQueueLength),
			AverageWaitTime:   stats.AverageWaitTime,
			AverageJudgeTime:  stats.AverageJudgeTime,
			ActiveJudges:      stats.ActiveJudges,
		},
	}, nil
}

// getUserFromJWT 从JWT中获取用户信息
func (l *GetQueueStatsLogic) getUserFromJWT() (*middleware.UserInfo, error) {
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

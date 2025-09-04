package judge

import (
	"context"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/service"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelJudgeLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	taskService *service.TaskService
}

func NewCancelJudgeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelJudgeLogic {
	return &CancelJudgeLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		taskService: service.NewTaskService(ctx, svcCtx),
	}
}

func (l *CancelJudgeLogic) CancelJudge(req *types.CancelJudgeReq) (resp *types.CancelJudgeResp, err error) {
	l.Logger.Infof("开始取消判题任务: SubmissionID=%d", req.SubmissionId)

	// 1. 验证提交ID
	if req.SubmissionId <= 0 {
		l.Logger.Errorf("无效的提交ID: %d", req.SubmissionId)
		return &types.CancelJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: "无效的提交ID",
			},
		}, nil
	}

	// 2. 从任务服务获取任务信息（真实业务逻辑）
	task, err := l.taskService.FindTaskBySubmissionID(req.SubmissionId)
	if err != nil {
		l.Logger.Errorf("未找到提交ID %d 的判题任务: %v", req.SubmissionId, err)
		return &types.CancelJudgeResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "未找到判题任务",
			},
		}, nil
	}

	// 3. TODO: 权限验证（需要从JWT或请求中获取用户信息）
	// userID := getUserIDFromContext(l.ctx)
	// userRole := getUserRoleFromContext(l.ctx)
	// if err := l.taskService.ValidateTaskAccess(task, userID, userRole); err != nil {
	//     l.Logger.Errorf("用户权限验证失败: %v", err)
	//     return &types.CancelJudgeResp{
	//         BaseResp: types.BaseResp{
	//             Code:    403,
	//             Message: "权限不足：无法取消该判题任务",
	//         },
	//     }, nil
	// }

	// 4. 检查任务是否可以取消
	if !l.taskService.IsTaskCancellable(task) {
		statusText := l.taskService.GetTaskStatusText(task.Status)
		l.Logger.Errorf("任务不可取消: SubmissionID=%d, Status=%s", req.SubmissionId, task.Status)
		return &types.CancelJudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: "任务已" + statusText + "，无法取消",
			},
		}, nil
	}

	// 5. 调用调度器取消任务
	if err := l.svcCtx.TaskScheduler.CancelTask(task.ID); err != nil {
		l.Logger.Errorf("调度器取消任务失败: TaskID=%s, Error=%v", task.ID, err)
		return &types.CancelJudgeResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "取消判题任务失败，请稍后重试",
			},
		}, nil
	}

	// 6. 记录取消操作日志
	l.Logger.Infof("判题任务取消成功: SubmissionID=%d, TaskID=%s, 原状态=%s", 
		req.SubmissionId, task.ID, task.Status)

	// 7. 构建成功响应
	return &types.CancelJudgeResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "判题任务已取消",
		},
		Data: types.CancelJudgeData{
			SubmissionId: req.SubmissionId,
			Status:       "cancelled",
			Message:      "任务已成功取消",
			CancelledAt:  l.getCurrentTimeString(),
		},
	}, nil
}

// getCurrentTimeString 获取当前时间字符串
func (l *CancelJudgeLogic) getCurrentTimeString() string {
	return time.Now().Format("2006-01-02T15:04:05Z07:00")
}
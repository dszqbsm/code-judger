package judge

import (
	"context"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/service"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetJudgeResultLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	taskService *service.TaskService
}

func NewGetJudgeResultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetJudgeResultLogic {
	return &GetJudgeResultLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		taskService: service.NewTaskService(ctx, svcCtx),
	}
}

func (l *GetJudgeResultLogic) GetJudgeResult(req *types.GetJudgeResultReq) (resp *types.GetJudgeResultResp, err error) {
	l.Logger.Infof("开始获取判题结果: SubmissionID=%d", req.SubmissionId)

	// 1. 验证提交ID
	if req.SubmissionId <= 0 {
		l.Logger.Errorf("无效的提交ID: %d", req.SubmissionId)
		return &types.GetJudgeResultResp{
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
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "未找到判题结果",
			},
		}, nil
	}

	// 3. TODO: 权限验证（需要从JWT或请求中获取用户信息）
	// userID := getUserIDFromContext(l.ctx)
	// userRole := getUserRoleFromContext(l.ctx)
	// if err := l.taskService.ValidateTaskAccess(task, userID, userRole); err != nil {
	//     l.Logger.Errorf("用户权限验证失败: %v", err)
	//     return &types.GetJudgeResultResp{
	//         BaseResp: types.BaseResp{
	//             Code:    403,
	//             Message: "权限不足：无法查看该判题结果",
	//         },
	//     }, nil
	// }

	// 4. 检查任务是否完成
	if !l.taskService.IsTaskCompleted(task) {
		l.Logger.Infof("判题任务尚未完成: SubmissionID=%d, Status=%s", req.SubmissionId, task.Status)
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    202,
				Message: "判题尚未完成",
			},
			Data: types.JudgeResult{
				SubmissionId: req.SubmissionId,
				Status:       l.convertTaskStatusToJudgeStatus(task.Status),
			},
		}, nil
	}

	// 5. 任务失败处理
	if task.Status == "failed" {
		l.Logger.Errorf("判题任务失败: SubmissionID=%d, Error=%s", req.SubmissionId, task.Error)
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "判题失败",
			},
			Data: types.JudgeResult{
				SubmissionId: req.SubmissionId,
				Status:       "system_error",
				ErrorMessage: task.Error,
			},
		}, nil
	}

	// 6. 任务取消处理
	if task.Status == "cancelled" {
		l.Logger.Infof("判题任务已取消: SubmissionID=%d", req.SubmissionId)
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    200,
				Message: "判题已取消",
			},
			Data: types.JudgeResult{
				SubmissionId: req.SubmissionId,
				Status:       "cancelled",
			},
		}, nil
	}

	// 7. 验证判题结果存在
	if task.Result == nil {
		l.Logger.Errorf("判题结果不存在: SubmissionID=%d", req.SubmissionId)
		return &types.GetJudgeResultResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "判题结果不存在",
			},
		}, nil
	}

	// 8. 返回完整的判题结果
	result := task.Result
	
	l.Logger.Infof("成功获取判题结果: SubmissionID=%d, Status=%s, Score=%d, TimeUsed=%dms, MemoryUsed=%dKB",
		req.SubmissionId, result.Status, result.Score, result.TimeUsed, result.MemoryUsed)

	return &types.GetJudgeResultResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "获取成功",
		},
		Data: *result,
	}, nil
}

// convertTaskStatusToJudgeStatus 将任务状态转换为判题状态
func (l *GetJudgeResultLogic) convertTaskStatusToJudgeStatus(taskStatus string) string {
	switch taskStatus {
	case "pending":
		return "waiting"
	case "running":
		return "judging"
	case "completed":
		return "finished"
	case "failed":
		return "system_error"
	case "cancelled":
		return "cancelled"
	default:
		return taskStatus
	}
}
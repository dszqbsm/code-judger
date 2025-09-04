package judge

import (
	"context"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/service"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetJudgeStatusLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	taskService *service.TaskService
}

func NewGetJudgeStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetJudgeStatusLogic {
	return &GetJudgeStatusLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		taskService: service.NewTaskService(ctx, svcCtx),
	}
}

func (l *GetJudgeStatusLogic) GetJudgeStatus(req *types.GetJudgeStatusReq) (resp *types.GetJudgeStatusResp, err error) {
	l.Logger.Infof("开始获取判题状态: SubmissionID=%d", req.SubmissionId)

	// 1. 验证提交ID
	if req.SubmissionId <= 0 {
		l.Logger.Errorf("无效的提交ID: %d", req.SubmissionId)
		return &types.GetJudgeStatusResp{
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
		return &types.GetJudgeStatusResp{
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
	//     return &types.GetJudgeStatusResp{
	//         BaseResp: types.BaseResp{
	//             Code:    403,
	//             Message: "权限不足：无法查看该判题状态",
	//         },
	//     }, nil
	// }

	// 4. 计算任务执行进度
	progress := l.taskService.CalculateTaskProgress(task)

	// 5. 确定当前测试用例和总数
	currentTestCase := 0
	totalTestCases := len(task.TestCases)

	if task.Result != nil && len(task.Result.TestCases) > 0 {
		currentTestCase = len(task.Result.TestCases)
	}

	// 6. 生成状态消息
	message := l.taskService.GenerateTaskStatusMessage(task)

	// 7. 获取队列位置信息（如果任务还在等待中）
	var queuePosition *int
	var estimatedTime *int
	
	if task.Status == "pending" {
		if pos, err := l.svcCtx.TaskScheduler.GetTaskPosition(task.ID); err == nil {
			queuePosition = &pos
			// 简单估算等待时间（位置 * 平均执行时间）
			avgTime := 30 // 30秒平均执行时间
			estimated := pos * avgTime
			estimatedTime = &estimated
		}
	}

	// 8. 记录成功日志
	l.Logger.Infof("成功获取判题状态: SubmissionID=%d, Status=%s, Progress=%d%%, CurrentCase=%d/%d",
		req.SubmissionId, task.Status, progress, currentTestCase, totalTestCases)

	// 9. 构建响应数据
	judgeStatus := types.JudgeStatus{
		SubmissionId:    req.SubmissionId,
		Status:          task.Status,
		Progress:        progress,
		CurrentTestCase: currentTestCase,
		TotalTestCases:  totalTestCases,
		Message:         message,
	}

	// 添加队列信息（如果有）
	if queuePosition != nil {
		judgeStatus.QueuePosition = queuePosition
	}
	if estimatedTime != nil {
		judgeStatus.EstimatedTime = estimatedTime
	}

	// 添加错误信息（如果有）
	if task.Error != "" {
		judgeStatus.ErrorMessage = &task.Error
	}

	// 添加执行时间信息（如果任务正在运行或已完成）
	if task.StartedAt != nil {
		startTime := task.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		judgeStatus.StartTime = &startTime
	}

	if task.CompletedAt != nil {
		endTime := task.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		judgeStatus.EndTime = &endTime
	}

	return &types.GetJudgeStatusResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "获取成功",
		},
		Data: judgeStatus,
	}, nil
}
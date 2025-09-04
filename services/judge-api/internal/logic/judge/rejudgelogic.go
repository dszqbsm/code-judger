package judge

import (
	"context"
	"fmt"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/scheduler"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/service"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejudgeLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	taskService *service.TaskService
}

func NewRejudgeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejudgeLogic {
	return &RejudgeLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		taskService: service.NewTaskService(ctx, svcCtx),
	}
}

func (l *RejudgeLogic) Rejudge(req *types.RejudgeReq) (resp *types.RejudgeResp, err error) {
	l.Logger.Infof("开始重新判题: SubmissionID=%d", req.SubmissionId)

	// 1. 验证提交ID
	if req.SubmissionId <= 0 {
		l.Logger.Errorf("无效的提交ID: %d", req.SubmissionId)
		return &types.RejudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: "无效的提交ID",
			},
		}, nil
	}

	// 2. 查找原始任务信息
	originalTask, err := l.taskService.FindTaskBySubmissionID(req.SubmissionId)
	if err != nil {
		l.Logger.Errorf("未找到提交ID %d 的原始任务: %v", req.SubmissionId, err)
		return &types.RejudgeResp{
			BaseResp: types.BaseResp{
				Code:    404,
				Message: "未找到原始提交记录",
			},
		}, nil
	}

	// 3. TODO: 权限验证（需要从JWT或请求中获取用户信息）
	// userID := getUserIDFromContext(l.ctx)
	// userRole := getUserRoleFromContext(l.ctx)
	// if err := l.validateRejudgePermission(originalTask, userID, userRole); err != nil {
	//     l.Logger.Errorf("重新判题权限验证失败: %v", err)
	//     return &types.RejudgeResp{
	//         BaseResp: types.BaseResp{
	//             Code:    403,
	//             Message: "权限不足：无法重新判题该提交",
	//         },
	//     }, nil
	// }

	// 4. 检查是否可以重新判题
	if err := l.validateRejudgeEligibility(originalTask); err != nil {
		l.Logger.Errorf("重新判题资格验证失败: %v", err)
		return &types.RejudgeResp{
			BaseResp: types.BaseResp{
				Code:    400,
				Message: err.Error(),
			},
		}, nil
	}

	// 5. 取消现有任务（如果还在进行中）
	if err := l.cancelExistingTask(originalTask); err != nil {
		l.Logger.Errorf("取消现有任务失败: %v", err)
		// 不中断流程，记录警告即可
	}

	// 6. 重新获取题目信息（可能测试用例已更新）
	problemInfo, err := l.getProblemInfoForRejudge(originalTask.ProblemID)
	if err != nil {
		l.Logger.Errorf("重新获取题目信息失败: ProblemID=%d, Error=%v", originalTask.ProblemID, err)
		return &types.RejudgeResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "获取题目信息失败，无法重新判题",
			},
		}, nil
	}

	// 7. 创建重新判题任务
	rejudgeTask, err := l.createRejudgeTask(originalTask, problemInfo)
	if err != nil {
		l.Logger.Errorf("创建重新判题任务失败: %v", err)
		return &types.RejudgeResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "创建重新判题任务失败",
			},
		}, nil
	}

	// 8. 提交任务到调度器
	if err := l.svcCtx.TaskScheduler.SubmitTask(rejudgeTask); err != nil {
		l.Logger.Errorf("提交重新判题任务失败: %v", err)
		return &types.RejudgeResp{
			BaseResp: types.BaseResp{
				Code:    500,
				Message: "提交重新判题任务失败",
			},
		}, nil
	}

	// 9. 获取队列位置信息
	queuePosition, err := l.svcCtx.TaskScheduler.GetTaskPosition(rejudgeTask.ID)
	if err != nil {
		l.Logger.Errorf("获取队列位置失败: %v", err)
		// 使用队列长度作为备用值
		queueStatus := l.svcCtx.TaskScheduler.GetQueueStatus()
		queuePosition = queueStatus.QueueLength
	}

	// 10. 计算预估等待时间
	estimatedTime := l.calculateEstimatedTime(queuePosition)

	// 11. 记录成功日志
	l.Logger.Infof("重新判题任务提交成功: SubmissionID=%d, NewTaskID=%s, QueuePosition=%d, EstimatedTime=%ds",
		req.SubmissionId, rejudgeTask.ID, queuePosition, estimatedTime)

	// 12. 构建响应
	return &types.RejudgeResp{
		BaseResp: types.BaseResp{
			Code:    200,
			Message: "重新判题任务已提交",
		},
		Data: types.RejudgeData{
			SubmissionId:  req.SubmissionId,
			Status:        "pending",
			Message:       fmt.Sprintf("重新判题任务已加入队列，当前位置: %d, 预估等待时间: %ds", queuePosition, estimatedTime),
			TaskId:        rejudgeTask.ID,
			QueuePosition: queuePosition,
			EstimatedTime: estimatedTime,
			CreatedAt:     time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}, nil
}

// validateRejudgePermission 验证重新判题权限
func (l *RejudgeLogic) validateRejudgePermission(task *scheduler.JudgeTask, userID int64, userRole string) error {
	// 1. 管理员可以重新判题任何提交
	if userRole == "admin" {
		return nil
	}

	// 2. 教师可以重新判题（根据业务需求）
	if userRole == "teacher" {
		return nil
	}

	// 3. 用户只能重新判题自己的提交
	if task.UserID == userID {
		return nil
	}

	return fmt.Errorf("权限不足：只能重新判题自己的提交")
}

// validateRejudgeEligibility 验证重新判题资格
func (l *RejudgeLogic) validateRejudgeEligibility(task *scheduler.JudgeTask) error {
	if task == nil {
		return fmt.Errorf("原始任务不存在")
	}

	// 检查任务年龄（可选：限制只能重新判题最近的提交）
	if task.CreatedAt.Before(time.Now().AddDate(0, 0, -30)) {
		return fmt.Errorf("该提交过于久远（超过30天），无法重新判题")
	}

	// 检查是否存在必要的信息
	if task.Code == "" {
		return fmt.Errorf("原始代码不存在，无法重新判题")
	}

	if task.Language == "" {
		return fmt.Errorf("编程语言信息缺失，无法重新判题")
	}

	if task.ProblemID <= 0 {
		return fmt.Errorf("题目ID无效，无法重新判题")
	}

	return nil
}

// cancelExistingTask 取消现有任务（如果还在进行中）
func (l *RejudgeLogic) cancelExistingTask(task *scheduler.JudgeTask) error {
	// 如果任务已经完成或失败，不需要取消
	if l.taskService.IsTaskCompleted(task) {
		l.Logger.Infof("原始任务已完成，无需取消: TaskID=%s, Status=%s", task.ID, task.Status)
		return nil
	}

	// 如果任务还在进行中，尝试取消
	if l.taskService.IsTaskCancellable(task) {
		l.Logger.Infof("取消现有任务: TaskID=%s, Status=%s", task.ID, task.Status)
		return l.svcCtx.TaskScheduler.CancelTask(task.ID)
	}

	return nil
}

// getProblemInfoForRejudge 重新获取题目信息（可能测试用例已更新）
func (l *RejudgeLogic) getProblemInfoForRejudge(problemID int64) (*types.ProblemInfo, error) {
	l.Logger.Infof("重新获取题目信息: ProblemID=%d", problemID)

	// 调用题目服务获取最新的题目信息
	problemInfo, err := l.svcCtx.ProblemClient.GetProblemDetail(l.ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("调用题目服务失败: %w", err)
	}

	// 验证题目信息
	if err := l.validateProblemInfo(problemInfo); err != nil {
		return nil, fmt.Errorf("题目信息验证失败: %w", err)
	}

	l.Logger.Infof("成功获取题目信息: ProblemID=%d, TestCases=%d", problemID, len(problemInfo.TestCases))
	return problemInfo, nil
}

// validateProblemInfo 验证题目信息
func (l *RejudgeLogic) validateProblemInfo(problemInfo *types.ProblemInfo) error {
	if problemInfo == nil {
		return fmt.Errorf("题目信息为空")
	}

	if len(problemInfo.TestCases) == 0 {
		return fmt.Errorf("题目测试用例为空")
	}

	if problemInfo.TimeLimit <= 0 {
		return fmt.Errorf("时间限制无效: %d", problemInfo.TimeLimit)
	}

	if problemInfo.MemoryLimit <= 0 {
		return fmt.Errorf("内存限制无效: %d", problemInfo.MemoryLimit)
	}

	return nil
}

// createRejudgeTask 创建重新判题任务
func (l *RejudgeLogic) createRejudgeTask(originalTask *scheduler.JudgeTask, problemInfo *types.ProblemInfo) (*scheduler.JudgeTask, error) {
	// 转换测试用例
	testCases := make([]*types.TestCase, len(problemInfo.TestCases))
	for i, tc := range problemInfo.TestCases {
		testCases[i] = &types.TestCase{
			CaseId:         tc.CaseId,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			TimeLimit:      tc.TimeLimit,
			MemoryLimit:    tc.MemoryLimit,
		}
	}

	// 创建新的判题任务，使用更高的优先级
	rejudgeTask := &scheduler.JudgeTask{
		SubmissionID: originalTask.SubmissionID,
		ProblemID:    originalTask.ProblemID,
		UserID:       originalTask.UserID,
		Language:     originalTask.Language,
		Code:         originalTask.Code,
		TimeLimit:    problemInfo.TimeLimit,   // 使用最新的限制
		MemoryLimit:  problemInfo.MemoryLimit, // 使用最新的限制
		TestCases:    testCases,               // 使用最新的测试用例
		Priority:     scheduler.PriorityHigh,  // 重新判题使用高优先级
		Status:       scheduler.TaskStatusPending,
		CreatedAt:    time.Now(),
		RetryCount:   0, // 重置重试次数
	}

	// 生成新的任务ID
	rejudgeTask.ID = fmt.Sprintf("rejudge_%d_%d", originalTask.SubmissionID, time.Now().Unix())

	l.Logger.Infof("创建重新判题任务: TaskID=%s, SubmissionID=%d, ProblemID=%d, Language=%s, TestCases=%d",
		rejudgeTask.ID, rejudgeTask.SubmissionID, rejudgeTask.ProblemID, rejudgeTask.Language, len(rejudgeTask.TestCases))

	return rejudgeTask, nil
}

// calculateEstimatedTime 计算预估等待时间
func (l *RejudgeLogic) calculateEstimatedTime(queuePosition int) int {
	if queuePosition <= 0 {
		return 0
	}

	// 获取系统配置
	avgTaskTime := l.svcCtx.Config.TaskQueue.AverageTaskTime
	if avgTaskTime <= 0 {
		avgTaskTime = 30 // 默认30秒
	}

	workerCount := l.svcCtx.Config.TaskQueue.MaxWorkers
	if workerCount <= 0 {
		workerCount = 1
	}

	// 重新判题通常优先级较高，减少等待时间
	baseWaitTime := ((queuePosition - 1) / workerCount) * avgTaskTime

	// 重新判题任务减少20%的预估时间（因为优先级高）
	estimatedTime := int(float64(baseWaitTime) * 0.8)

	// 最小等待时间
	if estimatedTime < avgTaskTime/2 {
		estimatedTime = avgTaskTime / 2
	}

	return estimatedTime
}

package service

import (
	"context"
	"fmt"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/scheduler"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// TaskService 任务服务，提供任务相关的通用操作
type TaskService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger logx.Logger
}

// NewTaskService 创建任务服务实例
func NewTaskService(ctx context.Context, svcCtx *svc.ServiceContext) *TaskService {
	return &TaskService{
		ctx:    ctx,
		svcCtx: svcCtx,
		logger: logx.WithContext(ctx),
	}
}

// FindTaskBySubmissionID 根据提交ID查找任务（真实业务逻辑）
func (s *TaskService) FindTaskBySubmissionID(submissionID int64) (*scheduler.JudgeTask, error) {
	if submissionID <= 0 {
		return nil, fmt.Errorf("无效的提交ID: %d", submissionID)
	}

	s.logger.Infof("查找提交ID %d 对应的判题任务", submissionID)

	// 1. 从调度器中查找任务
	task, err := s.svcCtx.TaskScheduler.FindTaskBySubmissionID(submissionID)
	if err != nil {
		s.logger.Errorf("在调度器中未找到提交ID %d 的任务: %v", submissionID, err)
		
		// 2. 如果调度器中没找到，尝试从缓存或数据库查找
		task, err = s.findTaskFromCache(submissionID)
		if err != nil {
			s.logger.Errorf("在缓存中也未找到提交ID %d 的任务: %v", submissionID, err)
			
			// 3. 最后尝试从持久化存储查找
			task, err = s.findTaskFromStorage(submissionID)
			if err != nil {
				return nil, fmt.Errorf("未找到提交ID %d 对应的判题任务", submissionID)
			}
		}
	}

	s.logger.Infof("成功找到提交ID %d 的任务: TaskID=%s, Status=%s", submissionID, task.ID, task.Status)
	return task, nil
}

// findTaskFromCache 从缓存中查找任务
func (s *TaskService) findTaskFromCache(submissionID int64) (*scheduler.JudgeTask, error) {
	// TODO: 实现Redis缓存查找
	// 缓存键格式: judge:task:submission:{submissionID}
	cacheKey := fmt.Sprintf("judge:task:submission:%d", submissionID)
	
	// 这里应该从Redis获取任务信息
	// taskData, err := s.svcCtx.RedisClient.Get(cacheKey)
	// if err != nil {
	//     return nil, fmt.Errorf("缓存中未找到任务: %v", err)
	// }
	
	// var task scheduler.JudgeTask
	// if err := json.Unmarshal([]byte(taskData), &task); err != nil {
	//     return nil, fmt.Errorf("解析缓存任务数据失败: %v", err)
	// }
	
	s.logger.Infof("尝试从缓存查找任务: %s", cacheKey)
	return nil, fmt.Errorf("缓存功能暂未实现")
}

// findTaskFromStorage 从持久化存储查找任务
func (s *TaskService) findTaskFromStorage(submissionID int64) (*scheduler.JudgeTask, error) {
	// TODO: 实现数据库查找
	// 这里应该查询数据库中的judge_tasks表
	
	s.logger.Infof("尝试从数据库查找提交ID %d 的任务", submissionID)
	
	// 示例SQL: SELECT * FROM judge_tasks WHERE submission_id = ?
	// 然后将数据库记录转换为scheduler.JudgeTask结构
	
	return nil, fmt.Errorf("数据库查找功能暂未实现")
}

// ValidateTaskAccess 验证任务访问权限
func (s *TaskService) ValidateTaskAccess(task *scheduler.JudgeTask, userID int64, userRole string) error {
	if task == nil {
		return fmt.Errorf("任务不存在")
	}

	// 1. 管理员可以访问所有任务
	if userRole == "admin" {
		return nil
	}

	// 2. 用户只能访问自己的任务
	if task.UserID == userID {
		return nil
	}

	// 3. 教师可以访问公开的任务（根据业务需求）
	if userRole == "teacher" {
		// TODO: 根据题目或比赛的公开性决定是否允许访问
		return nil
	}

	return fmt.Errorf("权限不足：无法访问该判题任务")
}

// GetTaskStatusText 获取任务状态的中文描述
func (s *TaskService) GetTaskStatusText(status string) string {
	switch status {
	case scheduler.TaskStatusPending:
		return "等待中"
	case scheduler.TaskStatusRunning:
		return "执行中"
	case scheduler.TaskStatusCompleted:
		return "已完成"
	case scheduler.TaskStatusFailed:
		return "失败"
	case scheduler.TaskStatusCancelled:
		return "已取消"
	default:
		return status
	}
}

// CalculateTaskProgress 计算任务执行进度
func (s *TaskService) CalculateTaskProgress(task *scheduler.JudgeTask) int {
	if task == nil {
		return 0
	}

	switch task.Status {
	case scheduler.TaskStatusPending:
		return 0
	case scheduler.TaskStatusRunning:
		if task.Result != nil && len(task.TestCases) > 0 {
			completed := len(task.Result.TestCases)
			total := len(task.TestCases)
			if total > 0 {
				progress := (completed * 100) / total
				// 确保运行中的任务至少显示10%的进度
				if progress < 10 {
					return 10
				}
				return progress
			}
		}
		return 10 // 至少10%表示已开始
	case scheduler.TaskStatusCompleted:
		return 100
	case scheduler.TaskStatusFailed, scheduler.TaskStatusCancelled:
		return 100
	default:
		return 0
	}
}

// GenerateTaskStatusMessage 生成任务状态消息
func (s *TaskService) GenerateTaskStatusMessage(task *scheduler.JudgeTask) string {
	if task == nil {
		return "任务不存在"
	}

	currentTestCase := 0
	totalTestCases := len(task.TestCases)

	if task.Result != nil {
		currentTestCase = len(task.Result.TestCases)
	}

	switch task.Status {
	case scheduler.TaskStatusPending:
		return "等待判题中..."
	case scheduler.TaskStatusRunning:
		if currentTestCase > 0 && totalTestCases > 0 {
			return fmt.Sprintf("正在执行测试用例 %d/%d", currentTestCase, totalTestCases)
		}
		return "正在编译代码..."
	case scheduler.TaskStatusCompleted:
		return "判题完成"
	case scheduler.TaskStatusFailed:
		if task.Error != "" {
			return fmt.Sprintf("判题失败: %s", task.Error)
		}
		return "判题失败"
	case scheduler.TaskStatusCancelled:
		return "判题已取消"
	default:
		return "未知状态"
	}
}

// IsTaskCancellable 检查任务是否可以取消
func (s *TaskService) IsTaskCancellable(task *scheduler.JudgeTask) bool {
	if task == nil {
		return false
	}

	// 只有等待中和运行中的任务可以取消
	return task.Status == scheduler.TaskStatusPending || task.Status == scheduler.TaskStatusRunning
}

// IsTaskCompleted 检查任务是否已完成（成功或失败）
func (s *TaskService) IsTaskCompleted(task *scheduler.JudgeTask) bool {
	if task == nil {
		return false
	}

	return task.Status == scheduler.TaskStatusCompleted || 
		   task.Status == scheduler.TaskStatusFailed || 
		   task.Status == scheduler.TaskStatusCancelled
}






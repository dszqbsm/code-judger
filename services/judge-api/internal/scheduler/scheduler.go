package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/config"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/judge"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// 任务优先级
const (
	PriorityLow    = 3 // 普通任务
	PriorityNormal = 2 // VIP用户任务
	PriorityHigh   = 1 // 比赛任务
)

// 任务状态
const (
	TaskStatusPending   = "pending"   // 等待中
	TaskStatusRunning   = "running"   // 执行中
	TaskStatusCompleted = "completed" // 已完成
	TaskStatusFailed    = "failed"    // 失败
	TaskStatusCancelled = "cancelled" // 已取消
)

// 判题任务
type JudgeTask struct {
	ID           string             `json:"id"`
	SubmissionID int64              `json:"submission_id"`
	ProblemID    int64              `json:"problem_id"`
	UserID       int64              `json:"user_id"`
	Language     string             `json:"language"`
	Code         string             `json:"code"`
	TimeLimit    int                `json:"time_limit"`
	MemoryLimit  int                `json:"memory_limit"`
	TestCases    []*types.TestCase  `json:"test_cases"`
	Priority     int                `json:"priority"`
	Status       string             `json:"status"`
	CreatedAt    time.Time          `json:"created_at"`
	StartedAt    *time.Time         `json:"started_at,omitempty"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
	Result       *types.JudgeResult `json:"result,omitempty"`
	Error        string             `json:"error,omitempty"`
	RetryCount   int                `json:"retry_count"`
	Context      context.Context    `json:"-"`
	CancelFunc   context.CancelFunc `json:"-"`
}

// 工作器
type Worker struct {
	ID       int
	TaskChan chan *JudgeTask
	QuitChan chan bool
	Judge    *judge.JudgeEngine
}

func NewWorker(id int, judge *judge.JudgeEngine) *Worker {
	return &Worker{
		ID:       id,
		TaskChan: make(chan *JudgeTask),
		QuitChan: make(chan bool),
		Judge:    judge,
	}
}

func (w *Worker) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	logx.Infof("Worker %d started", w.ID)

	for {
		select {
		case task := <-w.TaskChan:
			w.processTask(task)
		case <-w.QuitChan:
			logx.Infof("Worker %d stopped", w.ID)
			return
		}
	}
}

func (w *Worker) processTask(task *JudgeTask) {
	logx.Infof("Worker %d processing task %s", w.ID, task.ID)

	// 更新任务状态为运行中
	task.Status = TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now

	// 执行判题
	result, err := w.Judge.Judge(task.Context, &judge.JudgeRequest{
		SubmissionID: task.SubmissionID,
		ProblemID:    task.ProblemID,
		UserID:       task.UserID,
		Language:     task.Language,
		Code:         task.Code,
		TimeLimit:    task.TimeLimit,
		MemoryLimit:  task.MemoryLimit,
		TestCases:    task.TestCases,
	})

	// 更新任务结果
	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err.Error()
		logx.Errorf("Worker %d task %s failed: %v", w.ID, task.ID, err)
	} else {
		task.Status = TaskStatusCompleted
		task.Result = result
		logx.Infof("Worker %d task %s completed successfully", w.ID, task.ID)
	}
}

func (w *Worker) Stop() {
	w.QuitChan <- true
}

// 任务调度器
type TaskScheduler struct {
	config        *config.TaskQueueConf
	workers       []*Worker
	taskQueue     chan *JudgeTask
	priorityQueue *PriorityQueue
	tasks         sync.Map // map[string]*JudgeTask
	stats         *SchedulerStats
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	judge         *judge.JudgeEngine
}

// 调度器统计信息
type SchedulerStats struct {
	TotalTasks     int64 `json:"total_tasks"`
	PendingTasks   int64 `json:"pending_tasks"`
	RunningTasks   int64 `json:"running_tasks"`
	CompletedTasks int64 `json:"completed_tasks"`
	FailedTasks    int64 `json:"failed_tasks"`
	CancelledTasks int64 `json:"cancelled_tasks"`
}

func NewTaskScheduler(config *config.TaskQueueConf, judge *judge.JudgeEngine) *TaskScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	scheduler := &TaskScheduler{
		config:        config,
		workers:       make([]*Worker, config.MaxWorkers),
		taskQueue:     make(chan *JudgeTask, config.QueueSize),
		priorityQueue: NewPriorityQueue(),
		stats:         &SchedulerStats{},
		ctx:           ctx,
		cancel:        cancel,
		judge:         judge,
	}

	// 创建工作器
	for i := 0; i < config.MaxWorkers; i++ {
		scheduler.workers[i] = NewWorker(i, judge)
	}

	return scheduler
}

func (s *TaskScheduler) Start() error {
	logx.Info("Starting task scheduler...")

	// 启动工作器
	for _, worker := range s.workers {
		s.wg.Add(1)
		go worker.Start(&s.wg)
	}

	// 启动任务分发器
	go s.dispatch()

	// 启动任务分配器（从taskQueue分发给workers）
	go s.distributeToWorkers()

	// 启动统计更新器
	go s.updateStats()

	logx.Infof("Task scheduler started with %d workers", len(s.workers))
	return nil
}

func (s *TaskScheduler) Stop() error {
	logx.Info("Stopping task scheduler...")

	// 取消上下文
	s.cancel()

	// 停止工作器
	for _, worker := range s.workers {
		worker.Stop()
	}

	// 等待所有工作器停止
	s.wg.Wait()

	logx.Info("Task scheduler stopped")
	return nil
}

// 提交任务
func (s *TaskScheduler) SubmitTask(task *JudgeTask) error {
	// 生成任务ID
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d_%d", task.SubmissionID, time.Now().UnixNano())
	}

	// 设置任务上下文
	task.Context, task.CancelFunc = context.WithTimeout(s.ctx, time.Duration(s.config.TaskTimeout)*time.Second)
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()

	// 存储任务
	s.tasks.Store(task.ID, task)

	// 加入优先级队列
	s.priorityQueue.Push(task)

	// 更新统计
	atomic.AddInt64(&s.stats.TotalTasks, 1)
	atomic.AddInt64(&s.stats.PendingTasks, 1)

	logx.Infof("Task submitted: %s (priority: %d)", task.ID, task.Priority)
	return nil
}

// 取消任务
func (s *TaskScheduler) CancelTask(taskID string) error {
	taskInterface, exists := s.tasks.Load(taskID)
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task := taskInterface.(*JudgeTask)

	// 取消任务上下文
	if task.CancelFunc != nil {
		task.CancelFunc()
	}

	// 更新任务状态
	task.Status = TaskStatusCancelled
	completedAt := time.Now()
	task.CompletedAt = &completedAt

	// 更新统计
	if task.StartedAt == nil {
		atomic.AddInt64(&s.stats.PendingTasks, -1)
	} else {
		atomic.AddInt64(&s.stats.RunningTasks, -1)
	}
	atomic.AddInt64(&s.stats.CancelledTasks, 1)

	logx.Infof("Task cancelled: %s", taskID)
	return nil
}

// 获取任务状态
func (s *TaskScheduler) GetTaskStatus(taskID string) (*JudgeTask, error) {
	taskInterface, exists := s.tasks.Load(taskID)
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	task := taskInterface.(*JudgeTask)
	return task, nil
}

// 获取队列状态
func (s *TaskScheduler) GetQueueStatus() *QueueStatus {
	queueItems := make([]*QueueItem, 0)

	// 获取优先级队列中的任务
	s.priorityQueue.mutex.Lock()
	for _, task := range s.priorityQueue.tasks {
		if task.Status == TaskStatusPending {
			queueItems = append(queueItems, &QueueItem{
				SubmissionID:  task.SubmissionID,
				UserID:        task.UserID,
				ProblemID:     task.ProblemID,
				Language:      task.Language,
				Priority:      task.Priority,
				QueueTime:     task.CreatedAt.Format(time.RFC3339),
				EstimatedTime: s.estimateWaitTime(len(queueItems)),
			})
		}
	}
	s.priorityQueue.mutex.Unlock()

	return &QueueStatus{
		QueueLength:    len(queueItems),
		PendingTasks:   int(atomic.LoadInt64(&s.stats.PendingTasks)),
		RunningTasks:   int(atomic.LoadInt64(&s.stats.RunningTasks)),
		CompletedTasks: int(atomic.LoadInt64(&s.stats.CompletedTasks)),
		FailedTasks:    int(atomic.LoadInt64(&s.stats.FailedTasks)),
		QueueItems:     queueItems,
	}
}

// 获取任务在队列中的位置
func (s *TaskScheduler) GetTaskPosition(taskID string) (int, error) {
	s.priorityQueue.mutex.Lock()
	defer s.priorityQueue.mutex.Unlock()

	// 在优先级队列中查找任务位置
	for i, task := range s.priorityQueue.tasks {
		if task.ID == taskID && task.Status == TaskStatusPending {
			return i + 1, nil // 返回1基的位置
		}
	}

	return 0, fmt.Errorf("task not found in queue: %s", taskID)
}

// 根据提交ID查找任务
func (s *TaskScheduler) FindTaskBySubmissionID(submissionID int64) (*JudgeTask, error) {
	// 1. 先在优先级队列中查找（等待中的任务）
	s.priorityQueue.mutex.Lock()
	for _, task := range s.priorityQueue.tasks {
		if task.SubmissionID == submissionID {
			s.priorityQueue.mutex.Unlock()
			return task, nil
		}
	}
	s.priorityQueue.mutex.Unlock()

	// 2. 在所有任务映射中查找（包括运行中和已完成的任务）
	var foundTask *JudgeTask
	s.tasks.Range(func(key, value interface{}) bool {
		task := value.(*JudgeTask)
		if task.SubmissionID == submissionID {
			foundTask = task
			return false // 停止遍历
		}
		return true // 继续遍历
	})

	if foundTask != nil {
		return foundTask, nil
	}

	return nil, fmt.Errorf("task not found for submission_id: %d", submissionID)
}

// 任务分发器（从优先级队列到taskQueue）
func (s *TaskScheduler) dispatch() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// 从优先级队列获取任务
			task := s.priorityQueue.Pop()
			if task == nil {
				continue
			}

			// 检查任务是否已取消
			if task.Status == TaskStatusCancelled {
				continue
			}

			// 尝试分配到任务队列
			select {
			case s.taskQueue <- task:
				// 成功分配任务到队列
				logx.Infof("Task %s dispatched to queue", task.ID)
			default:
				// 队列已满，重新加入优先级队列
				s.priorityQueue.Push(task)
			}
		}
	}
}

// 任务分配器（从taskQueue分发给workers）
func (s *TaskScheduler) distributeToWorkers() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case task := <-s.taskQueue:
			// 检查任务是否已取消
			if task.Status == TaskStatusCancelled {
				continue
			}

			// 寻找空闲的worker
			distributed := false
			for _, worker := range s.workers {
				select {
				case worker.TaskChan <- task:
					// 成功分配给worker
					logx.Infof("Task %s distributed to worker %d", task.ID, worker.ID)
					atomic.AddInt64(&s.stats.PendingTasks, -1)
					atomic.AddInt64(&s.stats.RunningTasks, 1)
					distributed = true
					goto nextTask
				default:
					// 这个worker忙，尝试下一个
					continue
				}
			}

			// 如果没有空闲worker，重新放回队列
			if !distributed {
				select {
				case s.taskQueue <- task:
					// 重新放回队列
				default:
					// 队列已满，放回优先级队列
					s.priorityQueue.Push(task)
				}
			}

		nextTask:
		}
	}
}

// 更新统计信息
func (s *TaskScheduler) updateStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// 清理已完成的旧任务
			s.cleanupOldTasks()
		}
	}
}

// 清理旧任务
func (s *TaskScheduler) cleanupOldTasks() {
	cutoff := time.Now().Add(-24 * time.Hour) // 保留24小时内的任务

	s.tasks.Range(func(key, value interface{}) bool {
		task := value.(*JudgeTask)
		if task.CompletedAt != nil && task.CompletedAt.Before(cutoff) {
			s.tasks.Delete(key)
		}
		return true
	})
}

// 估算等待时间
func (s *TaskScheduler) estimateWaitTime(position int) int {
	avgTaskTime := 30 // 平均任务执行时间（秒）
	workersCount := len(s.workers)

	if workersCount == 0 {
		return position * avgTaskTime
	}

	return (position / workersCount) * avgTaskTime
}

// 获取调度器统计信息
func (s *TaskScheduler) GetStats() *SchedulerStats {
	return &SchedulerStats{
		TotalTasks:     atomic.LoadInt64(&s.stats.TotalTasks),
		PendingTasks:   atomic.LoadInt64(&s.stats.PendingTasks),
		RunningTasks:   atomic.LoadInt64(&s.stats.RunningTasks),
		CompletedTasks: atomic.LoadInt64(&s.stats.CompletedTasks),
		FailedTasks:    atomic.LoadInt64(&s.stats.FailedTasks),
		CancelledTasks: atomic.LoadInt64(&s.stats.CancelledTasks),
	}
}

// 队列状态
type QueueStatus struct {
	QueueLength    int          `json:"queue_length"`
	PendingTasks   int          `json:"pending_tasks"`
	RunningTasks   int          `json:"running_tasks"`
	CompletedTasks int          `json:"completed_tasks"`
	FailedTasks    int          `json:"failed_tasks"`
	QueueItems     []*QueueItem `json:"queue_items"`
}

// 队列项
type QueueItem struct {
	SubmissionID  int64  `json:"submission_id"`
	UserID        int64  `json:"user_id"`
	ProblemID     int64  `json:"problem_id"`
	Language      string `json:"language"`
	Priority      int    `json:"priority"`
	QueueTime     string `json:"queue_time"`
	EstimatedTime int    `json:"estimated_time"`
}

// 优先级队列
type PriorityQueue struct {
	tasks []*JudgeTask
	mutex sync.Mutex
}

func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		tasks: make([]*JudgeTask, 0),
	}
}

func (pq *PriorityQueue) Push(task *JudgeTask) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	// 按优先级插入任务
	inserted := false
	for i, t := range pq.tasks {
		if task.Priority < t.Priority ||
			(task.Priority == t.Priority && task.CreatedAt.Before(t.CreatedAt)) {
			// 插入到合适位置
			pq.tasks = append(pq.tasks[:i], append([]*JudgeTask{task}, pq.tasks[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		pq.tasks = append(pq.tasks, task)
	}
}

func (pq *PriorityQueue) Pop() *JudgeTask {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	if len(pq.tasks) == 0 {
		return nil
	}

	task := pq.tasks[0]
	pq.tasks = pq.tasks[1:]
	return task
}

func (pq *PriorityQueue) Len() int {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	return len(pq.tasks)
}

// 任务重试机制
func (s *TaskScheduler) retryFailedTask(task *JudgeTask) {
	if task.RetryCount >= s.config.RetryTimes {
		logx.Errorf("Task %s exceeded max retry count", task.ID)
		return
	}

	// 增加重试次数
	task.RetryCount++
	task.Status = TaskStatusPending
	task.StartedAt = nil
	task.CompletedAt = nil
	task.Error = ""

	// 延迟重试
	time.AfterFunc(time.Duration(s.config.RetryInterval)*time.Second, func() {
		s.priorityQueue.Push(task)
		logx.Infof("Retrying task %s (attempt %d)", task.ID, task.RetryCount)
	})
}

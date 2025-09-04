package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// QueueManager 队列管理器
type QueueManager struct {
	redisClient *redis.Redis
	logger      logx.Logger
}

// JudgeTaskInfo 判题任务信息
type JudgeTaskInfo struct {
	SubmissionID int64     `json:"submission_id"`
	UserID       int64     `json:"user_id"`
	ProblemID    int64     `json:"problem_id"`
	Priority     int       `json:"priority"`     // 优先级：1=最高，3=最低
	CreatedAt    time.Time `json:"created_at"`
	EstimatedTime int      `json:"estimated_time"` // 预估处理时间（秒）
}

// QueueStats 队列统计信息
type QueueStats struct {
	TotalTasks       int64   `json:"total_tasks"`        // 总任务数
	PendingTasks     int64   `json:"pending_tasks"`      // 等待中任务数
	ProcessingTasks  int64   `json:"processing_tasks"`   // 处理中任务数
	CompletedTasks   int64   `json:"completed_tasks"`    // 已完成任务数
	AverageWaitTime  float64 `json:"average_wait_time"`  // 平均等待时间
	AverageJudgeTime float64 `json:"average_judge_time"` // 平均判题时间
	ActiveJudges     int     `json:"active_judges"`      // 活跃判题服务器数量
}

// NewQueueManager 创建队列管理器
func NewQueueManager(redisClient *redis.Redis) *QueueManager {
	return &QueueManager{
		redisClient: redisClient,
		logger:      logx.WithContext(context.Background()),
	}
}

// AddTask 添加任务到队列
func (qm *QueueManager) AddTask(ctx context.Context, task *JudgeTaskInfo) (int, error) {
	// 1. 序列化任务信息
	taskData, err := json.Marshal(task)
	if err != nil {
		return 0, fmt.Errorf("序列化任务失败: %v", err)
	}

	// 2. 根据优先级添加到不同的队列
	queueKey := qm.getQueueKey(task.Priority)
	
	// 3. 使用LPUSH添加到队列头部（高优先级）或RPUSH添加到队列尾部（低优先级）
	var position int64
	if task.Priority == 1 { // 最高优先级，添加到队列头部
		pos, e := qm.redisClient.Lpush(queueKey, string(taskData))
		position = int64(pos)
		err = e
	} else { // 其他优先级，添加到队列尾部
		pos, e := qm.redisClient.Rpush(queueKey, string(taskData))
		position = int64(pos)
		err = e
	}
	
	if err != nil {
		return 0, fmt.Errorf("添加任务到队列失败: %v", err)
	}

	// 4. 更新队列统计信息
	if err := qm.updateQueueStats(ctx, "add", task.Priority); err != nil {
		qm.logger.Errorf("更新队列统计失败: %v", err)
	}

	// 5. 设置任务过期时间（24小时）
	qm.redisClient.Expire(queueKey, 86400)

	// 6. 记录任务添加日志
	qm.logger.Infof("任务添加到队列: SubmissionID=%d, Priority=%d, Position=%d", 
		task.SubmissionID, task.Priority, position)

	return int(position), nil
}

// GetQueuePosition 获取任务在队列中的位置
func (qm *QueueManager) GetQueuePosition(ctx context.Context, submissionID int64) (int, error) {
	position := 1

	// 1. 按优先级顺序检查所有队列
	for priority := 1; priority <= 3; priority++ {
		queueKey := qm.getQueueKey(priority)
		
		// 2. 获取队列长度
		length, err := qm.redisClient.Llen(queueKey)
		if err != nil {
			qm.logger.Errorf("获取队列长度失败: %v", err)
			continue
		}

		// 3. 在当前优先级队列中查找任务
		found, taskPosition, err := qm.findTaskInQueue(queueKey, submissionID, int(length))
		if err != nil {
			qm.logger.Errorf("在队列中查找任务失败: %v", err)
			continue
		}

		if found {
			return position + taskPosition, nil
		}

		// 4. 如果在当前队列中没找到，累加该队列的长度
		position += length
	}

	// 5. 如果在所有队列中都没找到，可能任务已经在处理中
	processingKey := "judge:processing"
	exists, err := qm.redisClient.Hexists(processingKey, strconv.FormatInt(submissionID, 10))
	if err == nil && exists {
		return 0, nil // 返回0表示正在处理中
	}

	return -1, fmt.Errorf("任务不在队列中: SubmissionID=%d", submissionID)
}

// GetEstimatedWaitTime 获取预估等待时间
func (qm *QueueManager) GetEstimatedWaitTime(ctx context.Context, queuePosition int) (int, error) {
	if queuePosition <= 0 {
		return 0, nil // 正在处理中或已完成
	}

	// 1. 获取队列统计信息
	stats, err := qm.GetQueueStats(ctx)
	if err != nil {
		qm.logger.Errorf("获取队列统计失败: %v", err)
		// 使用默认值
		stats = &QueueStats{
			AverageJudgeTime: 6.0,
			ActiveJudges:     1,
		}
	}

	// 2. 计算预估时间
	avgJudgeTime := stats.AverageJudgeTime
	if avgJudgeTime <= 0 {
		avgJudgeTime = 6.0 // 默认6秒
	}

	activeJudges := stats.ActiveJudges
	if activeJudges <= 0 {
		activeJudges = 1
	}

	// 3. 预估时间 = (队列位置 / 并发判题数) * 平均判题时间
	estimatedTime := int((float64(queuePosition) / float64(activeJudges)) * avgJudgeTime)

	// 4. 至少需要平均判题时间
	if estimatedTime < int(avgJudgeTime) {
		estimatedTime = int(avgJudgeTime)
	}

	// 5. 考虑队列拥堵情况，增加缓冲时间
	if queuePosition > 10 {
		bufferTime := int(float64(estimatedTime) * 0.1) // 增加10%的缓冲时间
		estimatedTime += bufferTime
	}

	return estimatedTime, nil
}

// GetQueueStats 获取队列统计信息
func (qm *QueueManager) GetQueueStats(ctx context.Context) (*QueueStats, error) {
	stats := &QueueStats{}

	// 1. 获取各优先级队列长度
	var totalPending int
	for priority := 1; priority <= 3; priority++ {
		queueKey := qm.getQueueKey(priority)
		length, err := qm.redisClient.Llen(queueKey)
		if err != nil {
			qm.logger.Errorf("获取队列长度失败: %v", err)
			continue
		}
		totalPending += length
	}
	stats.PendingTasks = int64(totalPending)

	// 2. 获取处理中任务数
	processingKey := "judge:processing"
	processingCount, err := qm.redisClient.Hlen(processingKey)
	if err != nil {
		qm.logger.Errorf("获取处理中任务数失败: %v", err)
	} else {
		stats.ProcessingTasks = int64(processingCount)
	}

	// 3. 获取统计数据
	statsKey := "judge:stats"
	
	// 获取平均等待时间
	avgWaitTimeStr, err := qm.redisClient.Hget(statsKey, "avg_wait_time")
	if err == nil && avgWaitTimeStr != "" {
		if avgWaitTime, err := strconv.ParseFloat(avgWaitTimeStr, 64); err == nil {
			stats.AverageWaitTime = avgWaitTime
		}
	}

	// 获取平均判题时间
	avgJudgeTimeStr, err := qm.redisClient.Hget(statsKey, "avg_judge_time")
	if err == nil && avgJudgeTimeStr != "" {
		if avgJudgeTime, err := strconv.ParseFloat(avgJudgeTimeStr, 64); err == nil {
			stats.AverageJudgeTime = avgJudgeTime
		}
	} else {
		stats.AverageJudgeTime = 6.0 // 默认值
	}

	// 获取已完成任务数
	completedStr, err := qm.redisClient.Hget(statsKey, "completed_tasks")
	if err == nil && completedStr != "" {
		if completed, err := strconv.ParseInt(completedStr, 10, 64); err == nil {
			stats.CompletedTasks = completed
		}
	}

	// 4. 获取活跃判题服务器数量
	judgeServersKey := "judge:servers:active"
	activeJudges, err := qm.redisClient.Scard(judgeServersKey)
	if err != nil {
		qm.logger.Errorf("获取活跃判题服务器数量失败: %v", err)
		stats.ActiveJudges = 1 // 默认值
	} else {
		stats.ActiveJudges = int(activeJudges)
		if stats.ActiveJudges == 0 {
			stats.ActiveJudges = 1
		}
	}

	stats.TotalTasks = stats.PendingTasks + stats.ProcessingTasks + stats.CompletedTasks

	return stats, nil
}

// UpdateJudgeServerHeartbeat 更新判题服务器心跳
func (qm *QueueManager) UpdateJudgeServerHeartbeat(ctx context.Context, serverID string) error {
	judgeServersKey := "judge:servers:active"
	
	// 1. 添加服务器到活跃集合
	if _, err := qm.redisClient.Sadd(judgeServersKey, serverID); err != nil {
		return fmt.Errorf("更新服务器心跳失败: %v", err)
	}

	// 2. 设置服务器心跳时间戳
	heartbeatKey := fmt.Sprintf("judge:server:%s:heartbeat", serverID)
	timestamp := time.Now().Unix()
	if err := qm.redisClient.Setex(heartbeatKey, strconv.FormatInt(timestamp, 10), 60); err != nil {
		return fmt.Errorf("设置心跳时间戳失败: %v", err)
	}

	return nil
}

// CleanupInactiveServers 清理非活跃的判题服务器
func (qm *QueueManager) CleanupInactiveServers(ctx context.Context) error {
	judgeServersKey := "judge:servers:active"
	
	// 1. 获取所有服务器
	servers, err := qm.redisClient.Smembers(judgeServersKey)
	if err != nil {
		return fmt.Errorf("获取服务器列表失败: %v", err)
	}

	currentTime := time.Now().Unix()
	
	// 2. 检查每个服务器的心跳时间
	for _, server := range servers {
		heartbeatKey := fmt.Sprintf("judge:server:%s:heartbeat", server)
		timestampStr, err := qm.redisClient.Get(heartbeatKey)
		
		if err != nil || timestampStr == "" {
			// 心跳不存在，移除服务器
			qm.redisClient.Srem(judgeServersKey, server)
			qm.logger.Infof("移除非活跃判题服务器: %s", server)
			continue
		}

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			continue
		}

		// 3. 如果超过90秒没有心跳，认为服务器离线
		if currentTime-timestamp > 90 {
			qm.redisClient.Srem(judgeServersKey, server)
			qm.logger.Infof("移除超时判题服务器: %s (最后心跳: %d秒前)", server, currentTime-timestamp)
		}
	}

	return nil
}

// getQueueKey 根据优先级获取队列键名
func (qm *QueueManager) getQueueKey(priority int) string {
	return fmt.Sprintf("judge:queue:priority:%d", priority)
}

// findTaskInQueue 在队列中查找指定任务
func (qm *QueueManager) findTaskInQueue(queueKey string, submissionID int64, length int) (bool, int, error) {
	// 分批查询，避免一次性查询过多数据
	batchSize := 100
	
	for start := 0; start < length; start += batchSize {
		end := start + batchSize - 1
		if end >= length {
			end = length - 1
		}

		// 获取队列中的任务数据
		tasks, err := qm.redisClient.Lrange(queueKey, start, end)
		if err != nil {
			return false, 0, err
		}

		// 在当前批次中查找任务
		for i, taskData := range tasks {
			var task JudgeTaskInfo
			if err := json.Unmarshal([]byte(taskData), &task); err != nil {
				continue
			}

			if task.SubmissionID == submissionID {
				return true, start + i, nil
			}
		}
	}

	return false, 0, nil
}

// updateQueueStats 更新队列统计信息
func (qm *QueueManager) updateQueueStats(ctx context.Context, operation string, priority int) error {
	statsKey := "judge:stats"
	
	switch operation {
	case "add":
		// 增加待处理任务计数
		qm.redisClient.Hincrby(statsKey, "pending_tasks", 1)
	case "start":
		// 任务开始处理
		qm.redisClient.Hincrby(statsKey, "pending_tasks", -1)
		qm.redisClient.Hincrby(statsKey, "processing_tasks", 1)
	case "complete":
		// 任务完成
		qm.redisClient.Hincrby(statsKey, "processing_tasks", -1)
		qm.redisClient.Hincrby(statsKey, "completed_tasks", 1)
	}

	return nil
}

// GetCurrentQueueLength 获取当前总队列长度
func (qm *QueueManager) GetCurrentQueueLength(ctx context.Context) (int, error) {
	totalLength := 0
	
	// 累加所有优先级队列的长度
	for priority := 1; priority <= 3; priority++ {
		queueKey := qm.getQueueKey(priority)
		length, err := qm.redisClient.Llen(queueKey)
		if err != nil {
			qm.logger.Errorf("获取队列长度失败: %v", err)
			continue
		}
		totalLength += int(length)
	}

	return totalLength, nil
}

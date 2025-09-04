package messagequeue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/client"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/config"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/scheduler"
	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// 判题任务消息结构
type JudgeTaskMessage struct {
	SubmissionID int64             `json:"submission_id"`
	ProblemID    int64             `json:"problem_id"`
	UserID       int64             `json:"user_id"`
	Language     string            `json:"language"`
	Code         string            `json:"code"`
	TimeLimit    int               `json:"time_limit"`   // 毫秒
	MemoryLimit  int               `json:"memory_limit"` // MB
	TestCases    []*types.TestCase `json:"test_cases"`
	Priority     int               `json:"priority"`
	CreatedAt    time.Time         `json:"created_at"`
}

// Kafka消费者接口
type Consumer interface {
	Start(ctx context.Context) error
	Stop() error
}

// Kafka消费者实现
type KafkaConsumer struct {
	reader        *kafka.Reader
	taskScheduler *scheduler.TaskScheduler
	producer      Producer                    // 用于发送判题结果回提交服务
	problemClient client.ProblemServiceClient // 题目服务客户端
	config        config.KafkaConf
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewKafkaConsumer(config config.KafkaConf, taskScheduler *scheduler.TaskScheduler, producer Producer, problemClient client.ProblemServiceClient) Consumer {
	// 创建Kafka Reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		Topic:          config.Topic,
		GroupID:        config.Group,
		MinBytes:       1,   // 1B - 更小的最小字节数
		MaxBytes:       1e6, // 1MB - 更小的最大字节数
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset, // 从最早的消息开始读取
	})

	ctx, cancel := context.WithCancel(context.Background())

	return &KafkaConsumer{
		reader:        reader,
		taskScheduler: taskScheduler,
		producer:      producer,
		problemClient: problemClient,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// 启动消费者
func (c *KafkaConsumer) Start(ctx context.Context) error {
	logx.Info("Starting Kafka consumer...")
	logx.Infof("Kafka配置 - Brokers: %v, Topic: %s, Group: %s", c.config.Brokers, c.config.Topic, c.config.Group)

	go c.consume()

	logx.Infof("Kafka consumer started, listening on topic: %s", c.config.Topic)
	return nil
}

// 停止消费者
func (c *KafkaConsumer) Stop() error {
	logx.Info("Stopping Kafka consumer...")

	c.cancel()

	if c.reader != nil {
		if err := c.reader.Close(); err != nil {
			logx.Errorf("Failed to close Kafka reader: %v", err)
			return err
		}
	}

	logx.Info("Kafka consumer stopped")
	return nil
}

// 消费消息
func (c *KafkaConsumer) consume() {
	retryCount := 0
	maxRetries := 10 // 最大重试次数

	logx.Info("Kafka consumer: 开始消费循环")
	logx.Infof("Kafka consumer: Reader配置 - Topic: %s, GroupID: %s", c.reader.Config().Topic, c.reader.Config().GroupID)

	for {
		select {
		case <-c.ctx.Done():
			logx.Info("Kafka consumer context cancelled, stopping consumer")
			return
		default:
			logx.Info("Kafka consumer: 尝试读取消息...")
			// 读取消息
			message, err := c.reader.ReadMessage(c.ctx)
			if err != nil {
				if err == context.Canceled {
					logx.Info("Kafka consumer context cancelled")
					return
				}

				retryCount++
				logx.Errorf("Failed to read message from Kafka (attempt %d/%d): %v", retryCount, maxRetries, err)

				// 如果重试次数过多，停止消费者
				if retryCount >= maxRetries {
					logx.Errorf("Max retries reached, stopping Kafka consumer")
					return
				}

				// 指数退避重试
				backoff := time.Duration(retryCount) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
				time.Sleep(backoff)
				continue
			}

			// 重置重试计数
			retryCount = 0

			logx.Infof("Kafka consumer: 成功读取消息 - Topic: %s, Partition: %d, Offset: %d, Key: %s",
				message.Topic, message.Partition, message.Offset, string(message.Key))
			logx.Infof("Kafka consumer: 消息内容长度: %d bytes", len(message.Value))
			// 安全地显示消息预览
			preview := string(message.Value)
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			logx.Infof("Kafka consumer: 消息内容预览: %s", preview)

			// 处理消息
			logx.Info("Kafka consumer: 开始处理消息...")
			if err := c.processMessage(&message); err != nil {
				logx.Errorf("Kafka consumer: 消息处理失败: %v", err)
				// 可以考虑将失败的消息发送到死信队列
				c.handleMessageError(&message, err)
			} else {
				logx.Info("Kafka consumer: 消息处理成功")
			}
		}
	}
}

// 处理消息
func (c *KafkaConsumer) processMessage(message *kafka.Message) error {
	logx.Infof("Processing message from topic: %s, partition: %d, offset: %d",
		message.Topic, message.Partition, message.Offset)

	// 解析消息
	var taskMessage JudgeTaskMessage
	if err := json.Unmarshal(message.Value, &taskMessage); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	logx.Infof("Received judge task: SubmissionID=%d, ProblemID=%d, Language=%s",
		taskMessage.SubmissionID, taskMessage.ProblemID, taskMessage.Language)

	// 验证消息内容
	if err := c.validateTaskMessage(&taskMessage); err != nil {
		return fmt.Errorf("invalid task message: %w", err)
	}

	// 通过RPC调用题目服务获取题目详细信息和测试用例
	logx.Infof("Fetching problem details via RPC: ProblemID=%d", taskMessage.ProblemID)
	problemDetails, err := c.fetchProblemDetails(taskMessage.ProblemID)
	if err != nil {
		return fmt.Errorf("failed to fetch problem details: %w", err)
	}

	logx.Infof("Successfully fetched problem details: ProblemID=%d, TimeLimit=%dms, MemoryLimit=%dMB, TestCases=%d",
		taskMessage.ProblemID, problemDetails.TimeLimit, problemDetails.MemoryLimit, len(problemDetails.TestCases))

	// 创建完整的调度器任务
	task := &scheduler.JudgeTask{
		SubmissionID: taskMessage.SubmissionID,
		ProblemID:    taskMessage.ProblemID,
		UserID:       taskMessage.UserID,
		Language:     taskMessage.Language,
		Code:         taskMessage.Code,
		TimeLimit:    problemDetails.TimeLimit,
		MemoryLimit:  problemDetails.MemoryLimit,
		TestCases:    problemDetails.TestCases,
		Priority:     taskMessage.Priority,
	}

	// 提交任务到调度器
	if err := c.taskScheduler.SubmitTask(task); err != nil {
		return fmt.Errorf("failed to submit task to scheduler: %w", err)
	}

	// 启动任务状态监控
	go c.monitorTaskStatus(task)

	logx.Infof("Successfully submitted task to scheduler: SubmissionID=%d, TaskID=%s",
		taskMessage.SubmissionID, task.ID)

	return nil
}

// 验证任务消息（支持简化任务消息）
func (c *KafkaConsumer) validateTaskMessage(msg *JudgeTaskMessage) error {
	if msg.SubmissionID <= 0 {
		return fmt.Errorf("invalid submission_id: %d", msg.SubmissionID)
	}

	if msg.ProblemID <= 0 {
		return fmt.Errorf("invalid problem_id: %d", msg.ProblemID)
	}

	if msg.UserID <= 0 {
		return fmt.Errorf("invalid user_id: %d", msg.UserID)
	}

	if msg.Language == "" {
		return fmt.Errorf("language is required")
	}

	if msg.Code == "" {
		return fmt.Errorf("code is required")
	}

	// 重构后：不再要求TestCases在消息中，将通过RPC获取
	logx.Infof("Validated simplified judge task: SubmissionID=%d, ProblemID=%d",
		msg.SubmissionID, msg.ProblemID)

	return nil
}

// 通过RPC调用题目服务获取题目详细信息
func (c *KafkaConsumer) fetchProblemDetails(problemID int64) (*ProblemDetails, error) {
	// 使用题目服务客户端获取题目信息（包含测试用例）
	problemInfo, err := c.problemClient.GetProblemDetail(context.Background(), problemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get problem detail: %w", err)
	}

	// 转换为调度器需要的格式
	var schedulerTestCases []*types.TestCase
	for _, tc := range problemInfo.TestCases {
		schedulerTestCases = append(schedulerTestCases, &types.TestCase{
			CaseId:         tc.CaseId,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
		})
	}

	return &ProblemDetails{
		ProblemID:   problemID,
		TimeLimit:   problemInfo.TimeLimit,
		MemoryLimit: problemInfo.MemoryLimit,
		TestCases:   schedulerTestCases,
	}, nil
}

// ProblemDetails 题目详细信息
type ProblemDetails struct {
	ProblemID   int64             `json:"problem_id"`
	TimeLimit   int               `json:"time_limit"`   // 毫秒
	MemoryLimit int               `json:"memory_limit"` // MB
	TestCases   []*types.TestCase `json:"test_cases"`
}

// 监控任务状态
func (c *KafkaConsumer) monitorTaskStatus(task *scheduler.JudgeTask) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(10 * time.Minute) // 10分钟超时
	defer timeout.Stop()

	for {
		select {
		case <-timeout.C:
			logx.Errorf("Task monitoring timeout for SubmissionID=%d", task.SubmissionID)
			return

		case <-ticker.C:
			// 获取任务状态
			currentTask, err := c.taskScheduler.GetTaskStatus(task.ID)
			if err != nil {
				logx.Errorf("Failed to get task status: %v", err)
				continue
			}

			// 检查任务是否完成
			if currentTask.Status == scheduler.TaskStatusCompleted {
				logx.Infof("Task completed successfully: SubmissionID=%d", task.SubmissionID)
				c.sendJudgeResult(currentTask, "completed")
				return

			} else if currentTask.Status == scheduler.TaskStatusFailed {
				logx.Errorf("Task failed: SubmissionID=%d, Error=%s", task.SubmissionID, currentTask.Error)
				c.sendJudgeResult(currentTask, "failed")
				return

			} else if currentTask.Status == scheduler.TaskStatusCancelled {
				logx.Infof("Task cancelled: SubmissionID=%d", task.SubmissionID)
				c.sendJudgeResult(currentTask, "cancelled")
				return
			}

			// 任务仍在进行中，发送状态更新
			c.sendStatusUpdate(currentTask)
		}
	}
}

// 发送判题结果
func (c *KafkaConsumer) sendJudgeResult(task *scheduler.JudgeTask, status string) {
	resultMessage := map[string]interface{}{
		"submission_id": task.SubmissionID,
		"status":        status,
		"timestamp":     time.Now().Unix(),
	}

	if task.Result != nil {
		resultMessage["result"] = map[string]interface{}{
			"verdict":     task.Result.Status,
			"score":       task.Result.Score,
			"time_used":   task.Result.TimeUsed,
			"memory_used": task.Result.MemoryUsed,
			"test_cases":  task.Result.TestCases,
		}
		resultMessage["compile_info"] = task.Result.CompileInfo
	}

	if task.Error != "" {
		resultMessage["error_message"] = task.Error
	}

	// 发送结果到Kafka
	if err := c.producer.PublishJudgeResult(context.Background(), resultMessage); err != nil {
		logx.Errorf("Failed to publish judge result: %v", err)
	}
}

// 发送状态更新
func (c *KafkaConsumer) sendStatusUpdate(task *scheduler.JudgeTask) {
	statusMessage := map[string]interface{}{
		"submission_id": task.SubmissionID,
		"status":        task.Status,
		"timestamp":     time.Now().Unix(),
	}

	// 添加进度信息
	if task.Status == scheduler.TaskStatusRunning {
		// 可以添加当前执行的测试用例信息
		statusMessage["progress"] = map[string]interface{}{
			"current_test_case": 0, // 这里需要从任务中获取实际进度
			"total_test_cases":  len(task.TestCases),
		}
	}

	if err := c.producer.PublishStatusUpdate(context.Background(), statusMessage); err != nil {
		logx.Errorf("Failed to publish status update: %v", err)
	}
}

// 处理消息错误
func (c *KafkaConsumer) handleMessageError(message *kafka.Message, err error) {
	logx.Errorf("Message processing failed - Topic: %s, Partition: %d, Offset: %d, Error: %v",
		message.Topic, message.Partition, message.Offset, err)

	// 可以实现死信队列逻辑
	// 或者记录到数据库用于后续重试
	deadLetterMessage := map[string]interface{}{
		"original_message": string(message.Value),
		"error":            err.Error(),
		"topic":            message.Topic,
		"partition":        message.Partition,
		"offset":           message.Offset,
		"timestamp":        time.Now().Unix(),
	}

	if dlqErr := c.producer.PublishDeadLetter(context.Background(), deadLetterMessage); dlqErr != nil {
		logx.Errorf("Failed to publish to dead letter queue: %v", dlqErr)
	}
}

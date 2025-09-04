package messagequeue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/config"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/dao"
	"github.com/dszqbsm/code-judger/services/submission-api/internal/websocket"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// 判题结果消息结构
type JudgeResultMessage struct {
	SubmissionID int64                      `json:"submission_id"`
	Status       string                     `json:"status"`
	Result       *ConsumerSubmissionResult  `json:"result,omitempty"`
	CompileInfo  *ConsumerCompileInfo       `json:"compile_info,omitempty"`
	ErrorMessage *string                    `json:"error_message,omitempty"`
	Timestamp    int64                      `json:"timestamp"`
}

// 提交结果详情
type ConsumerSubmissionResult struct {
	Verdict    string                      `json:"verdict"`
	Score      int                         `json:"score"`
	TimeUsed   int                         `json:"time_used"`
	MemoryUsed int                         `json:"memory_used"`
	TestCases  []ConsumerTestCaseResult    `json:"test_cases"`
}

// 测试用例结果
type ConsumerTestCaseResult struct {
	CaseID     int    `json:"case_id"`
	Status     string `json:"status"`
	TimeUsed   int    `json:"time_used"`
	MemoryUsed int    `json:"memory_used"`
	Input      string `json:"input,omitempty"`
	Output     string `json:"output,omitempty"`
	Expected   string `json:"expected,omitempty"`
}

// 编译信息
type ConsumerCompileInfo struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Time    int    `json:"time"`
}

// 状态更新消息结构
type StatusUpdateMessage struct {
	SubmissionID int64  `json:"submission_id"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
	Message      string `json:"message"`
	Timestamp    int64  `json:"timestamp"`
}

// Kafka消费者接口
type Consumer interface {
	Start(ctx context.Context) error
	Stop() error
}

// Kafka消费者实现
type KafkaConsumer struct {
	resultReader  *kafka.Reader
	statusReader  *kafka.Reader
	wsManager     *websocket.Manager
	submissionDao dao.SubmissionDao
	config        config.KafkaConf
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewKafkaConsumer(
	config config.KafkaConf, 
	wsManager *websocket.Manager, 
	submissionDao dao.SubmissionDao,
) Consumer {
	// 创建判题结果消费者
	resultReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           config.Brokers,
		Topic:             config.Topics.JudgeResult,
		GroupID:           config.Groups.SubmissionResult,
		MinBytes:          1,
		MaxBytes:          1e6,
		CommitInterval:    time.Second,
		StartOffset:       kafka.FirstOffset,
		MaxWait:           5 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		SessionTimeout:    30 * time.Second,
		RebalanceTimeout:  30 * time.Second,
		RetentionTime:     24 * time.Hour,
	})

	// 创建状态更新消费者
	statusReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           config.Brokers,
		Topic:             config.Topics.StatusUpdate,
		GroupID:           config.Groups.StatusUpdate,
		MinBytes:          1,
		MaxBytes:          1e6,
		CommitInterval:    time.Second,
		StartOffset:       kafka.FirstOffset,
		MaxWait:           5 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		SessionTimeout:    30 * time.Second,
		RebalanceTimeout:  30 * time.Second,
		RetentionTime:     24 * time.Hour,
	})

	ctx, cancel := context.WithCancel(context.Background())

	return &KafkaConsumer{
		resultReader:  resultReader,
		statusReader:  statusReader,
		wsManager:     wsManager,
		submissionDao: submissionDao,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// 启动消费者
func (c *KafkaConsumer) Start(ctx context.Context) error {
	logx.Info("启动判题结果消费者...")

	// 启动判题结果消费协程
	go c.consumeJudgeResults()

	// 启动状态更新消费协程
	go c.consumeStatusUpdates()

	logx.Info("判题结果消费者启动成功")
	return nil
}

// 停止消费者
func (c *KafkaConsumer) Stop() error {
	logx.Info("停止判题结果消费者...")

	c.cancel()

	if c.resultReader != nil {
		if err := c.resultReader.Close(); err != nil {
			logx.Errorf("关闭判题结果Reader失败: %v", err)
		}
	}

	if c.statusReader != nil {
		if err := c.statusReader.Close(); err != nil {
			logx.Errorf("关闭状态更新Reader失败: %v", err)
		}
	}

	logx.Info("判题结果消费者已停止")
	return nil
}

// 消费判题结果消息
func (c *KafkaConsumer) consumeJudgeResults() {
	retryCount := 0
	maxRetries := 10

	logx.Info("开始消费判题结果消息...")

	for {
		select {
		case <-c.ctx.Done():
			logx.Info("判题结果消费者上下文取消，停止消费")
			return
		default:
			// 创建带超时的上下文
			readCtx, cancel := context.WithTimeout(c.ctx, 30*time.Second)

			// 读取消息
			message, err := c.resultReader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if err == context.Canceled || readCtx.Err() == context.DeadlineExceeded {
					time.Sleep(1 * time.Second)
					continue
				}

				retryCount++
				logx.Errorf("读取判题结果消息失败 (重试 %d/%d): %v", retryCount, maxRetries, err)

				if retryCount >= maxRetries {
					logx.Errorf("达到最大重试次数，停止消费判题结果")
					return
				}

				backoff := time.Duration(retryCount*retryCount) * time.Second
				if backoff > 60*time.Second {
					backoff = 60 * time.Second
				}

				select {
				case <-c.ctx.Done():
					return
				case <-time.After(backoff):
					continue
				}
			}

			// 重置重试计数
			retryCount = 0

			logx.Infof("收到判题结果消息 - Topic: %s, Partition: %d, Offset: %d",
				message.Topic, message.Partition, message.Offset)

			// 处理消息
			if err := c.processJudgeResultMessage(&message); err != nil {
				logx.Errorf("处理判题结果消息失败: %v", err)
			}
		}
	}
}

// 消费状态更新消息
func (c *KafkaConsumer) consumeStatusUpdates() {
	retryCount := 0
	maxRetries := 10

	logx.Info("开始消费状态更新消息...")

	for {
		select {
		case <-c.ctx.Done():
			logx.Info("状态更新消费者上下文取消，停止消费")
			return
		default:
			// 创建带超时的上下文
			readCtx, cancel := context.WithTimeout(c.ctx, 30*time.Second)

			// 读取消息
			message, err := c.statusReader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if err == context.Canceled || readCtx.Err() == context.DeadlineExceeded {
					time.Sleep(1 * time.Second)
					continue
				}

				retryCount++
				logx.Errorf("读取状态更新消息失败 (重试 %d/%d): %v", retryCount, maxRetries, err)

				if retryCount >= maxRetries {
					logx.Errorf("达到最大重试次数，停止消费状态更新")
					return
				}

				backoff := time.Duration(retryCount*retryCount) * time.Second
				if backoff > 60*time.Second {
					backoff = 60 * time.Second
				}

				select {
				case <-c.ctx.Done():
					return
				case <-time.After(backoff):
					continue
				}
			}

			// 重置重试计数
			retryCount = 0

			logx.Infof("收到状态更新消息 - Topic: %s, Partition: %d, Offset: %d",
				message.Topic, message.Partition, message.Offset)

			// 处理消息
			if err := c.processStatusUpdateMessage(&message); err != nil {
				logx.Errorf("处理状态更新消息失败: %v", err)
			}
		}
	}
}

// 处理判题结果消息
func (c *KafkaConsumer) processJudgeResultMessage(message *kafka.Message) error {
	logx.Infof("处理判题结果消息，消息长度: %d bytes", len(message.Value))

	// 解析消息
	var resultMessage JudgeResultMessage
	if err := json.Unmarshal(message.Value, &resultMessage); err != nil {
		return fmt.Errorf("解析判题结果消息失败: %w", err)
	}

	logx.Infof("收到判题结果: SubmissionID=%d, Status=%s",
		resultMessage.SubmissionID, resultMessage.Status)

	// 1. 更新数据库中的提交状态
	if err := c.updateSubmissionStatus(&resultMessage); err != nil {
		logx.Errorf("更新提交状态失败: %v", err)
		// 继续处理WebSocket推送，即使数据库更新失败
	}

	// 2. 获取提交的用户ID
	userID, err := c.submissionDao.GetSubmissionUserID(c.ctx, resultMessage.SubmissionID)
	if err != nil {
		logx.Errorf("获取提交用户ID失败: %v", err)
		return fmt.Errorf("获取提交用户ID失败: %w", err)
	}

	// 3. 通过WebSocket推送给用户
	wsMessage := &websocket.Message{
		Type:         "judge_result",
		SubmissionID: &resultMessage.SubmissionID,
		UserID:       &userID,
		Data: map[string]interface{}{
			"submission_id": resultMessage.SubmissionID,
			"status":        resultMessage.Status,
			"result":        resultMessage.Result,
			"compile_info":  resultMessage.CompileInfo,
			"error_message": resultMessage.ErrorMessage,
			"timestamp":     resultMessage.Timestamp,
		},
		Timestamp: time.Now(),
	}

	// 推送WebSocket消息
	c.wsManager.BroadcastToUser(userID, wsMessage)

	logx.Infof("判题结果处理完成: SubmissionID=%d, UserID=%d, Status=%s",
		resultMessage.SubmissionID, userID, resultMessage.Status)

	return nil
}

// 处理状态更新消息
func (c *KafkaConsumer) processStatusUpdateMessage(message *kafka.Message) error {
	logx.Infof("处理状态更新消息，消息长度: %d bytes", len(message.Value))

	// 解析消息
	var statusMessage StatusUpdateMessage
	if err := json.Unmarshal(message.Value, &statusMessage); err != nil {
		return fmt.Errorf("解析状态更新消息失败: %w", err)
	}

	logx.Infof("收到状态更新: SubmissionID=%d, Status=%s, Progress=%d",
		statusMessage.SubmissionID, statusMessage.Status, statusMessage.Progress)

	// 获取提交的用户ID
	userID, err := c.submissionDao.GetSubmissionUserID(c.ctx, statusMessage.SubmissionID)
	if err != nil {
		logx.Errorf("获取提交用户ID失败: %v", err)
		return fmt.Errorf("获取提交用户ID失败: %w", err)
	}

	// 通过WebSocket推送状态更新
	wsMessage := &websocket.Message{
		Type:         "status_update",
		SubmissionID: &statusMessage.SubmissionID,
		UserID:       &userID,
		Data: map[string]interface{}{
			"submission_id": statusMessage.SubmissionID,
			"status":        statusMessage.Status,
			"progress":      statusMessage.Progress,
			"message":       statusMessage.Message,
			"timestamp":     statusMessage.Timestamp,
		},
		Timestamp: time.Now(),
	}

	// 推送WebSocket消息
	c.wsManager.BroadcastToUser(userID, wsMessage)

	logx.Infof("状态更新处理完成: SubmissionID=%d, UserID=%d, Status=%s",
		statusMessage.SubmissionID, userID, statusMessage.Status)

	return nil
}

// 更新提交状态到数据库
func (c *KafkaConsumer) updateSubmissionStatus(resultMessage *JudgeResultMessage) error {
	// 更新基本状态
	if err := c.submissionDao.UpdateSubmissionStatus(c.ctx, resultMessage.SubmissionID, resultMessage.Status); err != nil {
		return fmt.Errorf("更新提交状态失败: %w", err)
	}

	// 如果有详细结果，更新结果信息
	if resultMessage.Result != nil {
		resultData := map[string]interface{}{
			"verdict":     resultMessage.Result.Verdict,
			"score":       resultMessage.Result.Score,
			"time_used":   resultMessage.Result.TimeUsed,
			"memory_used": resultMessage.Result.MemoryUsed,
			"test_cases":  resultMessage.Result.TestCases,
		}

		if err := c.submissionDao.UpdateSubmissionResult(c.ctx, resultMessage.SubmissionID, resultData); err != nil {
			return fmt.Errorf("更新提交结果失败: %w", err)
		}
	}

	// 如果有编译信息，更新编译信息
	if resultMessage.CompileInfo != nil {
		compileData := map[string]interface{}{
			"success": resultMessage.CompileInfo.Success,
			"message": resultMessage.CompileInfo.Message,
			"time":    resultMessage.CompileInfo.Time,
		}

		if err := c.submissionDao.UpdateCompileInfo(c.ctx, resultMessage.SubmissionID, compileData); err != nil {
			return fmt.Errorf("更新编译信息失败: %w", err)
		}
	}

	return nil
}

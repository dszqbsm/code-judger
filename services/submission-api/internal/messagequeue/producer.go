package messagequeue

import (
	"context"
	"encoding/json"
	"fmt"

	"code-judger/services/submission-api/internal/config"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type Producer interface {
	PublishJudgeTask(ctx context.Context, taskData []byte) error
	PublishStatusUpdate(ctx context.Context, statusData []byte) error
	Close() error
}

type KafkaProducer struct {
	judgeTaskWriter    *kafka.Writer
	statusUpdateWriter *kafka.Writer
	config             config.KafkaConf
}

func NewKafkaProducer(config config.KafkaConf) Producer {
	// 创建判题任务生产者
	judgeTaskWriter := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.Topics.JudgeTask,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	// 创建状态更新生产者
	statusUpdateWriter := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.Topics.StatusUpdate,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	return &KafkaProducer{
		judgeTaskWriter:    judgeTaskWriter,
		statusUpdateWriter: statusUpdateWriter,
		config:             config,
	}
}

// PublishJudgeTask 发布判题任务
func (p *KafkaProducer) PublishJudgeTask(ctx context.Context, taskData []byte) error {
	message := kafka.Message{
		Key:   nil,
		Value: taskData,
	}

	err := p.judgeTaskWriter.WriteMessages(ctx, message)
	if err != nil {
		logx.Errorf("发布判题任务失败: %v", err)
		return fmt.Errorf("发布判题任务失败: %v", err)
	}

	logx.Infof("判题任务发布成功, 消息大小: %d bytes", len(taskData))
	return nil
}

// PublishStatusUpdate 发布状态更新
func (p *KafkaProducer) PublishStatusUpdate(ctx context.Context, statusData []byte) error {
	message := kafka.Message{
		Key:   nil,
		Value: statusData,
	}

	err := p.statusUpdateWriter.WriteMessages(ctx, message)
	if err != nil {
		logx.Errorf("发布状态更新失败: %v", err)
		return fmt.Errorf("发布状态更新失败: %v", err)
	}

	logx.Infof("状态更新发布成功, 消息大小: %d bytes", len(statusData))
	return nil
}

// Close 关闭生产者
func (p *KafkaProducer) Close() error {
	var err1, err2 error

	if p.judgeTaskWriter != nil {
		err1 = p.judgeTaskWriter.Close()
	}

	if p.statusUpdateWriter != nil {
		err2 = p.statusUpdateWriter.Close()
	}

	if err1 != nil {
		return err1
	}
	return err2
}

// StatusUpdateMessage 状态更新消息
type StatusUpdateMessage struct {
	SubmissionID int64             `json:"submission_id"`
	Status       string            `json:"status"`
	Result       *SubmissionResult `json:"result,omitempty"`
	CompileInfo  *CompileInfo      `json:"compile_info,omitempty"`
	ErrorMessage *string           `json:"error_message,omitempty"`
	Timestamp    int64             `json:"timestamp"`
}

// SubmissionResult 判题结果
type SubmissionResult struct {
	Verdict    string           `json:"verdict"`
	Score      int              `json:"score"`
	TimeUsed   int              `json:"time_used"`
	MemoryUsed int              `json:"memory_used"`
	TestCases  []TestCaseResult `json:"test_cases"`
}

// TestCaseResult 测试用例结果
type TestCaseResult struct {
	CaseID     int    `json:"case_id"`
	Status     string `json:"status"`
	TimeUsed   int    `json:"time_used"`
	MemoryUsed int    `json:"memory_used"`
	Input      string `json:"input,omitempty"`
	Output     string `json:"output,omitempty"`
	Expected   string `json:"expected,omitempty"`
}

// CompileInfo 编译信息
type CompileInfo struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Time    int    `json:"time"`
}

// PublishStatusUpdateMessage 发布格式化的状态更新消息
func (p *KafkaProducer) PublishStatusUpdateMessage(ctx context.Context, msg *StatusUpdateMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化状态更新消息失败: %v", err)
	}

	return p.PublishStatusUpdate(ctx, data)
}


package messagequeue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dszqbsm/code-judger/services/submission-api/internal/config"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type Producer interface {
	PublishJudgeTask(ctx context.Context, taskData interface{}, taskType ...string) error
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
		Addr:                   kafka.TCP(config.Brokers...),
		Topic:                  config.Topics.JudgeTask,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne,
		Async:                  true,                          // 启用异步发送，提高性能
		WriteTimeout:           5 * time.Second,               // 增加写入超时到5秒
		ReadTimeout:            5 * time.Second,               // 增加读取超时到5秒
		BatchTimeout:           100 * time.Millisecond,        // 批处理超时100ms
		BatchSize:              10,                            // 增加批处理大小到10，提高吞吐量
		AllowAutoTopicCreation: true,                          // 允许自动创建主题
		ErrorLogger:            kafka.LoggerFunc(logx.Errorf), // 异步错误日志
	}

	// 创建状态更新生产者
	statusUpdateWriter := &kafka.Writer{
		Addr:                   kafka.TCP(config.Brokers...),
		Topic:                  config.Topics.StatusUpdate,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne,
		Async:                  true,                          // 启用异步发送，提高性能
		WriteTimeout:           5 * time.Second,               // 增加写入超时到5秒
		ReadTimeout:            5 * time.Second,               // 增加读取超时到5秒
		BatchTimeout:           200 * time.Millisecond,        // 状态更新可以稍微延迟批处理
		BatchSize:              5,                             // 适中的批处理大小
		AllowAutoTopicCreation: true,                          // 允许自动创建主题
		ErrorLogger:            kafka.LoggerFunc(logx.Errorf), // 异步错误日志
	}

	return &KafkaProducer{
		judgeTaskWriter:    judgeTaskWriter,
		statusUpdateWriter: statusUpdateWriter,
		config:             config,
	}
}

// PublishJudgeTask 发布判题任务
func (p *KafkaProducer) PublishJudgeTask(ctx context.Context, taskData interface{}, taskType ...string) error {
	logx.Infof("Kafka生产者: 开始发布判题任务")

	// 序列化任务数据
	var data []byte
	var err error

	logx.Infof("Kafka生产者: 开始序列化任务数据")
	switch v := taskData.(type) {
	case []byte:
		data = v
	default:
		data, err = json.Marshal(taskData)
		if err != nil {
			logx.Errorf("序列化判题任务失败: %v", err)
			return fmt.Errorf("序列化判题任务失败: %v", err)
		}
	}
	logx.Infof("Kafka生产者: 任务数据序列化完成, 大小: %d bytes", len(data))
	// 安全地显示消息内容预览
	preview := string(data)
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	logx.Infof("Kafka生产者: 消息内容预览: %s", preview)

	// 构建消息
	message := kafka.Message{
		Value: data,
	}

	// 添加任务类型标识
	if len(taskType) > 0 && taskType[0] != "" {
		message.Key = []byte(taskType[0])
		logx.Infof("发布%s任务", taskType[0])
	}

	logx.Infof("Kafka生产者: 开始写入消息到主题: %s, Brokers: %v", p.judgeTaskWriter.Topic, p.config.Brokers)
	logx.Infof("Kafka生产者: Writer配置 - Addr: %v, Topic: %s, Async: %v", p.judgeTaskWriter.Addr, p.judgeTaskWriter.Topic, p.judgeTaskWriter.Async)

	// 异步模式下的消息发送
	logx.Infof("Kafka生产者: 异步发送消息")
	err = p.judgeTaskWriter.WriteMessages(ctx, message)
	if err != nil {
		// 异步模式下，这里的错误主要是连接或配置错误
		logx.Errorf("Kafka生产者: 消息提交到发送队列失败: %v", err)
		logx.Errorf("Kafka生产者: 错误详情 - Type: %T, Error: %s", err, err.Error())
		return fmt.Errorf("发布判题任务失败: %v", err)
	}

	// 异步模式下，消息已提交到内部队列，实际发送由后台处理
	// 发送错误会通过ErrorLogger异步记录
	logx.Infof("Kafka生产者: 消息已提交到异步发送队列")

	// 可选：等待一小段时间确保消息开始处理（非必需）
	// time.Sleep(1 * time.Millisecond)

	logx.Infof("判题任务发布成功, 消息大小: %d bytes", len(data))
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

// ProducerStatusUpdateMessage 生产者状态更新消息
type ProducerStatusUpdateMessage struct {
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
func (p *KafkaProducer) PublishStatusUpdateMessage(ctx context.Context, msg *ProducerStatusUpdateMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化状态更新消息失败: %v", err)
	}

	return p.PublishStatusUpdate(ctx, data)
}

package messagequeue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/config"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// 生产者接口
type Producer interface {
	PublishJudgeResult(ctx context.Context, resultData interface{}) error
	PublishStatusUpdate(ctx context.Context, statusData interface{}) error
	PublishDeadLetter(ctx context.Context, deadLetterData interface{}) error
	Close() error
}

// Kafka生产者实现
type KafkaProducer struct {
	resultWriter     *kafka.Writer
	statusWriter     *kafka.Writer
	deadLetterWriter *kafka.Writer
	config           config.KafkaConf
}

func NewKafkaProducer(config config.KafkaConf) Producer {
	// 判题结果生产者 - 异步模式
	resultWriter := &kafka.Writer{
		Addr:                   kafka.TCP(config.Brokers...),
		Topic:                  config.ResultTopic,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne,
		Async:                  true,                          // 启用异步发送
		WriteTimeout:           5 * time.Second,               // 写入超时
		ReadTimeout:            5 * time.Second,               // 读取超时
		BatchTimeout:           50 * time.Millisecond,         // 结果消息批处理超时较短
		BatchSize:              5,                             // 适中的批处理大小
		AllowAutoTopicCreation: true,                          // 允许自动创建主题
		ErrorLogger:            kafka.LoggerFunc(logx.Errorf), // 异步错误日志
	}

	// 状态更新生产者 - 异步模式
	statusWriter := &kafka.Writer{
		Addr:                   kafka.TCP(config.Brokers...),
		Topic:                  config.StatusTopic,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne,
		Async:                  true,                          // 启用异步发送
		WriteTimeout:           3 * time.Second,               // 状态更新超时稍短
		ReadTimeout:            3 * time.Second,               // 读取超时
		BatchTimeout:           100 * time.Millisecond,        // 状态更新可以稍微延迟
		BatchSize:              10,                            // 状态更新批处理大小可以更大
		AllowAutoTopicCreation: true,                          // 允许自动创建主题
		ErrorLogger:            kafka.LoggerFunc(logx.Errorf), // 异步错误日志
	}

	// 死信队列生产者 - 同步模式（确保错误消息不丢失）
	deadLetterWriter := &kafka.Writer{
		Addr:                   kafka.TCP(config.Brokers...),
		Topic:                  config.DeadLetterTopic,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireAll, // 死信队列要求所有副本确认
		Async:                  false,            // 死信队列使用同步模式确保可靠性
		WriteTimeout:           10 * time.Second, // 死信队列超时更长
		ReadTimeout:            10 * time.Second, // 读取超时
		AllowAutoTopicCreation: true,             // 允许自动创建主题
	}

	return &KafkaProducer{
		resultWriter:     resultWriter,
		statusWriter:     statusWriter,
		deadLetterWriter: deadLetterWriter,
		config:           config,
	}
}

// 发布判题结果
func (p *KafkaProducer) PublishJudgeResult(ctx context.Context, resultData interface{}) error {
	data, err := json.Marshal(resultData)
	if err != nil {
		return fmt.Errorf("failed to marshal judge result: %w", err)
	}

	message := kafka.Message{
		Value: data,
	}

	// 异步发送判题结果
	if err := p.resultWriter.WriteMessages(ctx, message); err != nil {
		// 异步模式下，这里的错误主要是连接或配置错误
		logx.Errorf("Failed to submit judge result to async queue: %v", err)
		return fmt.Errorf("failed to publish judge result: %w", err)
	}

	// 异步模式下，消息已提交到内部队列，实际发送由后台处理
	logx.Infof("Judge result submitted to async queue successfully, message size: %d bytes", len(data))
	return nil
}

// 发布状态更新
func (p *KafkaProducer) PublishStatusUpdate(ctx context.Context, statusData interface{}) error {
	data, err := json.Marshal(statusData)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	message := kafka.Message{
		Value: data,
	}

	// 异步发送状态更新
	if err := p.statusWriter.WriteMessages(ctx, message); err != nil {
		// 异步模式下，这里的错误主要是连接或配置错误
		logx.Errorf("Failed to submit status update to async queue: %v", err)
		return fmt.Errorf("failed to publish status update: %w", err)
	}

	// 异步模式下，消息已提交到内部队列，实际发送由后台处理
	logx.Infof("Status update submitted to async queue successfully, message size: %d bytes", len(data))
	return nil
}

// 发布到死信队列
func (p *KafkaProducer) PublishDeadLetter(ctx context.Context, deadLetterData interface{}) error {
	data, err := json.Marshal(deadLetterData)
	if err != nil {
		return fmt.Errorf("failed to marshal dead letter: %w", err)
	}

	message := kafka.Message{
		Value: data,
	}

	if err := p.deadLetterWriter.WriteMessages(ctx, message); err != nil {
		logx.Errorf("Failed to publish dead letter: %v", err)
		return fmt.Errorf("failed to publish dead letter: %w", err)
	}

	logx.Infof("Dead letter published successfully, message size: %d bytes", len(data))
	return nil
}

// 关闭生产者
func (p *KafkaProducer) Close() error {
	var errors []error

	if p.resultWriter != nil {
		if err := p.resultWriter.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close result writer: %w", err))
		}
	}

	if p.statusWriter != nil {
		if err := p.statusWriter.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close status writer: %w", err))
		}
	}

	if p.deadLetterWriter != nil {
		if err := p.deadLetterWriter.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close dead letter writer: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing producers: %v", errors)
	}

	return nil
}

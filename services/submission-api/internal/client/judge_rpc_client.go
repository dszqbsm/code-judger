package client

import (
	"context"
	"fmt"
	"time"

	"github.com/dszqbsm/code-judger/common/rpc"
	"github.com/zeromicro/go-zero/core/logx"
)

// JudgeRPCClient 基于Consul的判题服务RPC客户端
type JudgeRPCClient struct {
	rpcClient     *rpc.HTTPRPCClient
	circuitBreaker *rpc.SimpleCircuitBreaker
	retryConfig   rpc.RetryConfig
}

// NewJudgeRPCClient 创建判题服务RPC客户端
func NewJudgeRPCClient(consulAddr string, timeout time.Duration) (JudgeServiceClient, error) {
	// 创建RPC客户端
	rpcClient, err := rpc.NewHTTPRPCClient("judge-api", consulAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("创建判题服务RPC客户端失败: %w", err)
	}

	// 创建熔断器
	circuitBreaker := rpc.NewSimpleCircuitBreaker(5, 30*time.Second)

	// 配置重试策略
	retryConfig := rpc.RetryConfig{
		MaxRetries:    2,
		RetryDelay:    500 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	return &JudgeRPCClient{
		rpcClient:      rpcClient,
		circuitBreaker: circuitBreaker,
		retryConfig:    retryConfig,
	}, nil
}

// GetJudgeResult 获取判题结果
func (c *JudgeRPCClient) GetJudgeResult(ctx context.Context, submissionID int64) (*JudgeResultResp, error) {
	var result JudgeResultResp

	err := c.circuitBreaker.Call(ctx, func() error {
		return rpc.WithRetry(ctx, c.retryConfig, func() error {
			path := fmt.Sprintf("/api/v1/judge/result/%d", submissionID)
			return c.rpcClient.Get(ctx, path, &result)
		})
	})

	if err != nil {
		logx.WithContext(ctx).Errorf("RPC调用获取判题结果失败: SubmissionID=%d, Error=%v", submissionID, err)
		return nil, fmt.Errorf("获取判题结果失败: %w", err)
	}

	logx.WithContext(ctx).Infof("RPC调用获取判题结果成功: SubmissionID=%d, Status=%s", 
		submissionID, result.Data.Status)

	return &result, nil
}

// GetJudgeStatus 获取判题状态
func (c *JudgeRPCClient) GetJudgeStatus(ctx context.Context, submissionID int64) (*JudgeStatusResp, error) {
	var result JudgeStatusResp

	err := c.circuitBreaker.Call(ctx, func() error {
		return rpc.WithRetry(ctx, c.retryConfig, func() error {
			path := fmt.Sprintf("/api/v1/judge/status/%d", submissionID)
			return c.rpcClient.Get(ctx, path, &result)
		})
	})

	if err != nil {
		logx.WithContext(ctx).Errorf("RPC调用获取判题状态失败: SubmissionID=%d, Error=%v", submissionID, err)
		return nil, fmt.Errorf("获取判题状态失败: %w", err)
	}

	logx.WithContext(ctx).Infof("RPC调用获取判题状态成功: SubmissionID=%d, Status=%s", 
		submissionID, result.Data.Status)

	return &result, nil
}

// CancelJudge 取消判题
func (c *JudgeRPCClient) CancelJudge(ctx context.Context, submissionID int64) (*CancelJudgeResp, error) {
	var result CancelJudgeResp

	err := c.circuitBreaker.Call(ctx, func() error {
		return rpc.WithRetry(ctx, c.retryConfig, func() error {
			path := fmt.Sprintf("/api/v1/judge/cancel/%d", submissionID)
			return c.rpcClient.Delete(ctx, path, &result)
		})
	})

	if err != nil {
		logx.WithContext(ctx).Errorf("RPC调用取消判题失败: SubmissionID=%d, Error=%v", submissionID, err)
		return nil, fmt.Errorf("取消判题失败: %w", err)
	}

	logx.WithContext(ctx).Infof("RPC调用取消判题成功: SubmissionID=%d", submissionID)

	return &result, nil
}

// RejudgeSubmission 重新判题
func (c *JudgeRPCClient) RejudgeSubmission(ctx context.Context, submissionID int64) (*RejudgeResp, error) {
	var result RejudgeResp

	err := c.circuitBreaker.Call(ctx, func() error {
		return rpc.WithRetry(ctx, c.retryConfig, func() error {
			path := fmt.Sprintf("/api/v1/judge/rejudge/%d", submissionID)
			return c.rpcClient.Post(ctx, path, nil, &result)
		})
	})

	if err != nil {
		logx.WithContext(ctx).Errorf("RPC调用重新判题失败: SubmissionID=%d, Error=%v", submissionID, err)
		return nil, fmt.Errorf("重新判题失败: %w", err)
	}

	logx.WithContext(ctx).Infof("RPC调用重新判题成功: SubmissionID=%d", submissionID)

	return &result, nil
}

// GetJudgeQueue 获取队列状态
func (c *JudgeRPCClient) GetJudgeQueue(ctx context.Context) (*JudgeQueueResp, error) {
	var result JudgeQueueResp

	err := c.circuitBreaker.Call(ctx, func() error {
		return rpc.WithRetry(ctx, c.retryConfig, func() error {
			path := "/api/v1/judge/queue"
			return c.rpcClient.Get(ctx, path, &result)
		})
	})

	if err != nil {
		logx.WithContext(ctx).Errorf("RPC调用获取队列状态失败: Error=%v", err)
		return nil, fmt.Errorf("获取队列状态失败: %w", err)
	}

	logx.WithContext(ctx).Infof("RPC调用获取队列状态成功: QueueLength=%d", result.Data.QueueLength)

	return &result, nil
}

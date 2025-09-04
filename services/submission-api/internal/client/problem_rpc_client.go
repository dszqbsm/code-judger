package client

import (
	"context"
	"fmt"
	"time"

	"github.com/dszqbsm/code-judger/common/rpc"
	"github.com/zeromicro/go-zero/core/logx"
)

// ProblemServiceClient 题目服务客户端接口
type ProblemServiceClient interface {
	GetProblem(ctx context.Context, problemID int64) (*ProblemResp, error)
	GetProblemTestCases(ctx context.Context, problemID int64) (*TestCasesResp, error)
	ValidateProblemAccess(ctx context.Context, problemID int64, userID int64) (*AccessValidationResp, error)
}

// ProblemResp 题目响应
type ProblemResp struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    Problem `json:"data"`
}

// Problem 题目信息
type Problem struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	TimeLimit   int    `json:"time_limit"`   // 毫秒
	MemoryLimit int    `json:"memory_limit"` // MB
	Difficulty  int    `json:"difficulty"`
	IsPublic    bool   `json:"is_public"`
	CreatorID   int64  `json:"creator_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// TestCasesResp 测试用例响应
type TestCasesResp struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    []TestCase `json:"data"`
}

// TestCase 测试用例
type TestCase struct {
	CaseID   int    `json:"case_id"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
	IsHidden bool   `json:"is_hidden"`
}

// AccessValidationResp 访问权限验证响应
type AccessValidationResp struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    AccessValidation   `json:"data"`
}

// AccessValidation 访问权限验证结果
type AccessValidation struct {
	HasAccess bool   `json:"has_access"`
	Reason    string `json:"reason"`
}

// ProblemRPCClient 基于Consul的题目服务RPC客户端
type ProblemRPCClient struct {
	rpcClient     *rpc.HTTPRPCClient
	circuitBreaker *rpc.SimpleCircuitBreaker
	retryConfig   rpc.RetryConfig
}

// NewProblemRPCClient 创建题目服务RPC客户端
func NewProblemRPCClient(consulAddr string, timeout time.Duration) (ProblemServiceClient, error) {
	// 创建RPC客户端
	rpcClient, err := rpc.NewHTTPRPCClient("problem-api", consulAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("创建题目服务RPC客户端失败: %w", err)
	}

	// 创建熔断器
	circuitBreaker := rpc.NewSimpleCircuitBreaker(5, 30*time.Second)

	// 配置重试策略
	retryConfig := rpc.RetryConfig{
		MaxRetries:    2,
		RetryDelay:    300 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	return &ProblemRPCClient{
		rpcClient:      rpcClient,
		circuitBreaker: circuitBreaker,
		retryConfig:    retryConfig,
	}, nil
}

// GetProblem 获取题目信息
func (c *ProblemRPCClient) GetProblem(ctx context.Context, problemID int64) (*ProblemResp, error) {
	var result ProblemResp

	err := c.circuitBreaker.Call(ctx, func() error {
		return rpc.WithRetry(ctx, c.retryConfig, func() error {
			path := fmt.Sprintf("/api/v1/problems/%d", problemID)
			return c.rpcClient.Get(ctx, path, &result)
		})
	})

	if err != nil {
		logx.WithContext(ctx).Errorf("RPC调用获取题目信息失败: ProblemID=%d, Error=%v", problemID, err)
		return nil, fmt.Errorf("获取题目信息失败: %w", err)
	}

	logx.WithContext(ctx).Infof("RPC调用获取题目信息成功: ProblemID=%d, Title=%s", 
		problemID, result.Data.Title)

	return &result, nil
}

// GetProblemTestCases 获取题目测试用例
func (c *ProblemRPCClient) GetProblemTestCases(ctx context.Context, problemID int64) (*TestCasesResp, error) {
	var result TestCasesResp

	err := c.circuitBreaker.Call(ctx, func() error {
		return rpc.WithRetry(ctx, c.retryConfig, func() error {
			path := fmt.Sprintf("/api/v1/problems/%d/testcases", problemID)
			return c.rpcClient.Get(ctx, path, &result)
		})
	})

	if err != nil {
		logx.WithContext(ctx).Errorf("RPC调用获取测试用例失败: ProblemID=%d, Error=%v", problemID, err)
		return nil, fmt.Errorf("获取测试用例失败: %w", err)
	}

	logx.WithContext(ctx).Infof("RPC调用获取测试用例成功: ProblemID=%d, Count=%d", 
		problemID, len(result.Data))

	return &result, nil
}

// ValidateProblemAccess 验证题目访问权限
func (c *ProblemRPCClient) ValidateProblemAccess(ctx context.Context, problemID int64, userID int64) (*AccessValidationResp, error) {
	var result AccessValidationResp

	request := map[string]interface{}{
		"problem_id": problemID,
		"user_id":    userID,
	}

	err := c.circuitBreaker.Call(ctx, func() error {
		return rpc.WithRetry(ctx, c.retryConfig, func() error {
			path := "/api/v1/problems/validate-access"
			return c.rpcClient.Post(ctx, path, request, &result)
		})
	})

	if err != nil {
		logx.WithContext(ctx).Errorf("RPC调用验证题目访问权限失败: ProblemID=%d, UserID=%d, Error=%v", 
			problemID, userID, err)
		return nil, fmt.Errorf("验证题目访问权限失败: %w", err)
	}

	logx.WithContext(ctx).Infof("RPC调用验证题目访问权限成功: ProblemID=%d, UserID=%d, HasAccess=%v", 
		problemID, userID, result.Data.HasAccess)

	return &result, nil
}

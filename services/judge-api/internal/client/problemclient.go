package client

import (
	"context"
	"fmt"
	"time"

	"github.com/online-judge/code-judger/services/judge-api/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// 题目服务客户端接口
type ProblemServiceClient interface {
	GetProblemDetail(ctx context.Context, problemId int64) (*types.ProblemInfo, error)
}

// 模拟客户端（用于开发和测试）
type MockProblemClient struct{}

func NewMockProblemClient() ProblemServiceClient {
	return &MockProblemClient{}
}

func (c *MockProblemClient) GetProblemDetail(ctx context.Context, problemId int64) (*types.ProblemInfo, error) {
	// 模拟网络延迟
	time.Sleep(50 * time.Millisecond)

	// 模拟题目不存在的情况
	if problemId <= 0 || problemId > 10000 {
		return nil, fmt.Errorf("题目不存在: %d", problemId)
	}

	logx.Infof("Mock: fetching problem detail for problem_id=%d", problemId)

	// 返回模拟的题目信息
	return &types.ProblemInfo{
		ProblemId:   problemId,
		Title:       fmt.Sprintf("算法题目 %d", problemId),
		TimeLimit:   1000, // 1秒
		MemoryLimit: 128,  // 128MB
		Languages:   []string{"cpp", "c", "java", "python", "go", "javascript"},
		TestCases: []types.TestCase{
			{
				CaseId:         1,
				Input:          "1 2",
				ExpectedOutput: "3",
				TimeLimit:      0, // 使用全局限制
				MemoryLimit:    0, // 使用全局限制
			},
			{
				CaseId:         2,
				Input:          "5 10",
				ExpectedOutput: "15",
				TimeLimit:      0,
				MemoryLimit:    0,
			},
			{
				CaseId:         3,
				Input:          "100 200",
				ExpectedOutput: "300",
				TimeLimit:      0,
				MemoryLimit:    0,
			},
		},
		IsPublic: true,
	}, nil
}

// HTTP客户端实现（真实环境使用）
type HttpProblemClient struct {
	baseURL string
	timeout time.Duration
}

func NewHttpProblemClient(baseURL string) ProblemServiceClient {
	return &HttpProblemClient{
		baseURL: baseURL,
		timeout: 10 * time.Second,
	}
}

func (c *HttpProblemClient) GetProblemDetail(ctx context.Context, problemId int64) (*types.ProblemInfo, error) {
	// TODO: 实现真实的HTTP调用
	// 这里暂时返回模拟数据，实际部署时需要实现HTTP客户端
	logx.Infof("HTTP: fetching problem detail from %s for problem_id=%d", c.baseURL, problemId)

	// 模拟HTTP调用
	time.Sleep(100 * time.Millisecond)

	if problemId <= 0 {
		return nil, fmt.Errorf("题目不存在: %d", problemId)
	}

	return &types.ProblemInfo{
		ProblemId:   problemId,
		Title:       fmt.Sprintf("HTTP获取的题目 %d", problemId),
		TimeLimit:   2000, // 2秒
		MemoryLimit: 256,  // 256MB
		Languages:   []string{"cpp", "c", "java", "python", "go"},
		TestCases: []types.TestCase{
			{
				CaseId:         1,
				Input:          "1 1",
				ExpectedOutput: "2",
			},
		},
		IsPublic: true,
	}, nil
}

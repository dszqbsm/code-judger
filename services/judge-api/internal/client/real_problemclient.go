package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dszqbsm/code-judger/services/judge-api/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// 真实的HTTP客户端实现
type RealHttpProblemClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string // API密钥用于服务间认证
}

func NewRealHttpProblemClient(baseURL string, timeout time.Duration, apiKey string) ProblemServiceClient {
	return &RealHttpProblemClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

func (c *RealHttpProblemClient) GetProblemDetail(ctx context.Context, problemId int64) (*types.ProblemInfo, error) {
	// 1. 构建请求URL
	url := fmt.Sprintf("%s/api/v1/problems/%d", c.baseURL, problemId)

	// 2. 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 3. 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "judge-api/1.0.0")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		// 或者使用服务间认证：
		// req.Header.Set("X-Service-Token", c.apiKey)
	}

	// 4. 记录请求日志
	startTime := time.Now()
	logx.WithContext(ctx).Infof("Calling problem service: GET %s", url)

	// 5. 发送HTTP请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logx.WithContext(ctx).Errorf("HTTP请求失败: %v", err)
		return nil, fmt.Errorf("请求题目服务失败: %w", err)
	}
	defer resp.Body.Close()

	// 6. 记录响应时间
	duration := time.Since(startTime)
	logx.WithContext(ctx).Infof("Problem service response: status=%d, duration=%v", resp.StatusCode, duration)

	// 7. 处理HTTP状态码
	switch resp.StatusCode {
	case http.StatusOK:
		// 正常情况，继续处理
	case http.StatusNotFound:
		return nil, fmt.Errorf("题目不存在: %d", problemId)
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("服务认证失败")
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("请求频率过高，请稍后重试")
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("题目服务内部错误")
	default:
		return nil, fmt.Errorf("题目服务返回异常状态码: %d", resp.StatusCode)
	}

	// 8. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 9. 解析JSON响应
	var response struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    types.ProblemInfo `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		logx.WithContext(ctx).Errorf("JSON解析失败: %v, body: %s", err, string(body))
		return nil, fmt.Errorf("解析题目服务响应失败: %w", err)
	}

	// 10. 检查业务状态码
	if response.Code != 200 {
		return nil, fmt.Errorf("题目服务业务错误[%d]: %s", response.Code, response.Message)
	}

	// 11. 验证响应数据完整性
	if err := c.validateProblemInfo(&response.Data, problemId); err != nil {
		return nil, fmt.Errorf("题目信息验证失败: %w", err)
	}

	// 12. 记录成功日志
	logx.WithContext(ctx).Infof("Successfully fetched problem %d: %s, time_limit=%dms, memory_limit=%dMB, test_cases=%d",
		problemId, response.Data.Title, response.Data.TimeLimit, response.Data.MemoryLimit, len(response.Data.TestCases))

	return &response.Data, nil
}

// 验证题目信息的完整性和合理性
func (c *RealHttpProblemClient) validateProblemInfo(info *types.ProblemInfo, expectedId int64) error {
	if info.ProblemId != expectedId {
		return fmt.Errorf("题目ID不匹配: expected=%d, actual=%d", expectedId, info.ProblemId)
	}

	if info.Title == "" {
		return fmt.Errorf("题目标题为空")
	}

	if info.TimeLimit <= 0 || info.TimeLimit > 30000 {
		return fmt.Errorf("时间限制不合理: %dms", info.TimeLimit)
	}

	if info.MemoryLimit <= 0 || info.MemoryLimit > 1024 {
		return fmt.Errorf("内存限制不合理: %dMB", info.MemoryLimit)
	}

	if len(info.Languages) == 0 {
		return fmt.Errorf("支持的编程语言列表为空")
	}

	if len(info.TestCases) == 0 {
		return fmt.Errorf("测试用例列表为空")
	}

	// 验证测试用例
	for i, testCase := range info.TestCases {
		if testCase.Input == "" && testCase.ExpectedOutput == "" {
			return fmt.Errorf("测试用例[%d]输入输出都为空", i+1)
		}
	}

	return nil
}

// 带重试机制的客户端
type RetryableProblemClient struct {
	client     ProblemServiceClient
	maxRetries int
	retryDelay time.Duration
}

func NewRetryableProblemClient(client ProblemServiceClient, maxRetries int, retryDelay time.Duration) ProblemServiceClient {
	return &RetryableProblemClient{
		client:     client,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

func (c *RetryableProblemClient) GetProblemDetail(ctx context.Context, problemId int64) (*types.ProblemInfo, error) {
	var lastErr error

	for i := 0; i <= c.maxRetries; i++ {
		if i > 0 {
			// 重试前等待
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay):
			}

			logx.WithContext(ctx).Infof("Retrying problem service call, attempt %d/%d", i+1, c.maxRetries+1)
		}

		result, err := c.client.GetProblemDetail(ctx, problemId)
		if err == nil {
			if i > 0 {
				logx.WithContext(ctx).Infof("Problem service call succeeded after %d retries", i)
			}
			return result, nil
		}

		lastErr = err

		// 判断是否应该重试
		if !c.shouldRetry(err) {
			break
		}
	}

	logx.WithContext(ctx).Errorf("Problem service call failed after %d attempts: %v", c.maxRetries+1, lastErr)
	return nil, fmt.Errorf("题目服务调用失败(重试%d次): %w", c.maxRetries, lastErr)
}

func (c *RetryableProblemClient) shouldRetry(err error) bool {
	// 网络错误、超时、5xx错误应该重试
	// 4xx错误（如题目不存在）不应该重试
	errorMsg := err.Error()

	// 不重试的情况
	if contains(errorMsg, "题目不存在") ||
		contains(errorMsg, "认证失败") ||
		contains(errorMsg, "验证失败") {
		return false
	}

	// 重试的情况
	if contains(errorMsg, "timeout") ||
		contains(errorMsg, "connection") ||
		contains(errorMsg, "服务内部错误") {
		return true
	}

	return true // 默认重试
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || (len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				len(s) > len(substr) && findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

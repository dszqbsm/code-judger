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
	baseURL    string
	httpClient *http.Client
	apiKey     string // API密钥用于服务间认证
}

func NewHttpProblemClient(baseURL string) ProblemServiceClient {
	return &HttpProblemClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
			},
		},
		apiKey: "judge-service-api-key", // 实际部署时从配置读取
	}
}

func NewHttpProblemClientWithAuth(baseURL string, apiKey string) ProblemServiceClient {
	return &HttpProblemClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
			},
		},
		apiKey: apiKey,
	}
}

func (c *HttpProblemClient) GetProblemDetail(ctx context.Context, problemId int64) (*types.ProblemInfo, error) {
	// 1. 构建请求URL（使用内部接口）
	url := fmt.Sprintf("%s/internal/v1/problems/%d", c.baseURL, problemId)

	// 2. 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 3. 设置请求头（内部服务调用，使用内部API密钥）
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "judge-api/1.0.0")
	req.Header.Set("X-Internal-API-Key", "internal-service-secret-key-2024") // 内部API密钥

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
	case 200:
		// 正常情况，继续处理
	case 404:
		return nil, fmt.Errorf("题目不存在: %d", problemId)
	case 401:
		return nil, fmt.Errorf("服务认证失败")
	case 429:
		return nil, fmt.Errorf("请求频率过高，请稍后重试")
	case 500:
		return nil, fmt.Errorf("题目服务内部错误")
	default:
		return nil, fmt.Errorf("题目服务返回异常状态码: %d", resp.StatusCode)
	}

	// 8. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 9. 解析JSON响应（内部接口的响应结构）
	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Problem struct {
				Id           int64    `json:"id"`
				Title        string   `json:"title"`
				Description  string   `json:"description"`
				InputFormat  string   `json:"input_format"`
				OutputFormat string   `json:"output_format"`
				SampleInput  string   `json:"sample_input"`
				SampleOutput string   `json:"sample_output"`
				Difficulty   string   `json:"difficulty"`
				TimeLimit    int      `json:"time_limit"`   // 毫秒
				MemoryLimit  int      `json:"memory_limit"` // MB
				Languages    []string `json:"languages"`
				Tags         []string `json:"tags"`
				IsPublic     bool     `json:"is_public"`
				IsDeleted    bool     `json:"is_deleted"`
				CreatedBy    int64    `json:"created_by"`
				CreatedAt    string   `json:"created_at"`
				UpdatedAt    string   `json:"updated_at"`
			} `json:"problem"`
			RequestedAt string `json:"requested_at"`
		} `json:"data"`
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
	problemData := response.Data.Problem
	if problemData.Id != problemId {
		return nil, fmt.Errorf("题目ID不匹配: expected=%d, actual=%d", problemId, problemData.Id)
	}

	if problemData.Title == "" {
		return nil, fmt.Errorf("题目标题为空")
	}

	if problemData.TimeLimit <= 0 || problemData.TimeLimit > 30000 {
		return nil, fmt.Errorf("时间限制不合理: %dms", problemData.TimeLimit)
	}

	if problemData.MemoryLimit <= 0 || problemData.MemoryLimit > 1024 {
		return nil, fmt.Errorf("内存限制不合理: %dMB", problemData.MemoryLimit)
	}

	if len(problemData.Languages) == 0 {
		return nil, fmt.Errorf("支持的编程语言列表为空")
	}

	// 检查题目是否已删除
	if problemData.IsDeleted {
		logx.WithContext(ctx).Infof("题目已删除，但继续处理: ProblemId=%d", problemId)
	}

	// 12. 获取测试用例（从新的内部接口获取）
	testCases, err := c.getTestCases(ctx, problemId)
	if err != nil {
		logx.WithContext(ctx).Errorf("获取测试用例失败: %v", err)
		// 如果获取测试用例失败，使用样例数据作为测试用例
		testCases = c.createSampleTestCase(problemData.SampleInput, problemData.SampleOutput)
	}

	// 13. 转换为判题服务内部结构
	problemInfo := &types.ProblemInfo{
		ProblemId:   problemData.Id,
		Title:       problemData.Title,
		TimeLimit:   problemData.TimeLimit,
		MemoryLimit: problemData.MemoryLimit,
		Languages:   problemData.Languages,
		TestCases:   testCases,
		IsPublic:    problemData.IsPublic, // 现在可以从内部接口获取真实状态
	}

	// 14. 记录成功日志
	logx.WithContext(ctx).Infof("Successfully fetched problem %d: %s, time_limit=%dms, memory_limit=%dMB, test_cases=%d",
		problemId, problemInfo.Title, problemInfo.TimeLimit, problemInfo.MemoryLimit, len(problemInfo.TestCases))

	return problemInfo, nil
}

// 获取测试用例（从题目服务的内部接口获取）
func (c *HttpProblemClient) getTestCases(ctx context.Context, problemId int64) ([]types.TestCase, error) {
	// 调用题目服务的内部测试用例接口
	url := fmt.Sprintf("%s/internal/v1/problems/%d/test-cases?include_hidden=true", c.baseURL, problemId)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建测试用例请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "judge-api/1.0.0")
	req.Header.Set("X-Internal-API-Key", "internal-service-secret-key-2024") // 内部API密钥

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求测试用例失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// 如果测试用例接口不存在，返回空切片（后面会使用样例数据）
		return nil, fmt.Errorf("测试用例接口返回状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取测试用例响应失败: %w", err)
	}

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			ProblemId     int64 `json:"problem_id"`
			TotalCount    int   `json:"total_count"`
			IncludeHidden bool  `json:"include_hidden"`
			TestCases     []struct {
				Id             int64  `json:"id"`
				InputData      string `json:"input_data"`
				ExpectedOutput string `json:"expected_output"`
				IsSample       bool   `json:"is_sample"`
				Score          int    `json:"score"`
				SortOrder      int    `json:"sort_order"`
			} `json:"test_cases"`
			RequestedAt string `json:"requested_at"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析测试用例响应失败: %w", err)
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("测试用例业务错误[%d]: %s", response.Code, response.Message)
	}

	// 转换为内部结构
	testCases := make([]types.TestCase, len(response.Data.TestCases))
	for i, tc := range response.Data.TestCases {
		testCases[i] = types.TestCase{
			CaseId:         int(tc.Id),         // 转换为int类型
			Input:          tc.InputData,       // 字段名更新
			ExpectedOutput: tc.ExpectedOutput,
			TimeLimit:      0, // 使用全局时间限制
			MemoryLimit:    0, // 使用全局内存限制
		}
	}

	return testCases, nil
}

// 从样例输入输出创建测试用例（备用方案）
func (c *HttpProblemClient) createSampleTestCase(sampleInput, sampleOutput string) []types.TestCase {
	if sampleInput == "" || sampleOutput == "" {
		// 如果没有样例，创建一个默认的测试用例
		return []types.TestCase{
			{
				CaseId:         1,
				Input:          "1 1",
				ExpectedOutput: "2",
			},
		}
	}

	return []types.TestCase{
		{
			CaseId:         1,
			Input:          sampleInput,
			ExpectedOutput: sampleOutput,
		},
	}
}
